// MIT License
//
// Copyright (c) 2023 Lack
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package flow

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	json "github.com/json-iterator/go"
	"github.com/vine-io/flow/api"
	log "github.com/vine-io/vine/lib/logger"
	"github.com/vine-io/vine/util/is"
	"go.etcd.io/etcd/client/v3"
	"go.uber.org/atomic"
)

var (
	Root         = "/vine.io/flow"
	WorkflowPath = path.Join(Root, "wf")
	ErrAborted   = errors.New("be aborted")
)

type Workflow struct {
	// protects for api.Workflow and snapshot
	sync.RWMutex
	w        *api.Workflow
	snapshot *api.WorkflowSnapshot

	storage *clientv3.Client
	ps      *PipeSet

	cond  *sync.Cond
	pause atomic.Bool

	interactiveCh chan *api.Interactive

	abort chan struct{}

	err       error
	cleanErrs []error
	action    api.StepAction
	stepIdx   int
	cIdx      int
	committed []*api.WorkflowStep

	ctx    context.Context
	cancel context.CancelFunc
}

func NewWorkflow(id, instanceId, name string, items map[string]string, storage *clientv3.Client, ps *PipeSet) *Workflow {
	ctx, cancel := context.WithCancel(context.Background())

	spec := &api.Workflow{
		Option: &api.WorkflowOption{
			Name:       name,
			Wid:        id,
			InstanceId: instanceId,
			MaxRetries: 3,
		},
		Items: items,
	}

	spec.Status = &api.WorkflowStatus{Option: spec.Option}

	w := &Workflow{
		w: spec,
		snapshot: &api.WorkflowSnapshot{
			Name:       spec.Option.Name,
			Wid:        spec.Option.Wid,
			InstanceId: instanceId,
		},
		storage:       storage,
		ps:            ps,
		cond:          sync.NewCond(&sync.Mutex{}),
		pause:         atomic.Bool{},
		interactiveCh: make(chan *api.Interactive, 1),
		abort:         make(chan struct{}, 1),
		committed:     []*api.WorkflowStep{},
		ctx:           ctx,
		cancel:        cancel,
	}
	w.pause.Store(false)

	return w
}

func (w *Workflow) Init() (err error) {
	for k, v := range w.w.Items {
		key := path.Join(w.stepItemPath(), k)
		if err = w.put(w.ctx, key, v); err != nil {
			return
		}
	}

	for i := range w.w.Steps {
		step := w.w.Steps[i]
		if step.Uid == "" {
			step.Uid = uuid.New().String()
		}
		if step.Injects == nil {
			step.Injects = []string{}
		}
		step.Logs = []string{}
		step.Stages = []*api.WorkflowStepStage{}

		key := w.stepPath(step)
		if err = w.put(w.ctx, key, step); err != nil {
			return
		}
	}

	return w.put(w.ctx, w.rootPath(), w.w)
}

func (w *Workflow) ID() string {
	return w.w.Option.Wid
}

func (w *Workflow) InstanceId() string {
	return w.w.Option.InstanceId
}

func (w *Workflow) maxRetries() int32 {
	return w.w.Option.MaxRetries
}

func (w *Workflow) Context() context.Context {
	return w.ctx
}

func (w *Workflow) NewSnapshot() *api.WorkflowSnapshot {
	w.RLock()
	defer w.RUnlock()
	return w.snapshot.DeepCopy()
}

