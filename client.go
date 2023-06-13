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
	"io"
	"path"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/vine-io/flow/api"
	"github.com/vine-io/pkg/inject"
	vclient "github.com/vine-io/vine/core/client"
	"github.com/vine-io/vine/core/client/grpc"
	verrs "github.com/vine-io/vine/lib/errors"
	log "github.com/vine-io/vine/lib/logger"
)

const (
	minHeartbeat = time.Second * 10
)

var gStore = NewClientStore()

func Load(tps ...any) {
	gStore.Load(tps...)
}

func Provides(values ...any) error {
	return gStore.Provides(values...)
}

func ProvideWithName(name string, value any) error {
	return gStore.ProvideWithName(name, value)
}

type ClientStore struct {
	container inject.Container
	entitySet map[string]Entity
	echoSet   map[string]Echo
	stepSet   map[string]reflect.Type
}

func NewClientStore() *ClientStore {
	s := &ClientStore{
		container: inject.Container{},
		entitySet: map[string]Entity{},
		echoSet:   map[string]Echo{},
		stepSet:   map[string]reflect.Type{},
	}
	return s
}

func (s *ClientStore) Load(ts ...any) {
	for _, t := range ts {
		tp := reflect.TypeOf(t)
		kind := GetTypePkgName(tp)
		switch tt := t.(type) {
		case Entity:
			s.entitySet[kind] = tt
		case Echo:
			s.echoSet[kind] = tt
		case Step:
			if tp.Kind() == reflect.Ptr {
				tp = tp.Elem()
			}
			s.stepSet[kind] = tp
		}
	}
}

