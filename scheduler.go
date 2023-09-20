package flow

import (
	"context"
	"encoding/xml"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/olive-io/bpmn/flow"
	"github.com/olive-io/bpmn/flow_node/activity"
	"github.com/olive-io/bpmn/flow_node/activity/service"
	"github.com/olive-io/bpmn/process"
	"github.com/olive-io/bpmn/process/instance"
	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/bpmn/tracing"
	"github.com/panjf2000/ants/v2"
	"github.com/vine-io/flow/api"
	"github.com/vine-io/flow/bpmn"
	log "github.com/vine-io/vine/lib/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Scheduler struct {
	name string
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

func NewScheduler(name string, storage *clientv3.Client, size int) (*Scheduler, error) {
	pool, err := ants.NewPool(size)
	if err != nil {
		return nil, err
	}

	s := &Scheduler{
		name:      name,
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
		w := &api.Worker{}
		if err = json.Unmarshal(kv.Value, &w); err == nil {
			workers = append(workers, w)
		}
	}

	return workers, nil
}

func (s *Scheduler) GetWorker(ctx context.Context, id string) (*api.Worker, error) {
	options := []clientv3.OpOption{}

	key := path.Join(Root, "worker", id)
	rsp, err := s.storage.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, api.ErrNotFound("worker=%s", id)
	}

	w := &api.Worker{}
	if err = json.Unmarshal(rsp.Kvs[0].Value, &w); err != nil {
		return nil, err
	}

	return w, nil
}

func (s *Scheduler) DeployWorkflow(ctx context.Context, resource *api.BpmnResource) (int64, error) {

	b, err := bpmn.FromXML(string(resource.Definition))
	if err != nil {
		return 0, err
	}

	p, err := b.DefaultProcess()
	if err != nil {
		return 0, err
	}
	p.Elements.ScanMut(func(key string, value bpmn.Element) bool {
		if value.GetShape() == bpmn.ServiceTaskShape {
			task := value.(*bpmn.ServiceTask)
			if task.Extension == nil {
				task.SetExtension(&bpmn.ExtensionElement{TaskDefinition: &bpmn.TaskDefinition{Type: "dr-service-" + s.name}})
			} else {
				td := task.Extension.TaskDefinition
				if td == nil {
					task.Extension.TaskDefinition = &bpmn.TaskDefinition{Type: "dr-service-" + s.name}
				} else {
					switch td.Type {
					case "dr-user":
						td.Type = "dr-user-" + s.name
					case "dr-service":
						td.Type = "dr-service-" + s.name
					default:
						td.Type = "dr-service-" + s.name
					}
				}
			}
		}
		return true
	})

	content, err := b.WriteToBytes()
	if err != nil {
		return 0, err
	}
	resource.Definition = content

	data, _ := json.Marshal(resource)

	key := path.Join(Root, "definitions", resource.Id)
	_, err = s.storage.Put(ctx, key, string(data))
	if err != nil {
		return 0, err
	}

	return 1, nil
}

func (s *Scheduler) GetWorkflow(ctx context.Context, id string) (*api.BpmnResource, error) {

	key := path.Join(Root, "definitions", id)
	rsp, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) == 0 {
		return nil, api.ErrNotFound("id=%s", id)
	}

	resource := &api.BpmnResource{}
	err = json.Unmarshal(rsp.Kvs[0].Value, resource)
	if err != nil {
		return nil, api.ErrInternalServerError(err.Error())
	}

	return resource, nil
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

	if len(rsp.Kvs) == 0 {
		return nil, api.ErrNotFound("workflow deployment %s not exists", id)
	}

	resource := api.BpmnResource{}
	content := rsp.Kvs[0].Value
	err = json.Unmarshal(content, &resource)
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

func (s *Scheduler) SetWorkflowInstance(wf *Workflow) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	s.wfm[wf.ID()] = wf
}

func (s *Scheduler) RemoveWorkflowInstance(wid string) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	delete(s.wfm, wid)
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