func (w *Workflow) Inspect(ctx context.Context) (*api.Workflow, error) {
	root := w.rootPath()
	rsp, err := w.storage.Get(ctx, root)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, api.ErrInsufficientStorage("workflow not found")
	}

	var wf *api.Workflow
	if err = json.Unmarshal(rsp.Kvs[0].Value, &wf); err != nil {
		return nil, api.ErrInsufficientStorage("parse data: %v", err)
	}

	rsp, err = w.storage.Get(ctx, w.statusPath())
	if err != nil {
		return nil, api.ErrInsufficientStorage("data from etcd: %v", err)
	}

	value := rsp.Kvs[0].Value
	if err = json.Unmarshal(value, &wf.Status); err != nil {
		return nil, api.ErrInsufficientStorage("parse data: %v", err)
	}

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	rsp, err = w.storage.Get(ctx, path.Join(w.rootPath(), "store", "entity"), options...)
	if err != nil {
		return nil, api.ErrInsufficientStorage("data from etcd: %v", err)
	}

	rsp, err = w.storage.Get(ctx, w.stepItemPath(), options...)
	if err != nil {
		return nil, api.ErrInsufficientStorage("data from etcd: %v", err)
	}

	wf.Items = make(map[string]string)
	for i := range rsp.Kvs {
		kv := rsp.Kvs[i]
		wf.Items[string(kv.Key)] = string(kv.Value)
	}

	rsp, err = w.storage.Get(ctx, path.Join(w.rootPath(), "store", "step"), options...)
	if err != nil {
		return nil, api.ErrInsufficientStorage("data from etcd: %v", err)
	}

	wf.Steps = make([]*api.WorkflowStep, 0)
	for _, kv := range rsp.Kvs {
		var step api.WorkflowStep
		if e := json.Unmarshal(kv.Value, &step); e == nil {
			wf.Steps = append(wf.Steps, &step)
		}
	}

	return wf, nil
}

func (w *Workflow) IsStop() bool {
	select {
	case <-w.ctx.Done():
		return true
	default:
		return false
	}
}

func (w *Workflow) IsAbort() bool {
	select {
	case <-w.abort:
		return true
	default:
		return false
	}
}

func (w *Workflow) Abort() {
	if w.IsAbort() {
		return
	}
	close(w.abort)

	if w.pause.Load() {
		w.Resume()
	}
}

func (w *Workflow) Pause() bool {
	if w.pause.CAS(false, true) {
		w.cond.L.Lock()
		w.cond.Signal()
		w.cond.L.Unlock()
		return true
	}

	return false
}

func (w *Workflow) Resume() bool {
	if w.pause.CAS(true, false) {
		w.cond.L.Lock()
		w.cond.Signal()
		w.cond.L.Unlock()
		return true
	}

	return false
}

func (w *Workflow) Cancel() {
	w.cancel()
}

func (w *Workflow) rootPath() string {
	return path.Join(WorkflowPath, w.ID(), w.InstanceId())
}

func (w *Workflow) statusPath() string {
	return path.Join(WorkflowPath, w.ID(), w.InstanceId(), "store", "status")
}

func (w *Workflow) entityPath(entity *api.Entity) string {
	return path.Join(WorkflowPath, w.ID(), w.InstanceId(), "store", "entity", entity.Kind)
}

func (w *Workflow) stepPath(step *api.WorkflowStep) string {
	return path.Join(WorkflowPath, w.ID(), w.InstanceId(), "store", "step", step.Uid)
}

func (w *Workflow) stepItemPath() string {
	return path.Join(WorkflowPath, w.ID(), w.InstanceId(), "store", "item")
}

func (w *Workflow) stepTracePath() string {
	return path.Join(WorkflowPath, w.ID(), w.InstanceId(), "store", "trace")
}

func (w *Workflow) put(ctx context.Context, key string, data any, opts ...clientv3.OpOption) error {
	var b []byte
	switch tt := data.(type) {
	case []byte:
		b = tt
	case string:
		b = []byte(tt)
	default:
		b, _ = json.Marshal(data)
	}

	_, e := w.storage.Put(ctx, key, string(b), opts...)
	if e != nil {
		return api.ErrInsufficientStorage("save data to etcd: %v", e)
	}
	return nil
}

func (w *Workflow) del(ctx context.Context, key string, prefix bool) error {
	options := []clientv3.OpOption{}
	if prefix {
		options = append(options, clientv3.WithPrefix())
	}
	_, err := w.storage.Delete(ctx, key, options...)
	if err != nil {
		return api.ErrInsufficientStorage("save data to etcd: %v", err)
	}
	return nil
}

func (w *Workflow) trace(ctx context.Context, traceLog *api.TraceLog, opts ...clientv3.OpOption) error {

	key := path.Join(w.stepTracePath(), fmt.Sprintf("%d", traceLog.Timestamp))
	data, _ := json.Marshal(traceLog)
	_, e := w.storage.Put(ctx, key, string(data), opts...)
	if e != nil {
		return api.ErrInsufficientStorage("save data to etcd: %v", e)
	}
	return nil
}

func (w *Workflow) calProgress(stepName string) string {
	w.RLock()
	defer w.RUnlock()

	for i, step := range w.w.Steps {
		sname := step.Name + "_" + step.Uid
		if sname == stepName {
			return fmt.Sprintf("%.2f", float64(i)/float64(len(w.w.Steps))*100)
		}
	}

	return "100.00"
}

