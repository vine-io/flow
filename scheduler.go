package flow

import (
	"context"
	"fmt"
	"path"
	"sync"

	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	json "github.com/json-iterator/go"
	"github.com/vine-io/flow/api"
	log "github.com/vine-io/vine/lib/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Scheduler struct {
	storage *clientv3.Client

	// zeebe client
	zbClient zbc.Client
	// zeebe job worker
	userWorker worker.JobWorker
	// zeebe job worker
	serviceWorker worker.JobWorker

	smu       sync.RWMutex
	entitySet *EntitySet
	echoSet   *EchoSet
	stepSet   *StepSet

	wmu sync.RWMutex
	wfm map[string]*Workflow

	exit chan struct{}
}

func NewScheduler(storage *clientv3.Client, zbAddr string) (*Scheduler, error) {
	zbClient, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         zbAddr,
		UsePlaintextConnection: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to zeebe: %v", err)
	}

	s := &Scheduler{
		storage:   storage,
		zbClient:  zbClient,
		entitySet: NewEntitySet(),
		echoSet:   NewEchoSet(),
		stepSet:   NewStepSet(),
		wfm:       map[string]*Workflow{},
		exit:      make(chan struct{}, 1),
	}

	//zbClient.NewJobWorker().JobType("user").Handler(s.userWorker).Open()

	return s, nil
}

func (s *Scheduler) Register(worker *api.Worker, entities []*api.Entity, echoes []*api.Echo, steps []*api.Step) error {

	ctx := context.Background()
	key := path.Join(Root, "worker", worker.Id)
	val, _ := json.Marshal(worker)
	_, err := s.storage.Put(ctx, key, string(val))
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

func (s *Scheduler) GetWorkers(ctx context.Context) ([]*api.Worker, error) {
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	key := path.Join(Root, "worker")
	rsp, err := s.storage.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}

	workers := make([]*api.Worker, 0)
	for _, kv := range rsp.Kvs {
		worker := &api.Worker{}
		if err = json.Unmarshal(kv.Value, &worker); err == nil {
			workers = append(workers, worker)
		}
	}

	return workers, nil
}

func (s *Scheduler) DeployWorkflow(ctx context.Context, resource *api.BpmnResource) (int64, error) {

	name := resource.Name + ".bpmn"
	rsp, err := s.zbClient.NewDeployResourceCommand().AddResource(resource.Definition, name).Send(ctx)
	if err != nil {
		return 0, err
	}

	data, _ := json.Marshal(resource)

	key := path.Join(Root, "definitions", resource.Id)
	_, err = s.storage.Put(ctx, key, string(data))
	if err != nil {
		return 0, err
	}

	return rsp.GetKey(), nil
}

func (s *Scheduler) ListWorkflow(ctx context.Context) ([]*api.BpmnResource, error) {

	key := path.Join(Root, "definitions")
	rsp, err := s.storage.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	resources := make([]*api.BpmnResource, 0)
	for _, kv := range rsp.Kvs {
		resource := api.BpmnResource{}
		if e := json.Unmarshal(kv.Value, &resource); e == nil {
			resources = append(resources, &resource)
		}
	}

	return resources, nil
}

func (s *Scheduler) GetWorkflowInstance(wid string) (*Workflow, bool) {
	s.wmu.RLock()
	defer s.wmu.RUnlock()
	w, ok := s.wfm[wid]
	return w, ok
}

func (s *Scheduler) InspectWorkflowInstance(ctx context.Context, wid string) (*api.Workflow, error) {
	w, ok := s.GetWorkflowInstance(wid)
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

func (s *Scheduler) GetWorkflowInstances() []*api.WorkflowSnapshot {
	s.wmu.RLock()
	defer s.wmu.RUnlock()
	snapshots := make([]*api.WorkflowSnapshot, 0)
	for _, w := range s.wfm {
		snapshots = append(snapshots, w.NewSnapshot())
	}

	return snapshots
}

func (s *Scheduler) WatchWorkflowInstance(ctx context.Context, wid string) (<-chan *api.WorkflowWatchResult, error) {
	w, ok := s.GetWorkflowInstance(wid)
	if !ok {
		return nil, fmt.Errorf("workflow not found")
	}

	return w.NewWatcher(ctx, s.storage)
}

func (s *Scheduler) StepPut(ctx context.Context, wid, key, value string) error {
	w, ok := s.GetWorkflowInstance(wid)
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
	w, ok := s.GetWorkflowInstance(wid)
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
	_, ok := s.GetWorkflowInstance(wid)
	if !ok {
		return fmt.Errorf("workflow not found")
	}

	log.Trace("step %s trace: %s", step, string(text))
	// TODO: trace log record

	return nil
}

func (s *Scheduler) ExecuteWorkflowInstance(id string, ps *PipeSet) error {
	if s.IsClosed() {
		return fmt.Errorf("scheduler stopped")
	}

	ctx := context.Background()
	rsp, err := s.zbClient.NewCreateInstanceCommand().BPMNProcessId(id).LatestVersion().Send(ctx)
	if err != nil {
		return err
	}

	pKey := rsp.ProcessInstanceKey

	//s.smu.RLock()
	//for _, entity := range w.Entities {
	//	if !s.entitySet.Contains(entity.Kind) {
	//		s.smu.RUnlock()
	//		return fmt.Errorf("unknwon entity: kind=%s", entity.Kind)
	//	}
	//}
	//
	//for _, step := range w.Steps {
	//	if !s.stepSet.Contains(step.Name) {
	//		s.smu.RUnlock()
	//		return fmt.Errorf("unknown step: name=%s", step.Name)
	//	}
	//	if _, ok := ps.Get(step.Worker); !ok {
	//		s.smu.RUnlock()
	//		return fmt.Errorf("worker %s not found", step.Worker)
	//	}
	//}
	//s.smu.RUnlock()
	//
	wf := NewWorkflow(w)

	if _, ok := s.GetWorkflowInstance(id); ok {
		return fmt.Errorf("workflow already exists")
	}
	//
	//if err := wf.Init(s.storage); err != nil {
	//	return fmt.Errorf("initialize workflow %s failed: %v", wf.ID(), err)
	//}
	//
	//s.wmu.Lock()
	//s.wfm[wf.ID()] = wf
	//s.wmu.Unlock()
	//
	//s.wg.Add(1)
	//err := s.pool.Submit(func() {
	//	defer s.wg.Done()
	//
	//	defer wf.Cancel()
	//
	//	defer func() {
	//		s.wmu.Lock()
	//		delete(s.wfm, wf.ID())
	//		s.wmu.Unlock()
	//	}()
	//
	//	wf.Execute(ps, s.storage)
	//})
	//
	//if err != nil {
	//	return err
	//}

	return nil
}

func (s *Scheduler) Stop() {
	if s.IsClosed() {
		return
	}
	close(s.exit)
	s.userWorker.Close()
	s.userWorker.AwaitClose()
	s.serviceWorker.Close()
	s.serviceWorker.AwaitClose()
	_ = s.zbClient.Close()
}

func (s *Scheduler) IsClosed() bool {
	select {
	case <-s.exit:
		return true
	default:
		return false
	}
}

func handleUserJob(conn worker.JobClient, job entities.Job) {

}

func handlerServiceJob(conn worker.JobClient, job entities.Job) {
	key := job.GetProcessInstanceKey()

	headers, err := job.GetCustomHeadersAsMap()
	if err != nil {

	}

	vars, err := job.GetVariablesAsMap()
	if err != nil {

	}

	for key, value := range vars {

	}
}