func (s *Scheduler) StepTrace(ctx context.Context, traceLog *api.TraceLog) error {
	w, ok := s.GetWorkflowInstance(traceLog.Wid)
	if !ok {
		return fmt.Errorf("workflow not found")
	}

	log.Trace("step %s trace: %s", traceLog.Sid, string(traceLog.Text))

	return w.trace(ctx, traceLog)
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

	wf, ok := s.GetWorkflowInstance(pid)
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

func (s *Scheduler) ExecuteWorkflowInstance(id, name, definitionsText string, dataObjects, properties map[string]string, ps *PipeSet) error {
	if s.IsClosed() {
		return fmt.Errorf("scheduler stopped")
	}

	if _, ok := s.GetWorkflowInstance(id); ok {
		return fmt.Errorf("workflow already exists")
	}

	definitions, err := bpmn.FromXML(definitionsText)
	if err != nil {
		return err
	}

	content, _ := definitions.WriteToBytes()

	processBpmn, err := definitions.DefaultProcess()
	if err != nil {
		return err
	}

	if processBpmn.ExtensionElement != nil && processBpmn.ExtensionElement.Headers != nil {
		for _, item := range processBpmn.ExtensionElement.Headers.Items {
			key := item.Name
			if key == "__entities" {
				continue
			}
			if _, ok := properties[key]; !ok {
				return fmt.Errorf("missing property key=%v", key)
			}
		}
	}

	pvars := map[string]any{}
	items := map[string]string{}
	for key, value := range properties {
		pvars[key] = value
		if key == "action" {
			continue
		}
		items[oliveUnEscape(key)] = value
	}
	pvars["action"] = api.StepAction_SC_PREPARE.Readably()

	dos := make(map[string]any)
	for key, do := range dataObjects {
		dos[key] = do
	}

	var schemaDefinitions *schema.Definitions
	if err = xml.Unmarshal(content, &schemaDefinitions); err != nil {
		return err
	}

	ech := make(chan error, 1)
	done := make(chan struct{}, 1)
	instanceId, err := s.executeProcess(id, content, dos, pvars, ech, done)
	if err != nil {
		return fmt.Errorf("execute definition: %v", err)
	}

	wf := NewWorkflow(id, instanceId, name, items, s.storage, ps)
	if err = wf.Init(); err != nil {
		return fmt.Errorf("initalize workflow %s: %v", id, err)
	}
	s.SetWorkflowInstance(wf)

	log.Infof("create new process %s instance %s", id, instanceId)

	go func() {
		prepared := false

	LOOP:
		for {
			select {
			case <-done:
				if !prepared {
					prepared = true
					pvars["action"] = api.StepAction_SC_COMMIT.Readably()
					_, err = s.executeProcess(id, content, dos, pvars, ech, done)
					if err != nil {
						break LOOP
					}
				} else {
					break LOOP
				}
			case e1 := <-ech:
				log.Errorf("%v", e1)
				break LOOP
			}
		}

		log.Infof("Process %s Committed", instanceId)
		swf, ok := s.getWorkflowInstanceRetry(id, 3)
		if ok {
			s.wmu.Lock()
			delete(s.wfm, id)
			s.wmu.Unlock()
			swf.Destroy()
		}
	}()

	s.wg.Add(1)
	err = s.pool.Submit(func() {
		defer s.wg.Done()

		defer s.RemoveWorkflowInstance(wf.ID())

		wf.Execute()

		log.Infof("workflow %s done!", wf.ID())
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) executeProcess(
	id string, content []byte,
	dataObjects, variables map[string]any,
	ech chan<- error, done chan<- struct{},
) (string, error) {
	var schemaDefinitions *schema.Definitions
	if err := xml.Unmarshal(content, &schemaDefinitions); err != nil {
		return "", err
	}

	processElement := (*schemaDefinitions.Processes())[0]
	proc := process.New(&processElement, schemaDefinitions)
	options := []instance.Option{
		instance.WithDataObjects(dataObjects),
		instance.WithVariables(variables),
	}
	ins, err := proc.Instantiate(options...)
	if err != nil {
		return "", fmt.Errorf("failed to instantiate the process: %s", err)
	}

	traces := ins.Tracer.Subscribe()
	err = ins.StartAll(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to run the instance: %s", err)
	}

	go func() {
	LOOP:
		for {
			unwrapped := tracing.Unwrap(<-traces)
			switch tt := unwrapped.(type) {
			case flow.Trace:
			case activity.ActiveTaskTrace:
				switch stt := tt.(type) {
				case *service.ActiveTrace:
					s.handlerServiceJob(id, stt)
				//case *user.ActiveTrace:
				default:
					stt.Execute()
				}
			case tracing.ErrorTrace:
				ech <- tt.Error
				break LOOP
			case flow.CeaseFlowTrace:
				done <- struct{}{}
				break LOOP
			default:
				log.Infof("%#v", tt)
			}
		}

		ins.Tracer.Unsubscribe(traces)
	}()

	return ins.Id().String(), nil
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

//func (s *Scheduler) handleUserJob() func(conn worker.JobClient, job entities.Job) {
//	return func(conn worker.JobClient, job entities.Job) {
//		ctx := context.Background()
//		jobKey := job.Key
//		pid := job.BpmnProcessId
//
//		wf, ok := s.getWorkflowInstanceRetry(pid, 3)
//		if !ok {
//			s.failJob(conn, job, fmt.Errorf("workflow can't on active"))
//			return
//		}
//
//		headers, err := job.GetCustomHeadersAsMap()
//		if err != nil {
//			s.failJob(conn, job, err)
//			return
//		}
//
//		vars, err := job.GetVariablesAsMap()
//		if err != nil {
//			s.failJob(conn, job, err)
//			return
//		}
//
//		sid := job.ElementId
//		step := &api.WorkflowStep{
//			Uid:    sid,
//			Stages: []*api.WorkflowStepStage{},
//		}
//		if v, ok := headers["stepName"]; ok {
//			step.Name = oliveUnEscape(v)
//		}
//
//		it := &api.Interactive{Pid: pid, Sid: sid, Describe: step.Name, Properties: []*api.Property{}}
//		for k, v := range vars {
//			it.Properties = append(it.Properties, &api.Property{
//				Name:  k,
//				Type:  api.PropertyType_PYString,
//				Value: v.(string),
//			})
//		}
//
//		_ = wf.InteractiveHandle(ctx, step, it)
//		log.Infof("Successfully completed job %d, type %s", jobKey, job.Type)
//	}
//}

func (s *Scheduler) handlerServiceJob(pid string, activeTrace *service.ActiveTrace) {

	//ctx := activeTrace.Context
	id, _ := activeTrace.Activity.Element().Id()
	sid := *id
	headers := activeTrace.Headers
	vars := activeTrace.Properties
	//sid := job.ElementId

	log.Infof("processing job %s in process %s", sid, pid)

	wf, ok := s.getWorkflowInstanceRetry(pid, 3)
	if !ok {
		err := fmt.Errorf("workflow can't on active")
		activeTrace.Do(nil, nil, err, nil)
		return
	}

	step := &api.WorkflowStep{
		Uid:    sid,
		Stages: []*api.WorkflowStepStage{},
	}
	if v, ok := headers["stepName"]; ok {
		step.Name = oliveUnEscape(v.(string))
	}
	if v, ok := headers["describe"]; ok {
		step.Describe = v.(string)
	}
	if v, ok := headers["injects"]; ok {
		step.Injects = strings.Split(v.(string), ",")
	}
	if v, ok := headers["entity"]; ok {
		step.Entity = v.(string)
	}
	if v, ok := headers["worker"]; ok {
		step.Worker = v.(string)
	}
	if v, ok := vars["__step_mapping__"+sid]; ok {
		step.Worker = v.(string)
	}
	//if v, ok := headers["completed"]; ok {
	//	completed = v == "true"
	//}

	var action api.StepAction
	if v, ok := vars["action"]; ok {
		_ = action.UnmarshalJSON([]byte(v.(string)))
	} else {
		action = api.StepAction_SC_PREPARE
	}

	items := make(map[string]string)
	mappings := make(map[string]string)
	for key, value := range vars {
		if key == "action" {
			continue
		}
		if strings.HasPrefix(key, "__step_mapping__") {
			mappings[key] = value.(string)
			continue
		}
		items[oliveUnEscape(key)] = value.(string)
	}

	var out map[string]string
	out, err := wf.Handle(step, action, items)
	if err != nil {

	}
	result := map[string]any{}
	for key, value := range out {
		result[key] = value
	}

	activeTrace.Do(nil, result, err, nil)

	if err != nil {

	} else {
		log.Infof("Successfully completed job %s", sid)
	}
}

func (s *Scheduler) getWorkflowInstanceRetry(pid string, retry int) (*Workflow, bool) {
	wf, ok := s.GetWorkflowInstance(pid)
	if !ok && retry > 0 {
		time.Sleep(time.Millisecond * 1000)
		return s.getWorkflowInstanceRetry(pid, retry-1)
	}
	return wf, ok
}