func (s *ClientStore) Provides(values ...any) error {
	for _, v := range values {
		err := s.container.Provide(&inject.Object{
			Value: v,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ClientStore) ProvideWithName(name string, value any) error {
	err := s.container.Provide(&inject.Object{
		Value: value,
		Name:  name,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *ClientStore) GetEntity(kind string) (Entity, bool) {
	e, ok := s.entitySet[kind]
	return e, ok
}

func (s *ClientStore) GetEcho(name string) (Echo, bool) {
	e, ok := s.echoSet[name]
	return e, ok
}

func (s *ClientStore) PopulateEcho(name string) (Echo, error) {
	echo, ok := s.GetEcho(name)
	if !ok {
		return nil, fmt.Errorf("not found Echo<%s>", name)
	}

	err := s.container.PopulateTarget(echo)
	if err != nil {
		return nil, err
	}

	return echo, nil
}

func (s *ClientStore) GetStep(name string) (Step, bool) {
	tp, ok := s.stepSet[name]
	if !ok {
		return nil, false
	}
	step, ok := reflect.New(tp).Interface().(Step)
	return step, ok
}

func (s *ClientStore) PopulateStep(name string) (Step, error) {
	step, ok := s.GetStep(name)
	if !ok {
		return nil, fmt.Errorf("not found Step<%s>", name)
	}

	err := s.container.PopulateTarget(step)
	if err != nil {
		return nil, err
	}

	return step, nil
}

type ClientConfig struct {
	name      string
	id        string
	address   string
	timeout   time.Duration
	heartbeat time.Duration
	conn      vclient.Client

	store *ClientStore
}

func NewConfig(name, id, address string) ClientConfig {
	c := ClientConfig{
		name:      name,
		id:        id,
		address:   address,
		timeout:   time.Second * 15,
		heartbeat: time.Second * 30,
		store:     gStore,
	}
	return c
}

func (c *ClientConfig) WithStore(store *ClientStore) *ClientConfig {
	c.store = store
	return c
}

func (c *ClientConfig) WithConn(conn vclient.Client) *ClientConfig {
	c.conn = conn
	return c
}

func (c *ClientConfig) WithTimeout(t time.Duration) *ClientConfig {
	c.timeout = t
	return c
}

func (c *ClientConfig) WithHeartbeat(t time.Duration) *ClientConfig {
	if t < minHeartbeat {
		t = minHeartbeat
	}
	c.heartbeat = t
	return c
}

func (c *ClientConfig) dialOption() []vclient.Option {
	options := []vclient.Option{
		vclient.Retries(0),
	}
	return options
}

func (c *ClientConfig) callOptions() []vclient.CallOption {
	return []vclient.CallOption{
		vclient.WithAddress(c.address),
		vclient.WithDialTimeout(c.timeout),
		vclient.WithRequestTimeout(c.timeout),
	}
}

func (c *ClientConfig) streamOptions() []vclient.CallOption {
	options := []vclient.CallOption{
		vclient.WithAddress(c.address),
		vclient.WithStreamTimeout(0),
	}
	return options
}

type WorkflowWatcher interface {
	Next() (*api.WorkflowWatchResult, error)
}

type Client struct {
	cfg ClientConfig
	s   api.FlowRpcService
}

func NewClient(cfg ClientConfig, attrs map[string]string) (*Client, error) {
	if cfg.conn == nil {
		cfg.conn = grpc.NewClient(cfg.dialOption()...)
	}

	if err := cfg.conn.Init(); err != nil {
		return nil, err
	}

	var err error
	service := api.NewFlowRpcService(cfg.name, cfg.conn)
	ctx := context.Background()
	store := cfg.store

	worker := &api.Worker{
		Id: cfg.id,
	}

	entities := make([]*api.Entity, 0)
	for _, item := range store.entitySet {
		e := EntityToAPI(item)
		e.Workers = map[string]*api.Worker{worker.Id: worker}
		entities = append(entities, e)
	}
	echoes := make([]*api.Echo, 0)
	for _, item := range store.echoSet {
		echo := EchoToAPI(item)
		echo.Workers = map[string]*api.Worker{worker.Id: worker}
		echoes = append(echoes, echo)
	}
	steps := make([]*api.Step, 0)
	for name := range store.stepSet {
		item, _ := store.GetStep(name)
		step := StepToAPI(item)
		step.Workers = map[string]*api.Worker{worker.Id: worker}
		steps = append(steps, step)
	}
	in := &api.RegisterRequest{
		Id:       cfg.id,
		Attrs:    attrs,
		Entities: entities,
		Echoes:   echoes,
		Steps:    steps,
	}
	_, err = service.Register(ctx, in, cfg.callOptions()...)
	if err != nil {
		return nil, err
	}

	c := &Client{
		cfg: cfg,
		s:   service,
	}

	return c, nil
}

func (c *Client) Id() string {
	return c.cfg.id
}

func (c *Client) ListWorker(ctx context.Context) ([]*api.Worker, error) {
	in := &api.ListWorkerRequest{}
	rsp, err := c.s.ListWorker(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}
	return rsp.Workers, nil
}

func (c *Client) ListRegistry(ctx context.Context) ([]*api.Entity, []*api.Echo, []*api.Step, error) {
	in := &api.ListRegistryRequest{}
	rsp, err := c.s.ListRegistry(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return nil, nil, nil, verrs.FromErr(err)
	}
	return rsp.Entities, rsp.Echoes, rsp.Steps, nil
}

func (c *Client) ListWorkflow(ctx context.Context) ([]*api.BpmnResource, error) {
	rsp, err := c.s.ListWorkflow(ctx, &api.ListWorkflowRequest{}, c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}
	return rsp.Resources, nil
}

func (c *Client) GetWorkflow(ctx context.Context, id string) (*api.BpmnResource, error) {
	rsp, err := c.s.GetWorkflow(ctx, &api.GetWorkflowRequest{Id: id}, c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}
	return rsp.Resource, nil
}

func (c *Client) DeployWorkflow(ctx context.Context, resource *api.BpmnResource) (int64, error) {
	rsp, err := c.s.DeployWorkflow(ctx, &api.DeployWorkflowRequest{Resource: resource}, c.cfg.callOptions()...)
	if err != nil {
		return 0, err
	}
	return rsp.Key, nil
}

func (c *Client) ListWorkFlowInstance(ctx context.Context) ([]*api.WorkflowSnapshot, error) {
	rsp, err := c.s.ListWorkflowInstance(ctx, &api.ListWorkflowInstanceRequest{}, c.cfg.callOptions()...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}
	return rsp.Snapshots, nil
}

func (c *Client) NewWorkflow(opts ...Option) *WorkflowBuilder {
	return NewBuilder(opts...)
}

type runWorkflowWatcher struct {
	stream api.FlowRpc_RunWorkflowInstanceService
}

func (w *runWorkflowWatcher) Next() (*api.WorkflowWatchResult, error) {
	rsp, err := w.stream.Recv()
	if err != nil {
		return nil, verrs.FromErr(err)
	}
	if rsp.Result != nil && rsp.Result.Type == api.EventType_ET_RESULT {
		// workflow finished
		return nil, io.EOF
	}
	return rsp.Result, nil
}

func (c *Client) ExecuteWorkflowInstance(ctx context.Context, id, name string, watch bool) (WorkflowWatcher, error) {
	in := &api.RunWorkflowInstanceRequest{
		Id:    id,
		Name:  name,
		Watch: watch,
	}
	stream, err := c.s.RunWorkflowInstance(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}

	if !watch {
		return nil, nil
	}

	return &runWorkflowWatcher{stream: stream}, nil
}

func (c *Client) InspectWorkflowInstance(ctx context.Context, wid string) (*api.Workflow, error) {
	in := &api.InspectWorkflowInstanceRequest{Wid: wid}
	rsp, err := c.s.InspectWorkflowInstance(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}
	return rsp.Workflow, nil
}

func (c *Client) AbortWorkflowInstance(ctx context.Context, wid string) error {
	in := &api.AbortWorkflowInstanceRequest{Wid: wid}
	_, err := c.s.AbortWorkflowInstance(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return verrs.FromErr(err)
	}
	return nil
}

func (c *Client) PauseWorkflowInstance(ctx context.Context, wid string) error {
	in := &api.PauseWorkflowInstanceRequest{Wid: wid}
	_, err := c.s.PauseWorkflowInstance(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return verrs.FromErr(err)
	}
	return nil
}

func (c *Client) ResumeWorkflowInstance(ctx context.Context, wid string) error {
	in := &api.ResumeWorkflowInstanceRequest{Wid: wid}
	_, err := c.s.ResumeWorkflowInstance(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return verrs.FromErr(err)
	}
	return nil
}

type workflowWatcher struct {
	stream api.FlowRpc_WatchWorkflowInstanceService
}

func (w *workflowWatcher) Next() (*api.WorkflowWatchResult, error) {
	rsp, err := w.stream.Recv()
	if err != nil {
		return nil, verrs.FromErr(err)
	}
	if rsp.Result.Type == api.EventType_ET_RESULT {
		// workflow finished
		return nil, io.EOF
	}
	return rsp.Result, nil
}

func (c *Client) WatchWorkflowInstance(ctx context.Context, wid string, opts ...vclient.CallOption) (WorkflowWatcher, error) {
	in := &api.WatchWorkflowInstanceRequest{
		Wid: wid,
		Cid: c.Id(),
	}

	opts = append(c.cfg.callOptions(), opts...)
	stream, err := c.s.WatchWorkflowInstance(ctx, in, opts...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}

	return &workflowWatcher{stream: stream}, nil
}

func (c *Client) ListInteractive(ctx context.Context, pid string, opts ...vclient.CallOption) ([]*api.Interactive, error) {
	in := &api.ListInteractiveRequest{
		Pid: pid,
	}

	opts = append(c.cfg.callOptions(), opts...)
	rsp, err := c.s.ListInteractive(ctx, in, opts...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}

	return rsp.Interactive, nil
}

func (c *Client) CommitInteractive(ctx context.Context, pid, sid string, properties map[string]string, opts ...vclient.CallOption) error {
	in := &api.CommitInteractiveRequest{
		Pid:        pid,
		Sid:        sid,
		Properties: properties,
	}

	opts = append(c.cfg.callOptions(), opts...)
	_, err := c.s.CommitInteractive(ctx, in, opts...)
	return err
}

func (c *Client) Call(ctx context.Context, client, name string, data []byte, opts ...vclient.CallOption) ([]byte, error) {
	in := &api.CallRequest{
		Id:      client,
		Name:    name,
		Request: data,
	}
	opts = append(c.cfg.callOptions(), opts...)
	rsp, err := c.s.Call(ctx, in, opts...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}

	if len(rsp.Error) != 0 {
		return nil, api.Parse(rsp.Error)
	}

	return rsp.Data, nil
}

func (c *Client) Step(ctx context.Context, name string, action api.StepAction, items, args map[string]string, data []byte, opts ...vclient.CallOption) ([]byte, error) {
	in := &api.StepRequest{
		Cid:    c.Id(),
		Name:   name,
		Action: action,
		Items:  items,
		Args:   args,
		Entity: data,
	}

	opts = append(c.cfg.callOptions(), opts...)
	rsp, err := c.s.Step(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}

	if len(rsp.Error) == 0 {
		return nil, api.Parse(rsp.Error)
	}

	return rsp.Data, nil
}

func (c *Client) NewSession() (*PipeSession, error) {
	return NewPipeSession(c)
}

type PipeSession struct {
	c       *Client
	ctx     context.Context
	cancel  context.CancelFunc
	pipe    api.FlowRpc_PipeService
	stepMap sync.Map
	// flag for connect
	cch  chan struct{}
	exit chan struct{}
}

func NewPipeSession(c *Client) (*PipeSession, error) {

	ctx, cancel := context.WithCancel(context.Background())

	s := &PipeSession{
		c:       c,
		ctx:     ctx,
		cancel:  cancel,
		stepMap: sync.Map{},
		cch:     make(chan struct{}, 1),
		exit:    make(chan struct{}, 1),
	}

	err := s.connect()
	if err != nil {
		cancel()
		return nil, err
	}

	go s.process()

	return s, nil
}

func (s *PipeSession) IsConnected() bool {
	select {
	case <-s.cch:
		return true
	default:
		return false
	}
}

func (s *PipeSession) toReconnect() {
	select {
	case <-s.cch:
		s.cch = make(chan struct{}, 1)
	default:
	}
}

func (s *PipeSession) connect() error {
	if s.IsConnected() {
		return nil
	}

	log.Infof("establish a new pipe channel to %s", s.c.cfg.address)
	pipe, err := s.c.s.Pipe(s.ctx, s.c.cfg.streamOptions()...)
	if err != nil {
		return verrs.FromErr(err)
	}

	err = pipe.Send(&api.PipeRequest{
		Id:    s.c.Id(),
		Topic: api.Topic_T_CONN,
	})
	if err != nil {
		return fmt.Errorf("build pipe connect: %s", err)
	}

	rsp, err := pipe.Recv()
	if err != nil || rsp.Topic != api.Topic_T_CONN {
		return fmt.Errorf("waitting for pipe connect reply: %v", err)
	}
	s.pipe = pipe
	go s.heartbeat()
	close(s.cch)

	return nil
}

func (s *PipeSession) heartbeat() {
	duration := s.c.cfg.heartbeat
	log.Debugf("start heartbeat ticker every %ss", duration.Seconds())
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	for {
		if s.isClosed() {
			break
		}

		select {
		case <-ticker.C:
		}

		err := s.pipe.Send(&api.PipeRequest{
			Id:    s.c.Id(),
			Topic: api.Topic_T_PING,
		})
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("send heartbeat request: %v", err)
			break
		}
	}

	log.Debugf("stop heartbeat ticker")
}

func (s *PipeSession) process() {

	var attempts int
	for {
		if s.isClosed() {
			return
		}

		// connect to pipe service
		err := s.connect()
		if err != nil {
			attempts++
			log.Errorf("error keeping pipe connect: %+v", err)
			time.Sleep(time.Duration(attempts) * time.Second)
			continue
		}

		ch := make(chan bool)

		go func() {
			select {
			case <-ch:
				//s.pipe.Close()
			case <-s.exit:
				s.pipe.Close()
			}
		}()

		// reset if we get here
		attempts = 0

		for {
			rsp, e := s.pipe.Recv()
			if e != nil {
				if e != io.EOF {
					log.Errorf("failed to get receive data: %+v", e)
				}
				close(ch)
				break
			}

			if e = s.handleRecv(rsp); e != nil {
				log.Errorf("handle receive: %v", e)
			}
		}

		s.toReconnect()
	}
}

func (s *PipeSession) handleRecv(rsp *api.PipeResponse) error {
	var err error
	revision := rsp.Revision
	switch rsp.Topic {
	case api.Topic_T_PING:
		// to nothing
	case api.Topic_T_CALL:
		data := rsp.Call
		err = s.doCall(revision, data)
	case api.Topic_T_STEP:
		data := rsp.Step
		err = s.doStep(revision, data)
	}

	return err
}

func (s *PipeSession) doCall(revision *api.Revision, data *api.PipeCallRequest) error {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	call := func(ctx context.Context, in []byte) (out []byte, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("echo panic recovered: ", r)
				log.Error(string(debug.Stack()))
				err = api.ErrClientException("call panic recovered: %v", r)
			}
		}()
		echo, e := s.c.cfg.store.PopulateEcho(data.Name)
		if e != nil {
			err = e
			log.Error(err)
			return
		}

		out, err = echo.Call(ctx, in)
		return
	}

	out, err := call(ctx, data.Data)

	callRsp := &api.PipeCallResponse{
		Name: data.Name,
		Data: out,
	}
	if err != nil {
		callRsp.Error = api.FromErr(err).Detail
	}

	e := s.pipe.Send(&api.PipeRequest{
		Id:       s.c.Id(),
		Topic:    api.Topic_T_CALL,
		Revision: revision,
		Call:     callRsp,
	})

	if err != nil {
		return err
	}
	if e != nil {
		return e
	}

	return nil
}

func (s *PipeSession) doStep(revision *api.Revision, data *api.PipeStepRequest) error {

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	pCtx := NewSessionCtx(ctx, data.Wid, data.Name, *revision, s.c)
	do := func(ctx *PipeSessionCtx) (out []byte, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("step panic recovered: ", r)
				log.Error(string(debug.Stack()))
				err = api.ErrClientException("step panic recovered: %v", r)
			}
		}()

		var step Step
		sid := path.Join(data.Wid, data.Name)
		if data.Action == api.StepAction_SC_PREPARE {
			step, err = s.c.cfg.store.PopulateStep(data.Name)
		} else {
			var v any
			var ok bool
			v, ok = s.stepMap.Load(sid)
			if !ok {
				err = fmt.Errorf("not found Step<%s>", data.Name)
			} else {
				step = v.(Step)
			}
		}

		if err != nil {
			log.Error(err)
			return
		}

		e := InjectTypeFields(step, data.Items, data.Args, data.Entity)
		if e != nil {
			err = fmt.Errorf("inject step field: %v", e)
			log.Error(err)
			return
		}

		switch data.Action {
		case api.StepAction_SC_PREPARE:
			err = step.Prepare(ctx)
			if err == nil {
				s.stepMap.Store(sid, step)
			}
		case api.StepAction_SC_COMMIT:
			err = step.Commit(ctx)
		case api.StepAction_SC_ROLLBACK:
			err = step.Rollback(ctx)
		case api.StepAction_SC_CANCEL:
			err = step.Cancel(ctx)
			s.stepMap.Delete(sid)
		}

		if err != nil {
			return
		}

		out, _ = ExtractTypeField(step)

		return
	}

	var b []byte
	var err error

	log.Infof("[%s] workflow %s do step %s", data.Action.Readably(), data.Wid, data.Name)
	b, err = do(pCtx)

	rsp := &api.PipeStepResponse{
		Name: data.Name,
		Data: b,
	}

	if err != nil {
		rsp.Error = api.FromErr(err).Detail
	}
	e := s.pipe.Send(&api.PipeRequest{
		Id:       s.c.Id(),
		Topic:    api.Topic_T_STEP,
		Revision: revision,
		Step:     rsp,
	})

	if err != nil {
		return err
	}

	if e != nil {
		return e
	}

	return nil
}

