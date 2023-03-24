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
	"sync/atomic"

	"github.com/google/uuid"
	json "github.com/json-iterator/go"
	"github.com/panjf2000/ants/v2"
	"github.com/vine-io/flow/api"
	log "github.com/vine-io/vine/lib/logger"
	"github.com/vine-io/vine/util/is"
	"go.etcd.io/etcd/client/v3"
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

	cond  *sync.Cond
	pause atomic.Bool

	abort chan struct{}

	err       error
	action    api.StepAction
	stepIdx   int
	cIdx      int
	committed []*api.WorkflowStep

	ctx    context.Context
	cancel context.CancelFunc
}

func NewWorkflow(spec *api.Workflow) *Workflow {
	ctx, cancel := context.WithCancel(context.Background())

	if spec.Status == nil {
		spec.Status = &api.WorkflowStatus{}
	}

	w := &Workflow{
		w: spec,
		snapshot: &api.WorkflowSnapshot{
			Name: spec.Option.Name,
			Wid:  spec.Option.Wid,
		},
		cond:      sync.NewCond(&sync.Mutex{}),
		pause:     atomic.Bool{},
		abort:     make(chan struct{}, 1),
		committed: []*api.WorkflowStep{},
		ctx:       ctx,
		cancel:    cancel,
	}
	w.pause.Store(false)

	return w
}