func (w *Workflow) clock(ctx context.Context, action api.StepAction, step *api.WorkflowStep) {

	var sname string
	var progress string

	sname = step.Name + "_" + step.Uid
	if action == api.StepAction_SC_COMMIT {
		progress = w.calProgress(sname)
	}

	w.Lock()
	wf := w.w
	wf.Status.Action = action
	w.snapshot.Action = action

	wf.Status.Progress = progress
	wf.Status.Step = sname
	w.snapshot.Step = sname

	switch action {
	case api.StepAction_SC_PREPARE, api.StepAction_SC_COMMIT:
		wf.Status.State = api.WorkflowState_SW_RUNNING
		w.snapshot.State = api.WorkflowState_SW_RUNNING
	}

	defer w.Unlock()

	err := w.put(ctx, w.statusPath(), wf.Status)
	if err != nil {
		log.Warnf("update workflow %s status: %v", w.ID(), err)
	}
}

func (w *Workflow) doClean(doErr, doneErr error) error {

	wf, err := w.Inspect(w.ctx)
	if err != nil {
		return err
	}

	w.Lock()
	if doErr == nil && doneErr == nil {
		wf.Status.State = api.WorkflowState_SW_SUCCESS
		w.snapshot.State = api.WorkflowState_SW_SUCCESS
	} else if doErr != nil {
		wf.Status.State = api.WorkflowState_SW_FAILED
		wf.Status.Msg = doErr.Error()
		w.snapshot.State = api.WorkflowState_SW_FAILED
	} else {
		wf.Status.State = api.WorkflowState_SW_WARN
		wf.Status.Msg = doneErr.Error()
		w.snapshot.State = api.WorkflowState_SW_WARN
	}
	w.w = wf
	defer w.Unlock()

	// keep 3min
	rsp, err := w.storage.Lease.Grant(w.ctx, 180)
	if err != nil {
		return err
	}
	opOpts := []clientv3.OpOption{clientv3.WithLease(rsp.ID)}

	err = w.put(w.ctx, w.rootPath(), wf, opOpts...)
	if err != nil {
		return api.ErrInsufficientStorage("save data to etcd: %v", err)
	}
	err = w.del(w.ctx, path.Join(w.rootPath(), "store"), true)
	if err != nil {
		return api.ErrInsufficientStorage("save data to etcd: %v", err)
	}

	return nil
}

func (w *Workflow) doStep(ctx context.Context, step *api.WorkflowStep, action api.StepAction, items map[string]string) (map[string]string, error) {

	sname := step.Name
	sid := step.Uid
	w.clock(ctx, action, step)

	workerId := step.Worker
	pipe, ok := w.ps.Get(workerId)
	if !ok {
		return nil, api.ErrClientException("pipe %s down or not found", workerId)
	}

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	rsp, err := w.storage.Get(ctx, w.stepItemPath(), options...)
	if err != nil {
		return nil, api.ErrInsufficientStorage("data from etcd: %v", err)
	}

	for i := range rsp.Kvs {
		kv := rsp.Kvs[i]
		key := strings.TrimPrefix(string(kv.Key), w.stepItemPath()+"/")
		items[key] = string(kv.Value)
	}

	chunk := &api.PipeStepRequest{
		Wid:        w.ID(),
		InstanceId: w.InstanceId(),
		Name:       sname,
		Sid:        sid,
		Action:     action,
		Items:      items,
	}

	stage := &api.WorkflowStepStage{
		Action:         action,
		State:          api.WorkflowState_SW_RUNNING,
		StartTimestamp: time.Now().Unix(),
	}
	step.Stages = append(step.Stages, stage)

	defer func() {
		stage.State = api.WorkflowState_SW_SUCCESS
		if err != nil {
			stage.ErrorMsg = err.Error()
			stage.State = api.WorkflowState_SW_FAILED
		}
		stage.EndTimestamp = time.Now().Unix()
		_ = w.put(ctx, w.stepPath(step), step)
	}()

	err = w.put(ctx, w.stepPath(step), step)
	if err != nil {
		log.Warnf("update workflow %s step: %v", w.ID(), err)
	}

	sctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pack := NewStep(sctx, chunk)
	defer pack.Destroy()
	rch, ech := pipe.Step(pack)

	out := map[string]string{}
	select {
	case <-ctx.Done():
		err = api.ErrCancel("workflow do step: %s", sname)
	case err = <-ech:
	case b := <-rch:
		out = b
	}

	return out, err
}

