package flow

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	json "github.com/json-iterator/go"
	"github.com/panjf2000/ants/v2"
	"github.com/vine-io/flow/api"
	"github.com/vine-io/flow/bpmn"
	"github.com/vine-io/flow/zeebe/pkg/entities"
	"github.com/vine-io/flow/zeebe/pkg/worker"
	"github.com/vine-io/flow/zeebe/pkg/zbc"
	log "github.com/vine-io/vine/lib/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Scheduler struct {
	wg   sync.WaitGroup
	pool *ants.Pool

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

func NewScheduler(storage *clientv3.Client, zbAddr string, size int) (*Scheduler, error) {
	zbClient, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         zbAddr,
		UsePlaintextConnection: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to zeebe: %v", err)
	}

	pool, err := ants.NewPool(size)
	if err != nil {
		return nil, err
	}

	s := &Scheduler{
		pool:      pool,
		storage:   storage,
		zbClient:  zbClient,
		entitySet: NewEntitySet(),
		echoSet:   NewEchoSet(),
		stepSet:   NewStepSet(),
		wfm:       map[string]*Workflow{},
		exit:      make(chan struct{}, 1),
	}

	s.userWorker = zbClient.NewJobWorker().JobType("dr-user").Handler(s.handleUserJob()).Open()
	s.serviceWorker = zbClient.NewJobWorker().JobType("dr-service").Handler(s.handlerServiceJob()).Open()

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

	for _, deploy := range rsp.Deployments {
		deploy.GetProcess()
	}

	data, _ := json.Marshal(resource)

	key := path.Join(Root, "definitions", resource.Id)
	_, err = s.storage.Put(ctx, key, string(data))
	if err != nil {
		return 0, err
	}

	return rsp.Key, nil
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

func (s *Scheduler) GetWorkflowDeployment(ctx context.Context, id string) (*bpmn.Definitions, error) {

	key := path.Join(Root, "definitions", id)
	rsp, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	resource := api.BpmnResource{}
	err = json.Unmarshal(rsp.Kvs[0].Value, &resource)
	if err != nil {
		return nil, err
	}

	definitions, err := bpmn.FromXML(string(resource.Definition))
	if err != nil {
		return nil, err
	}

	return definitions, nil
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

	data, err := w.Inspect(ctx)
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

func (s *Scheduler) ListInteractive(ctx context.Context, pid string) ([]*api.Interactive, error) {
	interactive := make([]*api.Interactive, 0)

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}
	key := path.Join(Root, "interactive", pid)
	rsp, err := s.storage.Get(ctx, key, options...)
	if err != nil {
		return nil, api.ErrInsufficientStorage(err.Error())
	}

	for _, kv := range rsp.Kvs {
		it := &api.Interactive{}
		if e := json.Unmarshal(kv.Value, it); e != nil {
			s.storage.Delete(ctx, string(kv.Key))
			continue
		}
		interactive = append(interactive, it)
	}

	return interactive, nil
}

func (s *Scheduler) CommitInteractive(ctx context.Context, pid, sid string, properties map[string]string) error {

	s.wmu.Lock()
	wf, ok := s.wfm[pid]
	s.wmu.Unlock()

	if ok {
		return nil
	}

	it := &api.Interactive{
		Pid:        pid,
		Sid:        sid,
		Properties: []*api.Property{},
	}

	for k, v := range properties {
		it.Properties = append(it.Properties, &api.Property{
			Name:  k,
			Value: v,
		})
	}

	wf.CommitInteractive(it)

	return nil
}

