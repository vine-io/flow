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
	"sync"

	"github.com/google/uuid"
	json "github.com/json-iterator/go"
	"github.com/panjf2000/ants/v2"
	"github.com/vine-io/flow/api"
	log "github.com/vine-io/vine/lib/logger"
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

func (w *Workflow) Cancel() {
	w.cancel()
}

func (w *Workflow) statusPath() string {
	return path.Join(WorkflowPath, w.ID(), "status")
}

func (w *Workflow) entityPath(entity *api.Entity) string {
	return path.Join(WorkflowPath, w.ID(), "entity", entity.Id)
}

func (w *Workflow) stepPath(step *api.WorkflowStep) string {
	return path.Join(WorkflowPath, w.ID(), "step", step.Uid)
}

func (w *Workflow) stepItemPath(step *api.WorkflowStep) string {
	return path.Join(w.stepPath(step), "item")
}

func (w *Workflow) put(ctx context.Context, client *clientv3.Client, key string, data any) error {
	b, _ := json.Marshal(data)
	_, e := client.Put(ctx, key, string(b))
	return e
}

func (w *Workflow) calProgress(stepName string) string {
	w.RLock()
	defer w.RUnlock()

	for i, step := range w.w.Steps {
		sname := step.Name + "-" + step.Uid
		if sname == stepName {
			return fmt.Sprintf("%.2f", float64(i)/float64(len(w.w.Steps))*100)
		}
	}

	return "100.00"
}

func (w *Workflow) forward(ctx context.Context, client *clientv3.Client, action api.StepAction, step *api.WorkflowStep) {

	var sname string
	var progress string

	if step != nil {
		sname = step.Name + "-" + step.Uid
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
	case api.StepAction_SC_ROLLBACK:
		wf.Status.State = api.WorkflowState_SW_ROLLBACK
		w.snapshot.State = api.WorkflowState_SW_ROLLBACK
	case api.StepAction_SC_CANCEL:
		wf.Status.State = api.WorkflowState_SW_CANCEL
		w.snapshot.State = api.WorkflowState_SW_CANCEL
	}

	defer w.Unlock()

	_ = w.put(ctx, client, w.statusPath(), wf.Status)

	if step != nil {
		_ = w.put(ctx, client, w.stepPath(step), step)
	}
}

func (w *Workflow) clean(ctx context.Context, client *clientv3.Client, err error) {
	w.Lock()
	wf := w.w

	if err != nil {
		if w.snapshot.Action == api.StepAction_SC_CANCEL {
			wf.Status.State = api.WorkflowState_SW_WARN
			w.snapshot.State = api.WorkflowState_SW_WARN
		} else {
			wf.Status.State = api.WorkflowState_SW_FAILED
			w.snapshot.State = api.WorkflowState_SW_FAILED
		}
	} else {
		wf.Status.State = api.WorkflowState_SW_SUCCESS
		w.snapshot.State = api.WorkflowState_SW_SUCCESS
	}

	defer w.Unlock()

	_ = w.put(ctx, client, w.statusPath(), wf.Status)
}

func (w *Workflow) doStep(ctx context.Context, ps *PipeSet, client *clientv3.Client, step *api.WorkflowStep, action api.StepAction) (err error) {

	sname := step.Name + "-" + step.Uid
	w.forward(ctx, client, action, step)

	cid := step.Client
	pipe, ok := ps.Get(cid)
	if !ok {
		return fmt.Errorf("pipe %s not found", cid)
	}

	rch, ech := pipe.Step(NewStep(ctx, action))

	select {
	case <-ctx.Done():
		return context.Canceled
	case err = <-ech:
		return err
	case b := <-rch:
		var entity *api.Entity
		e := json.Unmarshal(b, entity)
		if e != nil {
			log.Errorf("step %s receiving entity failed: %v", sname, e)
			return
		}

		_ = w.put(ctx, client, w.entityPath(entity), entity)
	}

	return nil
}

func (w *Workflow) doPrepare(ps *PipeSet, client *clientv3.Client) (err error) {

	ctx := w.ctx
	steps := w.w.Steps
	action := api.StepAction_SC_PREPARE

	for i := range steps {
		step := steps[i]

		err = w.doStep(ctx, ps, client, step, action)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Workflow) doCommit(ps *PipeSet, client *clientv3.Client) (err error) {

	ctx := w.ctx
	steps := w.w.Steps
	state := api.StepAction_SC_COMMIT
	for i := range steps {
		step := steps[i]

		err = w.doStep(ctx, ps, client, step, state)
		if err != nil {
			return err
		}

		w.committed = append(w.committed, step)
	}

	return nil
}

func (w *Workflow) doRollback(ps *PipeSet, client *clientv3.Client) (err error) {

	ctx := w.ctx
	state := api.StepAction_SC_ROLLBACK
	for i := len(w.committed) - 1; i >= 0; i-- {
		step := w.committed[i]
		err = w.doStep(ctx, ps, client, step, state)
	}

	return nil
}

func (w *Workflow) doCancel(ps *PipeSet, client *clientv3.Client) (err error) {

	ctx := w.ctx
	state := api.StepAction_SC_CANCEL
	for i := len(w.committed) - 1; i >= 0; i-- {
		step := w.committed[i]
		err = w.doStep(ctx, ps, client, step, state)
	}

	return nil
}

func (w *Workflow) Execute(ps *PipeSet, client *clientv3.Client) {

	for i := range w.w.Steps {
		step := w.w.Steps[i]
		if step.Uid == "" {
			step.Uid = uuid.New().String()
		}
		if step.Items == nil {
			step.Items = map[string][]byte{}
		}
		step.Logs = []string{}

		key := path.Join(w.stepPath(step), step.Uid)
		_ = w.put(w.ctx, client, key, step)
	}

	log.Debugf("workflow %s prepare", w.ID())

	var err error
	if err = w.doPrepare(ps, client); err != nil {
		log.Errorf("workflow %s prepare failed: %v", w.ID(), err)
	}

	log.Debugf("workflow %s commit", w.ID())

	if err = w.doCommit(ps, client); err != nil {
		log.Errorf("workflow %s commit failed: %v", w.ID(), err)
	}

	if err != nil {
		log.Debugf("workflow %s rollback", w.ID())
		e := w.doRollback(ps, client)
		if e != nil {
			log.Errorf("workflow %s rollback failed: %v", w.ID(), e)
		}
	}

	log.Debugf("workflow %s cancel", w.ID())
	e := w.doCancel(ps, client)
	if e != nil {
		log.Errorf("workflow %s cancel failed: %v", w.ID(), err)
	}
}

type Scheduler struct {
	wg   sync.WaitGroup
	pool *ants.Pool

	storage *clientv3.Client

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

func (s *Scheduler) ExecuteWorkflow(w *api.Workflow, ps *PipeSet) error {

	select {
	case <-s.exit:
		return fmt.Errorf("scheduler stopped")
	default:
	}

	for _, entity := range w.Entities {
		if !s.entitySet.Contains(entity.Kind) {
			return fmt.Errorf("unknwon entity: kind=%s", entity.Kind)
		}
	}

	for _, step := range w.Steps {
		if !s.stepSet.Contains(step.Name) {
			return fmt.Errorf("unknown step: name=%s", step.Name)
		}
	}

	s.wg.Add(1)
	err := s.pool.Submit(func() {
		defer s.wg.Done()

		wf := NewWorkflow(w)
		defer wf.Cancel()

		s.wmu.Lock()
		s.wfm[wf.ID()] = wf
		s.wmu.Unlock()

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
	close(s.exit)
}