func (s *PipeSession) isClosed() bool {
	select {
	case <-s.exit:
		return true
	default:
		return false
	}
}

func (s *PipeSession) Close() {
	s.cancel()
	close(s.exit)
}

type PipeSessionCtx struct {
	context.Context

	wid      string
	step     string
	revision *api.Revision

	c *Client
}

func NewSessionCtx(ctx context.Context, wid, step string, revision api.Revision, c *Client) *PipeSessionCtx {
	cc := &PipeSessionCtx{
		Context:  ctx,
		wid:      wid,
		step:     step,
		revision: &revision,
		c:        c,
	}

	return cc
}

func (c *PipeSessionCtx) WorkflowID() string {
	return c.wid
}

func (c *PipeSessionCtx) Revision() *api.Revision {
	return c.revision
}

func (c *PipeSessionCtx) Call(ctx context.Context, target, echo string, data []byte, opts ...vclient.CallOption) ([]byte, error) {
	in := &api.CallRequest{
		Id:      target,
		Name:    echo,
		Request: data,
	}
	opts = append(c.c.cfg.callOptions(), opts...)
	rsp, err := c.c.s.Call(ctx, in, opts...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}

	if rsp.Error != "" {
		return nil, api.Parse(rsp.Error)
	}

	return rsp.Data, nil
}

