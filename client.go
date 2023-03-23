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
	"io"
	"reflect"
	"runtime/debug"
	"time"

	json "github.com/json-iterator/go"
	"github.com/vine-io/flow/api"
	vclient "github.com/vine-io/vine/core/client"
	"github.com/vine-io/vine/core/client/grpc"
	log "github.com/vine-io/vine/lib/logger"
)

var gStore = NewClientStore()

type ClientStore struct {
	entitySet map[string]Entity
	echoSet   map[string]Echo
	stepSet   map[string]Step
}

func NewClientStore() *ClientStore {
	return &ClientStore{entitySet: map[string]Entity{}, echoSet: map[string]Echo{}, stepSet: map[string]Step{}}
}

func Load(tps ...any) {
	for _, t := range tps {
		load(t)
	}
}

func load(t any) {
	kind := GetTypePkgName(reflect.TypeOf(t))
	switch tt := t.(type) {
	case Entity:
		gStore.entitySet[kind] = tt
	case Echo:
		gStore.echoSet[kind] = tt
	case Step:
		gStore.stepSet[kind] = tt
	}
}

func (s *ClientStore) GetEntity(kind string) (Entity, bool) {
	e, ok := s.entitySet[kind]
	return e, ok
}

func (s *ClientStore) GetEcho(name string) (Echo, bool) {
	e, ok := s.echoSet[name]
	return e, ok
}

func (s *ClientStore) GetStep(name string) (Step, bool) {
	step, ok := s.stepSet[name]
	return step, ok
}

type ClientConfig struct {
	name    string
	id      string
	address string
	timeout time.Duration
	conn    vclient.Client
}

func NewConfig(name, id, address string) ClientConfig {
	c := ClientConfig{
		name:    name,
		id:      id,
		address: address,
		timeout: time.Second * 30,
		conn:    grpc.NewClient(),
	}
	return c
}

func (c *ClientConfig) callOptions() []vclient.CallOption {
	return []vclient.CallOption{vclient.WithAddress(c.address)}
}

type WorkflowWatcher interface {
	Next() (*api.WorkflowWatchResult, error)
}

type Client struct {
	cfg ClientConfig
	s   api.FlowRpcService
}

