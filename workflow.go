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
	"fmt"
	"path"
	"strings"
	"sync"

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
)

type Workflow struct {
	// protects for api.Workflow and snapshot
	sync.RWMutex
	w        *api.Workflow
	snapshot *api.WorkflowSnapshot

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
		committed: []*api.WorkflowStep{},
		ctx:       ctx,
		cancel:    cancel,
	}
	return w
}

func (w *Workflow) ID() string {
	return w.w.Option.Wid
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
		return nil, fmt.Errorf("workflow not found")
	}

	kv := rsp.Kvs[0]

	var wf *api.Workflow
	if err = json.Unmarshal(kv.Value, &wf); err != nil {
		return nil, err
	}

	rsp, err = client.Get(ctx, w.statusPath())
	if err != nil {
		return nil, err
	}

	value := rsp.Kvs[0].Value
	if err = json.Unmarshal(value, &wf.Status); err != nil {
		return nil, err
	}

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithMinCreateRev(kv.CreateRevision),
	}

	rsp, err = client.Get(ctx, path.Join(w.statusPath(), "entity"), options...)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	wf.Items = make(map[string][]byte)
	for i := range rsp.Kvs {
		kv := rsp.Kvs[i]
		wf.Items[string(kv.Key)] = kv.Value
	}

	rsp, err = client.Get(ctx, path.Join(w.statusPath(), "step"), options...)
	if err != nil {
		return nil, err
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

func (w *Workflow) Cancel() {
	w.cancel()
}

func (w *Workflow) rootPath() string {
	return path.Join(WorkflowPath, w.ID())
}

func (w *Workflow) statusPath() string {
	return path.Join(WorkflowPath, w.ID(), "status")
}

func (w *Workflow) entityPath(entity *api.Entity) string {
	return path.Join(WorkflowPath, w.ID(), "entity", entity.Kind)
}

func (w *Workflow) stepPath(step *api.WorkflowStep) string {
	return path.Join(WorkflowPath, w.ID(), "step", step.Uid)
}

func (w *Workflow) stepItemPath() string {
	return path.Join(WorkflowPath, w.ID(), "item")
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
	return e
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

		key := path.Join(w.stepPath(step), step.Uid)
		if err = w.put(w.ctx, client, key, step); err != nil {
			return
		}
	}

	return w.put(w.ctx, client, w.rootPath(), w.w)
}

func (w *Workflow) forward(ctx context.Context, client *clientv3.Client, action api.StepAction, step *api.WorkflowStep) {

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

	_ = w.put(ctx, client, w.statusPath(), wf.Status)

	if step != nil {
		_ = w.put(ctx, client, w.stepPath(step), step)
	}
}

func (w *Workflow) doClean(client *clientv3.Client, doErr, doneErr error) {
	w.Lock()
	wf := w.w

	if doErr == nil && doneErr == nil {
		wf.Status.State = api.WorkflowState_SW_SUCCESS
		w.snapshot.State = api.WorkflowState_SW_SUCCESS
	} else if doErr != nil {
		wf.Status.State = api.WorkflowState_SW_FAILED
		w.snapshot.State = api.WorkflowState_SW_FAILED
	} else {
		wf.Status.State = api.WorkflowState_SW_WARN
		w.snapshot.State = api.WorkflowState_SW_WARN
	}

	defer w.Unlock()

	_ = w.put(w.ctx, client, w.statusPath(), wf.Status)
}