func (c *PipeSessionCtx) Get(ctx context.Context, key string, opts ...vclient.CallOption) ([]byte, error) {
	in := &api.StepGetRequest{
		Wid:  c.wid,
		Step: c.step,
		Key:  key,
	}
	opts = append(c.c.cfg.callOptions(), opts...)
	rsp, err := c.c.s.StepGet(ctx, in, opts...)
	if err != nil {
		return nil, verrs.FromErr(err)
	}

	return rsp.Value, nil
}

func (c *PipeSessionCtx) Put(ctx context.Context, key string, data any, opts ...vclient.CallOption) error {
	var err error
	var vv string
	switch tv := data.(type) {
	case []byte:
		vv = string(tv)
	case string:
		vv = tv
	default:
		data, _ := json.Marshal(data)
		vv = string(data)
	}

	in := &api.StepPutRequest{
		Wid:   c.wid,
		Step:  c.step,
		Key:   key,
		Value: vv,
	}
	opts = append(c.c.cfg.callOptions(), opts...)
	_, err = c.c.s.StepPut(ctx, in, opts...)
	if err != nil {
		return verrs.FromErr(err)
	}
	return nil
}

func (c *PipeSessionCtx) Trace(ctx context.Context, text []byte, opts ...vclient.CallOption) error {
	in := &api.StepTraceRequest{
		Wid:  c.wid,
		Step: c.step,
		Text: text,
	}
	opts = append(c.c.cfg.callOptions(), opts...)
	_, err := c.c.s.StepTrace(ctx, in, opts...)
	if err != nil {
		return verrs.FromErr(err)
	}
	return nil
}