func (w *Workflow) through() {
	for w.pause.Load() {

		w.Lock()
		wf := w.w
		wf.Status.State = api.WorkflowState_SW_PAUSE
		w.snapshot.State = api.WorkflowState_SW_PAUSE
		w.Unlock()

		_ = w.put(w.ctx, w.statusPath(), wf.Status)

		w.cond.L.Lock()
		w.cond.Wait()
		w.cond.L.Unlock()
	}
}

func (w *Workflow) Handle(step *api.WorkflowStep, action api.StepAction, items map[string]string) (map[string]string, error) {

	// waiting for pause become false
	w.through()

	if w.IsAbort() {
		// workflow be aborted
		w.err = ErrAborted
	}

	sname := step.Name + "_" + step.Uid
	log.Infof("[%s] workflow %s do step %s", action.Readably(), w.ID(), sname)

	out, err := w.doStep(w.ctx, step, action, items)
	for k, v := range out {
		key := path.Join(w.stepItemPath(), k)
		if err = w.put(w.ctx, key, v); err != nil {
			return nil, err
		}
		w.w.Items[k] = v
	}

	switch action {
	case api.StepAction_SC_PREPARE:
		if err != nil {
			w.err = err
		}
	case api.StepAction_SC_COMMIT:
		if err != nil {
			w.err = err
		}
		w.committed = append(w.committed, step)
	}

	return out, err
}

func (w *Workflow) Destroy() {
	defer w.Cancel()

	errs := make([]error, 0)
	if w.err != nil {
		errs1 := w.destroy(api.StepAction_SC_ROLLBACK)
		errs = append(errs, errs1...)
	}

	errs1 := w.destroy(api.StepAction_SC_CANCEL)
	errs = append(errs, errs1...)

	if w.err != nil {
		log.Errorf("workflow %s failed: %v", w.ID(), w.err)
	} else {
		log.Infof("workflow %s successful", w.ID())
	}

	doErr, doneErr := w.err, is.MargeErr(errs...)
	log.Debugf("delete workflow %s runtime data", w.ID())
	if e := w.doClean(doErr, doneErr); e != nil {
		log.Errorf("clean workflow %s data: %v", w.ID(), e)
	}
}

func (w *Workflow) fetchStepParam(ctx context.Context, step *api.WorkflowStep) (map[string]string, error) {
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	items := make(map[string]string)
	rsp, err := w.storage.Get(ctx, w.stepItemPath(), options...)
	if err != nil {
		return nil, api.ErrInsufficientStorage("data from etcd: %v", err)
	}

	for i := range rsp.Kvs {
		kv := rsp.Kvs[i]
		key := strings.TrimPrefix(string(kv.Key), w.stepItemPath()+"/")
		items[key] = string(kv.Value)
	}

	return items, nil
}

func (w *Workflow) InteractiveHandle(ctx context.Context, step *api.WorkflowStep, it *api.Interactive) error {
	stage := &api.WorkflowStepStage{
		Action:         api.StepAction_SC_COMMIT,
		State:          api.WorkflowState_SW_SUCCESS,
		StartTimestamp: time.Now().Unix(),
	}

	key := path.Join(Root, "interactive", w.ID(), step.Name)
	_ = w.put(ctx, key, it)
	_ = w.put(ctx, w.stepPath(step), step)
	defer w.del(ctx, key, false)

	select {
	case <-ctx.Done():
		return nil
	case it := <-w.interactiveCh:
		for _, property := range it.Properties {
			k, v := property.Name, property.Value
			key := path.Join(w.stepItemPath(), k)
			if err := w.put(w.ctx, key, v); err != nil {
				return err
			}
		}
	}

	stage.EndTimestamp = time.Now().Unix()
	step.Stages = append(step.Stages, stage)

	err := w.put(ctx, w.stepPath(step), step)
	if err != nil {
		log.Warnf("update workflow %s step: %v", w.ID(), err)
	}

	return nil
}

func (w *Workflow) CommitInteractive(it *api.Interactive) {
	w.interactiveCh <- it
}