func (w *Workflow) Init(client *clientv3.Client) (err error) {
	for i := range w.w.Entities {
		entity := w.w.Entities[i]
		key := w.entityPath(entity)
		if len(entity.Raw) == 0 {
			entity.Raw = []byte("{}")
		}
		if err = w.put(w.ctx, client, key, entity); err != nil {
			return
		}
	}

	for k, v := range w.w.Items {
		key := path.Join(w.stepItemPath(), k)
		if err = w.put(w.ctx, client, key, v); err != nil {
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

		key := w.stepPath(step)
		if err = w.put(w.ctx, client, key, step); err != nil {
			return
		}
	}

	return w.put(w.ctx, client, w.rootPath(), w.w)
}

func (w *Workflow) ID() string {
	return w.w.Option.Wid
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

func (w *Workflow) Inspect(ctx context.Context, client *clientv3.Client) (*api.Workflow, error) {
	root := w.rootPath()
	rsp, err := client.Get(ctx, root)
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

	rsp, err = client.Get(ctx, w.statusPath())
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

	rsp, err = client.Get(ctx, path.Join(w.rootPath(), "store", "entity"), options...)
	if err != nil {
		return nil, api.ErrInsufficientStorage("data from etcd: %v", err)
	}

	wf.Entities = make([]*api.Entity, 0)
	for _, kv := range rsp.Kvs {
		var entity api.Entity
		if e := json.Unmarshal(kv.Value, &entity); e == nil {
			wf.Entities = append(wf.Entities, &entity)
		}
	}

	rsp, err = client.Get(ctx, w.stepItemPath(), options...)
	if err != nil {
		return nil, api.ErrInsufficientStorage("data from etcd: %v", err)
	}

	wf.Items = make(map[string][]byte)
	for i := range rsp.Kvs {
		kv := rsp.Kvs[i]
		wf.Items[string(kv.Key)] = kv.Value
	}

	rsp, err = client.Get(ctx, path.Join(w.rootPath(), "store", "step"), options...)
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
	if w.pause.CompareAndSwap(false, true) {
		w.cond.L.Lock()
		w.cond.Signal()
		w.cond.L.Unlock()
		return true
	}

	return false
}

func (w *Workflow) Resume() bool {
	if w.pause.CompareAndSwap(true, false) {
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
	return path.Join(WorkflowPath, w.ID())
}

func (w *Workflow) statusPath() string {
	return path.Join(WorkflowPath, w.ID(), "store", "status")
}

func (w *Workflow) entityPath(entity *api.Entity) string {
	return path.Join(WorkflowPath, w.ID(), "store", "entity", entity.Kind)
}

func (w *Workflow) stepPath(step *api.WorkflowStep) string {
	return path.Join(WorkflowPath, w.ID(), "store", "step", step.Uid)
}

func (w *Workflow) stepItemPath() string {
	return path.Join(WorkflowPath, w.ID(), "store", "item")
}

func (w *Workflow) put(ctx context.Context, client *clientv3.Client, key string, data any) error {
	var b []byte
	switch tt := data.(type) {
	case []byte:
		b = tt
	case string:
		b = []byte(tt)
	default:
		b, _ = json.Marshal(data)
	}
	_, e := client.Put(ctx, key, string(b))
	if e != nil {
		return api.ErrInsufficientStorage("save data to etcd: %v", e)
	}
	return nil
}

func (w *Workflow) del(ctx context.Context, client *clientv3.Client, key string, prefix bool) error {
	options := []clientv3.OpOption{}
	if prefix {
		options = append(options, clientv3.WithPrefix())
	}
	_, err := client.Delete(ctx, key, options...)
	if err != nil {
		return api.ErrInsufficientStorage("save data to etcd: %v", err)
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

func (w *Workflow) clock(ctx context.Context, client *clientv3.Client, action api.StepAction, step *api.WorkflowStep) {

	var sname string
	var progress string

	if step != nil {
		sname = step.Name + "_" + step.Uid
		if action == api.StepAction_SC_COMMIT {
			progress = w.calProgress(sname)
		}
	}

	w.Lock()
	wf := w.w
	wf.Status.Action = action
	w.snapshot.Action = action

	if step != nil {
		wf.Status.Progress = progress
		wf.Status.Step = sname
		w.snapshot.Step = sname
	}

	switch action {
	case api.StepAction_SC_PREPARE, api.StepAction_SC_COMMIT:
		wf.Status.State = api.WorkflowState_SW_RUNNING
		w.snapshot.State = api.WorkflowState_SW_RUNNING
	}

	defer w.Unlock()

	err := w.put(ctx, client, w.statusPath(), wf.Status)
	if err != nil {
		log.Warnf("update workflow %s status: %v", w.ID(), err)
	}
	if step != nil {
		err = w.put(ctx, client, w.stepPath(step), step)
		if err != nil {
			log.Warnf("update workflow %s step: %v", w.ID(), err)
		}
	}
}

func (w *Workflow) doClean(client *clientv3.Client, doErr, doneErr error) error {

	wf, err := w.Inspect(w.ctx, client)
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

	err = w.put(w.ctx, client, w.rootPath(), wf)
	if err != nil {
		return api.ErrInsufficientStorage("save data to etcd: %v", err)
	}
	err = w.del(w.ctx, client, path.Join(w.rootPath(), "store"), true)
	if err != nil {
		return api.ErrInsufficientStorage("save data to etcd: %v", err)
	}

	return err
}

func (w *Workflow) doStep(ctx context.Context, ps *PipeSet, client *clientv3.Client, step *api.WorkflowStep, action api.StepAction) (err error) {

	sname := step.Name
	sid := step.Uid
	w.clock(ctx, client, action, step)

	workerId := step.Worker
	pipe, ok := ps.Get(workerId)
	if !ok {
		return api.ErrClientException("pipe %s down or not found", workerId)
	}

	chunk := &api.PipeStepRequest{
		Wid:    w.ID(),
		Name:   sname,
		Sid:    sid,
		Action: action,
	}

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	chunk.Items = make(map[string][]byte)
	rsp, err := client.Get(ctx, w.stepItemPath(), options...)
	if err != nil {
		return api.ErrInsufficientStorage("data from etcd: %v", err)
	}

	for i := range rsp.Kvs {
		kv := rsp.Kvs[i]
		key := strings.TrimPrefix(string(kv.Key), w.stepItemPath()+"/")
		chunk.Items[key] = kv.Value
	}

	rsp, err = client.Get(ctx, w.entityPath(&api.Entity{Kind: step.Entity}), options...)
	if err != nil {
		return api.ErrInsufficientStorage("data from etcd: %v", err)
	}
	var entity api.Entity
	err = json.Unmarshal(rsp.Kvs[0].Value, &entity)
	if err != nil {
		return api.ErrInsufficientStorage("data from etcd: %v", err)
	}
	chunk.Entity = entity.Raw

	rch, ech := pipe.Step(NewStep(ctx, chunk))

	select {
	case <-ctx.Done():
		err = api.ErrCancel("workflow do step: %s", sname)
	case err = <-ech:
	case b := <-rch:
		entity.Raw = b

		if entity.Kind != "" {
			if e := w.put(ctx, client, w.entityPath(&entity), entity); e != nil {
				log.Warnf("update workflow %s entity: %v", w.ID(), e)
			}
		}
	}

	return
}

func (w *Workflow) prev() (*api.WorkflowStep, api.StepAction, bool) {
	steps := w.w.Steps
	stepLen := len(steps)
	switch w.action {
	case api.StepAction_SA_UNKNOWN:
		return nil, w.action, false
	case api.StepAction_SC_PREPARE:
		if w.stepIdx == 0 {
			return nil, w.action, false
		}
		w.stepIdx -= 1
		step := steps[w.stepIdx]
		return step, w.action, true
	case api.StepAction_SC_COMMIT:
		if w.stepIdx == 0 {
			w.action = api.StepAction_SC_PREPARE
			w.stepIdx = stepLen
			return w.prev()
		}
		w.stepIdx -= 1
		step := steps[w.stepIdx]
		return step, w.action, true
	case api.StepAction_SC_ROLLBACK:
		if w.cIdx == 0 {
			w.action = api.StepAction_SC_COMMIT
			w.stepIdx = stepLen
			return w.prev()
		}
		rlen := len(w.committed)
		w.cIdx -= 1
		idx := rlen - w.cIdx - 1
		step := w.committed[idx]
		return step, w.action, true
	case api.StepAction_SC_CANCEL:
		if w.cIdx == 0 {
			if w.err != nil {
				w.action = api.StepAction_SC_ROLLBACK
				w.cIdx = len(w.committed)
				return w.prev()
			}
			w.action = api.StepAction_SC_COMMIT
			w.stepIdx = stepLen
			return w.prev()
		}
		rlen := len(w.committed)
		w.cIdx -= 1
		idx := rlen - w.cIdx - 1
		step := w.committed[idx]
		return step, w.action, true
	default:
		return nil, w.action, false
	}
}

func (w *Workflow) next() (*api.WorkflowStep, api.StepAction, bool) {
	steps := w.w.Steps
	stepLen := len(steps)
	switch w.action {
	case api.StepAction_SA_UNKNOWN:
		w.action = api.StepAction_SC_PREPARE
		return w.next()
	case api.StepAction_SC_PREPARE:
		if w.err != nil {
			return nil, w.action, false
		}
		if w.stepIdx >= stepLen {
			w.action = api.StepAction_SC_COMMIT
			w.stepIdx = 0
			return w.next()
		}
		step := steps[w.stepIdx]
		w.stepIdx += 1
		return step, w.action, true
	case api.StepAction_SC_COMMIT:
		if w.err != nil {
			w.action = api.StepAction_SC_ROLLBACK
			w.cIdx = 0
			return w.next()
		}
		if w.stepIdx >= stepLen {
			w.action = api.StepAction_SC_CANCEL
			w.cIdx = 0
			w.stepIdx = 0
			return w.next()
		}
		step := steps[w.stepIdx]
		w.stepIdx += 1
		return step, w.action, true
	case api.StepAction_SC_ROLLBACK:
		rlen := len(w.committed)
		if w.cIdx >= rlen {
			w.action = api.StepAction_SC_CANCEL
			w.cIdx = 0
			w.stepIdx = 0
			return w.next()
		}
		idx := rlen - w.cIdx - 1
		step := w.committed[idx]
		w.cIdx += 1
		return step, w.action, true
	case api.StepAction_SC_CANCEL:
		rlen := len(w.committed)
		if w.cIdx >= rlen {
			return nil, w.action, false
		}
		idx := rlen - w.cIdx - 1
		step := w.committed[idx]
		w.cIdx += 1
		return step, w.action, true
	default:
		return nil, w.action, false
	}
}

func (w *Workflow) through(client *clientv3.Client) {
	for w.pause.Load() {

		w.Lock()
		wf := w.w
		wf.Status.State = api.WorkflowState_SW_PAUSE
		w.snapshot.State = api.WorkflowState_SW_PAUSE
		w.Unlock()

		_ = w.put(w.ctx, client, w.statusPath(), wf.Status)

		w.cond.L.Lock()
		w.cond.Wait()
		w.cond.L.Unlock()
	}
}

func (w *Workflow) Execute(ps *PipeSet, client *clientv3.Client) {

	errs := make([]error, 0)
	for {
		// waiting for pause become false
		w.through(client)

		if w.IsAbort() {
			// workflow be aborted
			w.err = ErrAborted
		}

		step, action, doNext := w.next()
		if !doNext {
			break
		}

		sname := step.Name + "_" + step.Uid
		log.Infof("workflow %s [%s] do step %s", w.ID(), action.Readably(), sname)

		err := w.doStep(w.ctx, ps, client, step, action)

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

	doErr, doneErr := w.err, is.MargeErr(errs...)
	log.Debugf("delete workflow %s runtime data", w.ID())
	if e := w.doClean(client, doErr, doneErr); e != nil {
		log.Errorf("clean workflow %s data: %v", w.ID(), e)
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
				result := &api.WorkflowWatchResult{
					Name: wf.Option.Name,
					Wid:  wf.Option.Wid,
					Type: api.EventType_ET_RESULT,
				}

				out <- result
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
					} else if prefix = path.Join(root, "store", "status"); strings.HasPrefix(eKey, prefix) {
						eType = api.EventType_ET_STATUS
						eKey = strings.TrimPrefix(eKey, path.Join(root, "store")+"/")
					} else {
						eType = api.EventType_ET_WORKFLOW
						eKey = strings.TrimPrefix(eKey, path.Join(Root, "wf")+"/")
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
				}
			}
		}
	}(ch)

	return ch, nil
}

type Scheduler struct {
	wg   sync.WaitGroup
	pool *ants.Pool

	storage *clientv3.Client

	smu       sync.RWMutex
	entitySet *EntitySet
	echoSet   *EchoSet
	stepSet   *StepSet

	wmu sync.RWMutex
	wfm map[string]*Workflow

	exit chan struct{}
}

func NewScheduler(storage *clientv3.Client, size int) (*Scheduler, error) {
	pool, err := ants.NewPool(size)
	if err != nil {
		return nil, err
	}

	s := &Scheduler{
		pool:      pool,
		storage:   storage,
		entitySet: NewEntitySet(),
		echoSet:   NewEchoSet(),
		stepSet:   NewStepSet(),
		wfm:       map[string]*Workflow{},
		exit:      make(chan struct{}, 1),
	}

	return s, nil
}

func (s *Scheduler) Register(worker string, entities []*api.Entity, echoes []*api.Echo, steps []*api.Step) error {

	ctx := context.Background()
	key := path.Join(Root, "worker", worker)
	_, err := s.storage.Put(ctx, key, "")
	if err != nil {
		return err
	}

	s.smu.Lock()
	defer s.smu.Unlock()

	for i := range entities {
		entity := entities[i]
		e, ok := s.entitySet.Get(entity.Kind)
		if !ok {
			s.entitySet.Add(entity)
		} else {
			for k, v := range entity.Workers {
				e.Workers[k] = v
			}
			s.entitySet.Add(e)
		}
	}
	for i := range echoes {
		echo := echoes[i]
		e, ok := s.echoSet.Get(echo.Name)
		if !ok {
			s.echoSet.Add(echo)
		} else {
			for k, v := range echo.Workers {
				e.Workers[k] = v
			}
			s.echoSet.Add(e)
		}
	}
	for i := range steps {
		step := steps[i]
		vv, ok := s.stepSet.Get(step.Name)
		if !ok {
			s.stepSet.Add(step)
		} else {
			for k, v := range step.Workers {
				vv.Workers[k] = v
			}
			s.stepSet.Add(vv)
		}
	}

	return nil
}

func (s *Scheduler) GetRegistry() (entities []*api.Entity, echoes []*api.Echo, steps []*api.Step) {
	entities = s.entitySet.List()
	echoes = s.echoSet.List()
	steps = s.stepSet.List()
	return
}

func (s *Scheduler) GetWorkers(ctx context.Context) ([]string, error) {
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithKeysOnly(),
	}

	key := path.Join(Root, "worker")
	rsp, err := s.storage.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}

	workers := make([]string, len(rsp.Kvs))
	for i, kv := range rsp.Kvs {
		workers[i] = strings.TrimPrefix(key+"/", string(kv.Key))
	}

	return workers, nil
}

func (s *Scheduler) GetWorkflow(wid string) (*Workflow, bool) {
	s.wmu.RLock()
	defer s.wmu.RUnlock()
	w, ok := s.wfm[wid]
	return w, ok
}

func (s *Scheduler) InspectWorkflow(ctx context.Context, wid string) (*api.Workflow, error) {
	w, ok := s.GetWorkflow(wid)
	if !ok {
		rsp, err := s.storage.Get(ctx, path.Join(Root, wid))
		if err != nil {
			return nil, err
		}

		if len(rsp.Kvs) == 0 {
			return nil, fmt.Errorf("workflow not found")
		}

		var wf api.Workflow
		err = json.Unmarshal(rsp.Kvs[0].Value, &wf)
		if err != nil {
			return nil, fmt.Errorf("bad data")
		}

		return &wf, nil
	}

	data, err := w.Inspect(ctx, s.storage)
	return data, err
}

func (s *Scheduler) WorkflowSnapshots() []*api.WorkflowSnapshot {
	s.wmu.RLock()
	defer s.wmu.RUnlock()
	snapshots := make([]*api.WorkflowSnapshot, 0)
	for _, w := range s.wfm {
		snapshots = append(snapshots, w.NewSnapshot())
	}

	return snapshots
}

func (s *Scheduler) WatchWorkflow(ctx context.Context, wid string) (<-chan *api.WorkflowWatchResult, error) {
	w, ok := s.GetWorkflow(wid)
	if !ok {
		return nil, fmt.Errorf("workflow not found")
	}

	return w.NewWatcher(ctx, s.storage)
}

func (s *Scheduler) StepPut(ctx context.Context, wid, key, value string) error {
	w, ok := s.GetWorkflow(wid)
	if !ok {
		return fmt.Errorf("workflow not found")
	}

	key = path.Join(w.stepItemPath(), key)

	_, err := s.storage.Put(ctx, key, value)
	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) StepGet(ctx context.Context, wid, key string) ([]byte, error) {
	w, ok := s.GetWorkflow(wid)
	if !ok {
		return nil, fmt.Errorf("workflow not found")
	}

	key = path.Join(w.stepItemPath(), key)

	rsp, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, fmt.Errorf("key not found")
	}

	return rsp.Kvs[0].Value, nil
}