func (s *Scheduler) ExecuteWorkflowInstance(id, name string, ps *PipeSet) error {
	if s.IsClosed() {
		return fmt.Errorf("scheduler stopped")
	}

	ctx := context.Background()
	definitions, err := s.GetWorkflowDeployment(ctx, id)
	if err != nil {
		return err
	}
	process, err := definitions.DefaultProcess()
	if err != nil {
		return err
	}

	pvars := map[string]interface{}{}
	if process.ExtensionElement != nil && process.ExtensionElement.Properties != nil {
		properties := process.ExtensionElement.Properties.Items
		for _, item := range properties {
			pvars[item.Name] = item.Value
		}
	}
	if _, ok := pvars["action"]; !ok {
		pvars["action"] = api.StepAction_SC_PREPARE.Readably()
	}

	req, err := s.zbClient.NewCreateInstanceCommand().BPMNProcessId(id).LatestVersion().VariablesFromMap(pvars)
	if err != nil {
		return err
	}

	rsp, err := req.Send(ctx)
	if err != nil {
		return err
	}

	log.Infof("create new process instance %d", rsp.ProcessInstanceKey)

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
	wf := NewWorkflow(id, name, s.storage, ps)
	if err = wf.Init(); err != nil {
		return fmt.Errorf("initalize workflow %s: %v", id, err)
	}

	if _, ok := s.GetWorkflowInstance(id); ok {
		return fmt.Errorf("workflow already exists")
	}

	s.wmu.Lock()
	s.wfm[wf.ID()] = wf
	s.wmu.Unlock()

	s.wg.Add(1)
	err = s.pool.Submit(func() {
		defer s.wg.Done()

		defer wf.Cancel()

		defer func() {
			s.wmu.Lock()
			delete(s.wfm, wf.ID())
			s.wmu.Unlock()
		}()

		wf.Execute()
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

func (s *Scheduler) handleUserJob() func(conn worker.JobClient, job entities.Job) {
	return func(conn worker.JobClient, job entities.Job) {
		ctx := context.Background()
		jobKey := job.Key
		pid := job.BpmnProcessId

		s.wmu.Lock()
		wf, ok := s.wfm[pid]
		s.wmu.Unlock()

		if !ok {
			failJob(conn, job, fmt.Errorf("workflow can't on active"))
			return
		}

		headers, err := job.GetCustomHeadersAsMap()
		if err != nil {
			failJob(conn, job, err)
			return
		}

		vars, err := job.GetVariablesAsMap()
		if err != nil {
			failJob(conn, job, err)
			return
		}

		sid := job.ElementId
		step := &api.WorkflowStep{
			Uid:    sid,
			Args:   &api.WorkflowArgs{Args: map[string]string{}},
			Stages: []*api.WorkflowStepStage{},
		}
		if v, ok := headers["stepName"]; ok {
			step.Name = zeebeUnEscape(v)
		}

		it := &api.Interactive{Pid: pid, Sid: sid, Describe: step.Name, Properties: []*api.Property{}}
		for k, v := range vars {
			it.Properties = append(it.Properties, &api.Property{
				Name:  k,
				Type:  api.PropertyType_PYString,
				Value: v.(string),
			})
		}

		_ = wf.InteractiveHandle(ctx, step, it)
		log.Infof("Successfully completed job %d, type %s", jobKey, job.Type)
	}
}

func (s *Scheduler) handlerServiceJob() func(conn worker.JobClient, job entities.Job) {
	return func(conn worker.JobClient, job entities.Job) {

		ctx := context.Background()
		jobKey := job.Key
		pid := job.BpmnProcessId
		elementId := job.ElementId

		log.Infof("processing job %d of type %s from element %s in process %s", jobKey, job.Type, elementId, pid)

		s.wmu.Lock()
		wf, ok := s.wfm[pid]
		s.wmu.Unlock()

		if !ok {
			failJob(conn, job, fmt.Errorf("workflow can't on active"))
			return
		}

		sid := job.ElementId

		headers, err := job.GetCustomHeadersAsMap()
		if err != nil {
			failJob(conn, job, err)
			return
		}

		vars, err := job.GetVariablesAsMap()
		if err != nil {
			failJob(conn, job, err)
			return
		}

		completed := false

		step := &api.WorkflowStep{
			Uid:    sid,
			Args:   &api.WorkflowArgs{Args: map[string]string{}},
			Stages: []*api.WorkflowStepStage{},
		}
		if v, ok := headers["stepName"]; ok {
			step.Name = zeebeUnEscape(v)
		}
		if v, ok := headers["describe"]; ok {
			step.Describe = v
		}
		if v, ok := headers["injects"]; ok {
			step.Injects = strings.Split(v, ",")
		}
		if v, ok := headers["entity"]; ok {
			step.Entity = v
		}
		if v, ok := headers["worker"]; ok {
			step.Worker = v
		}
		if v, ok := headers["completed"]; ok {
			completed = v == "true"
		}

		var action api.StepAction
		if v, ok := vars["action"]; ok {
			_ = action.UnmarshalJSON([]byte(v.(string)))
		} else {
			action = api.StepAction_SC_PREPARE
		}

		items := make(map[string]string)
		entity := &api.Entity{}
		for key, value := range vars {
			if strings.HasPrefix(key, sid+"___") {
				keyText := strings.TrimPrefix(key, sid+"___")
				step.Args.Args[keyText] = value.(string)
				continue
			}
			if key == "action" {
				continue
			}
			items[zeebeUnEscape(key)] = value.(string)
		}
		if v, ok := vars["entity___"+zeebeEscape(step.Entity)]; ok {
			entity.Kind = step.Entity
			entity.Raw = v.(string)
		}

		var deferErr error
		defer func() {
			switch {
			case completed && action == api.StepAction_SC_PREPARE:
				pvars := map[string]interface{}{"action": api.StepAction_SC_COMMIT.Readably()}
				req, e1 := s.zbClient.NewCreateInstanceCommand().BPMNProcessId(pid).LatestVersion().VariablesFromMap(pvars)
				if e1 != nil {
					log.Errorf("track process %d to commit: %v", pid, e1)
					return
				}
				rsp, e1 := req.Send(ctx)
				if e1 != nil {
					log.Errorf("track process %d to commit: %v", pid, e1)
					return
				}

				log.Infof("Process %s Prepared, create new instance %d", pid, rsp.ProcessInstanceKey)
			case (completed || deferErr != nil) && action == api.StepAction_SC_COMMIT:
				wf.Destroy()
				log.Infof("Process %s Committed, destroy it", pid)
				s.wmu.Lock()
				delete(s.wfm, pid)
				s.wmu.Unlock()
			}
		}()

		err = wf.Handle(step, action, items, entity)
		if err != nil {
			if IsShadowErr(err) {
				failShadowJob(conn, job, err)
				return
			}
			deferErr = err
			failJob(conn, job, err)
			return
		}

		_, err = conn.NewCompleteJobCommand().JobKey(jobKey).Send(ctx)
		if err != nil {
			deferErr = err
			log.Errorf("send to zeebe: %v", err)
			return
		}

		log.Infof("Successfully completed job %d, type %s", jobKey, job.Type)
	}
}

func failJob(client worker.JobClient, job entities.Job, err error) {
	log.Errorf("Failed to complete workflow %s: %v", job.BpmnProcessId, err)

	apiErr := api.FromErr(err)
	ctx := context.Background()
	_, e := client.NewFailJobCommand().JobKey(job.Key).Retries(apiErr.Retries).ErrorMessage(apiErr.Detail).Send(ctx)
	if e != nil {
		return
	}
}

func failShadowJob(client worker.JobClient, job entities.Job, err error) {
	log.Errorf("Failed to complete workflow %s (shadow): %v", job.BpmnProcessId, err)

	ctx := context.Background()
	resultKey := job.ElementId + "___result"
	errKey := job.ElementId + "___msg"
	pvars := map[string]interface{}{
		resultKey: false,
		errKey:    err.Error(),
	}
	req, e := client.NewCompleteJobCommand().JobKey(job.Key).VariablesFromMap(pvars)
	if e != nil {
		return
	}

	_, e = req.Send(ctx)
	if e != nil {
		return
	}
}