func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.conn == nil {
		cfg.conn = grpc.NewClient()
	}

	if err := cfg.conn.Init(); err != nil {
		return nil, err
	}

	var err error
	service := api.NewFlowRpcService(cfg.name, cfg.conn)
	ctx := context.Background()

	cc := &api.Client{
		Id: cfg.id,
	}

	entities := make([]*api.Entity, 0)
	for _, item := range gStore.entitySet {
		e := EntityToAPI(item)
		e.Clients = map[string]*api.Client{cc.Id: cc}
		entities = append(entities, e)
	}
	echoes := make([]*api.Echo, 0)
	for _, item := range gStore.echoSet {
		echo := EchoToAPI(item)
		echo.Clients = map[string]*api.Client{cc.Id: cc}
		echoes = append(echoes, echo)
	}
	steps := make([]*api.Step, 0)
	for _, item := range gStore.stepSet {
		step := StepToAPI(item)
		step.Clients = map[string]*api.Client{cc.Id: cc}
		steps = append(steps, step)
	}
	in := &api.RegisterRequest{
		Id:       cfg.id,
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

func (c *Client) ListWorkFlow(ctx context.Context) ([]*api.WorkflowSnapshot, error) {
	rsp, err := c.s.ListWorkflow(ctx, &api.ListWorkflowRequest{}, c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}
	return rsp.Snapshots, nil
}

func (c *Client) InspectWorkflow(ctx context.Context, wid string) (*api.Workflow, error) {
	in := &api.InspectWorkflowRequest{Wid: wid}
	rsp, err := c.s.InspectWorkflow(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}
	return rsp.Workflow, nil
}

func (c *Client) AbortWorkflow(ctx context.Context, wid string) error {
	in := &api.AbortWorkflowRequest{Wid: wid}
	_, err := c.s.AbortWorkflow(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) PauseWorkflow(ctx context.Context, wid string) error {
	in := &api.PauseWorkflowRequest{Wid: wid}
	_, err := c.s.PauseWorkflow(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ResumeWorkflow(ctx context.Context, wid string) error {
	in := &api.ResumeWorkflowRequest{Wid: wid}
	_, err := c.s.ResumeWorkflow(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return err
	}
	return nil
}

type workflowWatcher struct {
	stream api.FlowRpc_WatchWorkflowService
}

func (w *workflowWatcher) Next() (*api.WorkflowWatchResult, error) {
	rsp, err := w.stream.Recv()
	if err != nil {
		return nil, err
	}
	if rsp.Result.Type == api.EventType_ET_RESULT {
		// workflow finished
		return nil, io.EOF
	}
	return rsp.Result, nil
}

func (c *Client) WatchWorkflow(ctx context.Context, wid string) (WorkflowWatcher, error) {
	in := &api.WatchWorkflowRequest{
		Wid: wid,
		Cid: c.Id(),
	}
	stream, err := c.s.WatchWorkflow(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}

	return &workflowWatcher{stream: stream}, nil
}

func (c *Client) Call(ctx context.Context, client, name string, data []byte) ([]byte, error) {
	in := &api.CallRequest{
		Id:      client,
		Name:    name,
		Request: data,
	}
	rsp, err := c.s.Call(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}

	if len(rsp.Error) != 0 {
		return nil, errors.New(rsp.Error)
	}

	return rsp.Data, nil
}

func (c *Client) Step(ctx context.Context, name string, action api.StepAction, items map[string][]byte, data []byte) ([]byte, error) {
	in := &api.StepRequest{
		Cid:    c.Id(),
		Name:   name,
		Action: action,
		Items:  items,
		Entity: data,
	}
	rsp, err := c.s.Step(ctx, in, c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}

	if len(rsp.Error) == 0 {
		return nil, errors.New(rsp.Error)
	}

	return rsp.Data, nil
}

func (c *Client) NewSession() (*PipeSession, error) {
	return NewPipeSession(c)
}

type PipeSession struct {
	c      *Client
	ctx    context.Context
	cancel context.CancelFunc
	pipe   api.FlowRpc_PipeService
	// flag for connect
	cch  chan struct{}
	exit chan struct{}
}

func NewPipeSession(c *Client) (*PipeSession, error) {

	ctx, cancel := context.WithCancel(context.Background())

	s := &PipeSession{
		c:      c,
		ctx:    ctx,
		cancel: cancel,
		cch:    make(chan struct{}, 1),
		exit:   make(chan struct{}, 1),
	}

	err := s.connect()
	if err != nil {
		cancel()
		return nil, err
	}

	go s.process()

	return s, nil
}

func (s *PipeSession) NewWorkflow(opts ...Option) *WorkflowBuilder {
	return NewBuilder(opts...)
}

type runWorkflowWatcher struct {
	stream api.FlowRpc_RunWorkflowService
}

func (w *runWorkflowWatcher) Next() (*api.WorkflowWatchResult, error) {
	rsp, err := w.stream.Recv()
	if err != nil {
		return nil, err
	}
	if rsp.Result.Type == api.EventType_ET_RESULT {
		// workflow finished
		return nil, io.EOF
	}
	return rsp.Result, nil
}

func (s *PipeSession) ExecuteWorkflow(ctx context.Context, spec *api.Workflow, watch bool) (WorkflowWatcher, error) {
	in := &api.RunWorkflowRequest{
		Workflow: spec,
		Watch:    watch,
	}
	stream, err := s.c.s.RunWorkflow(ctx, in, s.c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}

	if !watch {
		return nil, nil
	}

	return &runWorkflowWatcher{stream: stream}, nil
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

	pipe, err := s.c.s.Pipe(s.ctx)
	if err != nil {
		return err
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
	close(s.cch)

	return nil
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
				s.pipe.Close()
			case <-s.exit:
				s.pipe.Close()
			}
		}()

		// reset if we get here
		attempts = 0

		for {
			rsp, e := s.pipe.Recv()
			if e != nil {
				log.Errorf("failed to get receive data: %+v", e)
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
	echo, ok := gStore.GetEcho(data.Name)
	if !ok {
		return fmt.Errorf("not found Echo<%s>", data.Name)
	}

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	call := func(ctx context.Context, in []byte) (out []byte, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("echo panic recovered: ", r)
				log.Error(string(debug.Stack()))
				err = api.ClientException("call panic recovered: %v", r)
			}
		}()
		out, err = echo.Call(ctx, in)
		return
	}

	out, err := call(ctx, data.Data)

	callRsp := &api.PipeCallResponse{
		Name: data.Name,
		Data: out,
	}
	if err != nil {
		callRsp.Error = api.FromErr(err).Error()
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
	step, ok := gStore.GetStep(data.Name)
	if !ok {
		return fmt.Errorf("not found Step<%s>", data.Name)
	}

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	pCtx := NewSessionCtx(ctx, data.Wid, data.Name, *revision, s.c)
	var err error

	e := InjectTypeFields(step, data.Items, data.Entity)
	if e != nil {
		log.Errorf("inject step field: %v", e)
	}

	do := func(ctx *PipeSessionCtx, step Step) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("step panic recovered: ", r)
				log.Error(string(debug.Stack()))
				err = api.ClientException("step panic recovered: %v", r)
			}
		}()

		switch data.Action {
		case api.StepAction_SC_PREPARE:
			err = step.Prepare(ctx)
		case api.StepAction_SC_COMMIT:
			err = step.Commit(ctx)
		case api.StepAction_SC_ROLLBACK:
			err = step.Rollback(ctx)
		case api.StepAction_SC_CANCEL:
			err = step.Cancel(ctx)
		}
		return
	}

	err = do(pCtx, step)

	b, e := ExtractTypeField(step)
	if e != nil {
		log.Fatal("extract step entity data: %v", e)
	}

	rsp := &api.PipeStepResponse{
		Name: data.Name,
		Data: b,
	}

	if err != nil {
		rsp.Error = api.FromErr(err).Error()
	}
	e = s.pipe.Send(&api.PipeRequest{
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

func (c *PipeSessionCtx) Call(ctx context.Context, data []byte) ([]byte, error) {
	in := &api.CallRequest{
		Id:      c.c.Id(),
		Request: data,
	}
	rsp, err := c.c.s.Call(ctx, in, c.c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}

	if rsp.Error != "" {
		return nil, errors.New(rsp.Error)
	}

	return rsp.Data, nil
}

func (c *PipeSessionCtx) Get(ctx context.Context, key string) ([]byte, error) {
	in := &api.StepGetRequest{
		Wid:  c.wid,
		Step: c.step,
		Key:  key,
	}
	rsp, err := c.c.s.StepGet(ctx, in, c.c.cfg.callOptions()...)
	if err != nil {
		return nil, err
	}

	return rsp.Value, nil
}

func (c *PipeSessionCtx) Put(ctx context.Context, key string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	in := &api.StepPutRequest{
		Wid:   c.wid,
		Step:  c.step,
		Key:   key,
		Value: string(b),
	}
	_, err = c.c.s.StepPut(ctx, in, c.c.cfg.callOptions()...)
	return err
}

func (c *PipeSessionCtx) Trace(ctx context.Context, text []byte) error {
	in := &api.StepTraceRequest{
		Wid:  c.wid,
		Step: c.step,
		Text: text,
	}
	_, err := c.c.s.StepTrace(ctx, in, c.c.cfg.callOptions()...)
	return err
}