func (s *Scheduler) StepTrace(ctx context.Context, wid, step string, text []byte) error {
	_, ok := s.GetWorkflow(wid)
	if !ok {
		return fmt.Errorf("workflow not found")
	}

	log.Trace("step %s trace: %s", step, string(text))
	// TODO: trace log record

	return nil
}

func (s *Scheduler) ExecuteWorkflow(w *api.Workflow, ps *PipeSet) error {
	if s.IsClosed() {
		return fmt.Errorf("scheduler stopped")
	}

	s.smu.RLock()
	for _, entity := range w.Entities {
		if !s.entitySet.Contains(entity.Kind) {
			s.smu.RUnlock()
			return fmt.Errorf("unknwon entity: kind=%s", entity.Kind)
		}
	}

	for _, step := range w.Steps {
		if !s.stepSet.Contains(step.Name) {
			s.smu.RUnlock()
			return fmt.Errorf("unknown step: name=%s", step.Name)
		}
		if _, ok := ps.Get(step.Worker); !ok {
			s.smu.RUnlock()
			return fmt.Errorf("worker %s not found", step.Worker)
		}
	}
	s.smu.RUnlock()

	wf := NewWorkflow(w)

	if _, ok := s.GetWorkflow(wf.ID()); ok {
		return fmt.Errorf("workflow already exists")
	}

	if err := wf.Init(s.storage); err != nil {
		return fmt.Errorf("initialize workflow %s failed: %v", wf.ID(), err)
	}

	s.wmu.Lock()
	s.wfm[wf.ID()] = wf
	s.wmu.Unlock()

	s.wg.Add(1)
	err := s.pool.Submit(func() {
		defer s.wg.Done()

		defer wf.Cancel()

		defer func() {
			s.wmu.Lock()
			delete(s.wfm, wf.ID())
			s.wmu.Unlock()
		}()

		wf.Execute(ps, s.storage)
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) Stop(wait bool) {
	if wait {
		s.wg.Wait()
	}
	if s.IsClosed() {
		return
	}
	s.pool.Release()
	close(s.exit)
}

func (s *Scheduler) IsClosed() bool {
	select {
	case <-s.exit:
		return true
	default:
		return false
	}
}