func (w *Workflow) doStep(ctx context.Context, ps *PipeSet, client *clientv3.Client, step *api.WorkflowStep, action api.StepAction) (err error) {

	sname := step.Name
	sid := step.Uid
	w.forward(ctx, client, action, step)

	cid := step.Client
	pipe, ok := ps.Get(cid)
	if !ok {
		return fmt.Errorf("pipe %s not found", cid)
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
	if err == nil {
		for i := range rsp.Kvs {
			kv := rsp.Kvs[i]
			key := strings.TrimPrefix(string(kv.Key), w.stepItemPath()+"/")
			chunk.Items[key] = kv.Value
		}
	}

	rsp, err = client.Get(ctx, w.entityPath(&api.Entity{Kind: step.Entity}), options...)
	if err != nil {
		return err
	}
	var entity api.Entity
	err = json.Unmarshal(rsp.Kvs[0].Value, &entity)
	if err != nil {
		return err
	}
	chunk.Entity = entity.Raw

	rch, ech := pipe.Step(NewStep(ctx, chunk))

	select {
	case <-ctx.Done():
		return context.Canceled
	case err = <-ech:
		return err
	case b := <-rch:
		entity.Raw = b

		if entity.Kind != "" {
			_ = w.put(ctx, client, w.entityPath(&entity), entity)
		}
	}

	return
}

func (w *Workflow) doPrepare(ps *PipeSet, client *clientv3.Client) error {

	ctx := w.ctx
	steps := w.w.Steps
	action := api.StepAction_SC_PREPARE

	for i := range steps {
		step := steps[i]
		err := w.doStep(ctx, ps, client, step, action)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Workflow) doCommit(ps *PipeSet, client *clientv3.Client) error {

	ctx := w.ctx
	steps := w.w.Steps
	state := api.StepAction_SC_COMMIT
	for i := range steps {
		step := steps[i]

		err := w.doStep(ctx, ps, client, step, state)
		w.committed = append(w.committed, step)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Workflow) doRollback(ps *PipeSet, client *clientv3.Client) error {

	ctx := w.ctx
	state := api.StepAction_SC_ROLLBACK
	errs := make([]error, 0)
	for i := len(w.committed) - 1; i >= 0; i-- {
		step := w.committed[i]
		sname := step.Name + "_" + step.Uid
		e := w.doStep(ctx, ps, client, step, state)
		if e != nil {
			errs = append(errs, fmt.Errorf("step %s rollback: %w", sname, e))
		}
	}

	return is.MargeErr(errs...)
}

func (w *Workflow) doCancel(ps *PipeSet, client *clientv3.Client) error {

	ctx := w.ctx
	state := api.StepAction_SC_CANCEL
	errs := make([]error, 0)
	for i := len(w.committed) - 1; i >= 0; i-- {
		step := w.committed[i]
		sname := step.Name + "_" + step.Uid
		e := w.doStep(ctx, ps, client, step, state)
		if e != nil {
			errs = append(errs, fmt.Errorf("step %s cancel: %w", sname, e))
		}
	}

	return is.MargeErr(errs...)
}

func (w *Workflow) Execute(ps *PipeSet, client *clientv3.Client) {

	log.Debugf("workflow %s prepare", w.ID())

	var doErr, doneErr error

	defer func() {
		w.doClean(client, doErr, doneErr)
	}()

	if doErr = w.doPrepare(ps, client); doErr != nil {
		log.Errorf("workflow %s prepare failed: %v", w.ID(), doErr)
		return
	}

	log.Debugf("workflow %s commit", w.ID())
	if doErr = w.doCommit(ps, client); doErr != nil {
		log.Errorf("workflow %s commit failed: %v", w.ID(), doErr)
	}

	if doErr != nil {
		log.Debugf("workflow %s rollback", w.ID())
		doneErr = w.doRollback(ps, client)
		if doneErr != nil {
			log.Errorf("workflow %s rollback failed: %v", w.ID(), doneErr)
		}
	}

	log.Debugf("workflow %s cancel", w.ID())
	if doneErr = w.doCancel(ps, client); doneErr != nil {
		log.Errorf("workflow %s cancel failed: %v", w.ID(), doneErr)
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
					if strings.HasPrefix(eKey, path.Join(root, "entity")) {
						eType = api.EventType_ET_ENTITY
					} else if strings.HasPrefix(eKey, path.Join(root, "item")) {
						eType = api.EventType_ET_ITEM
					} else if strings.HasPrefix(eKey, path.Join(root, "step")) {
						eType = api.EventType_ET_STEP
					} else {
						eType = api.EventType_ET_WORKFLOW
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

func (s *Scheduler) Register(entities []*api.Entity, echoes []*api.Echo, steps []*api.Step) {
	s.smu.Lock()
	defer s.smu.Unlock()

	for i := range entities {
		entity := entities[i]
		e, ok := s.entitySet.Get(entity.Kind)
		if !ok {
			s.entitySet.Add(entity)
		} else {
			for k, v := range entity.Clients {
				e.Clients[k] = v
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
			for k, v := range echo.Clients {
				e.Clients[k] = v
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
			for k, v := range step.Clients {
				vv.Clients[k] = v
			}
			s.stepSet.Add(vv)
		}
	}
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
		return nil, fmt.Errorf("workflow not found")
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