func (w *Workflow) destroy(action api.StepAction) (errs []error) {
	ctx := w.ctx
	length := len(w.committed)
	for i := length - 1; i >= 0; i-- {
		// waiting for pause become false
		w.through()

		if w.IsAbort() {
			// workflow be aborted
			w.err = ErrAborted
		}

		step := w.committed[i]
		sname := step.Name + "_" + step.Uid
		log.Infof("[%s] workflow %s do step %s", action.Readably(), w.ID(), sname)

		items, err := w.fetchStepParam(ctx, step)
		if err == nil {
			_, err = w.doStep(w.ctx, step, action, items)
		}

		switch action {
		case api.StepAction_SC_ROLLBACK:
			if err != nil {
				err = fmt.Errorf("step %s rollback: %w", sname, err)
				errs = append(errs, err)
			}
		case api.StepAction_SC_CANCEL:
			if err != nil {
				err = fmt.Errorf("step %s cancel: %w", sname, err)
				errs = append(errs, err)
			}
		}
	}

	return
}

func (w *Workflow) Execute() {
	select {
	case <-w.ctx.Done():
	}
}

func (w *Workflow) NewWatcher(ctx context.Context, client *clientv3.Client) (<-chan *api.WorkflowWatchResult, error) {

	root := w.rootPath()
	wRsp, err := client.Get(ctx, root)
	if err != nil {
		return nil, err
	}
	if len(wRsp.Kvs) == 0 {
		return nil, fmt.Errorf("workflow not found")
	}

	kv := wRsp.Kvs[0]

	var wf *api.Workflow
	if err = json.Unmarshal(kv.Value, &wf); err != nil {
		return nil, err
	}

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	wch := client.Watch(ctx, root, options...)

	ch := make(chan *api.WorkflowWatchResult, 10)

	go func(out chan<- *api.WorkflowWatchResult) {
		defer close(out)

		for {
			select {
			case <-w.ctx.Done():
				return
			case rsp := <-wch:

				for _, e := range rsp.Events {
					var action api.EventAction
					switch e.Type {
					case clientv3.EventTypePut:
						if e.IsCreate() {
							action = api.EventAction_EA_CREATE
						} else if e.IsModify() {
							action = api.EventAction_EA_UPDATE
						}
					case clientv3.EventTypeDelete:
						action = api.EventAction_EA_DELETE
					}

					eKey := string(e.Kv.Key)

					var eType api.EventType
					if prefix := path.Join(root, "store", "entity"); strings.HasPrefix(eKey, prefix) {
						eType = api.EventType_ET_ENTITY
						eKey = strings.TrimPrefix(eKey, prefix+"/")
					} else if prefix = path.Join(root, "store", "item"); strings.HasPrefix(eKey, prefix) {
						eType = api.EventType_ET_ITEM
						eKey = strings.TrimPrefix(eKey, prefix+"/")
					} else if prefix = path.Join(root, "store", "step"); strings.HasPrefix(eKey, prefix) {
						eType = api.EventType_ET_STEP
						eKey = strings.TrimPrefix(eKey, prefix+"/")
					} else if prefix = path.Join(root, "store", "trace"); strings.HasPrefix(eKey, prefix) {
						eType = api.EventType_ET_TRACE
						eKey = strings.TrimPrefix(eKey, path.Join(root, "store", "trace")+"/")
					} else if prefix = path.Join(root, "store", "status"); strings.HasPrefix(eKey, prefix) {
						eType = api.EventType_ET_STATUS
						eKey = strings.TrimPrefix(eKey, path.Join(root, "store")+"/")
					} else {
						eType = api.EventType_ET_WORKFLOW
						eKey = strings.TrimPrefix(eKey, WorkflowPath+"/")
					}

					result := &api.WorkflowWatchResult{
						Name:   wf.Option.Name,
						Wid:    wf.Option.Wid,
						Action: action,
						Type:   eType,
						Key:    eKey,
						Value:  e.Kv.Value,
					}

					out <- result

					if action == api.EventAction_EA_DELETE && eType == api.EventType_ET_STATUS {
						result = &api.WorkflowWatchResult{
							Name: wf.Option.Name,
							Wid:  wf.Option.Wid,
							Type: api.EventType_ET_RESULT,
						}

						out <- result
						return
					}
				}
			}
		}
	}(ch)

	return ch, nil
}
