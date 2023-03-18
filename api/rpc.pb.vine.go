// Code generated by proto-gen-vine. DO NOT EDIT.
// source: github.com/vine-io/flow/api/rpc.proto

package api

import (
	context "context"
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	client "github.com/vine-io/vine/core/client"
	server "github.com/vine-io/vine/core/server"
	api "github.com/vine-io/vine/lib/api"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// API Endpoints for FlowRpc service
func NewFlowRpcEndpoints() []api.Endpoint {
	return []api.Endpoint{}
}

// Client API for FlowRpc service
type FlowRpcService interface {
	Register(ctx context.Context, in *RegisterRequest, opts ...client.CallOption) (*RegisterResponse, error)
	Call(ctx context.Context, in *CallRequest, opts ...client.CallOption) (*CallResponse, error)
	Step(ctx context.Context, in *StepRequest, opts ...client.CallOption) (*StepResponse, error)
	Pipe(ctx context.Context, opts ...client.CallOption) (FlowRpc_PipeService, error)
	ListWorkflow(ctx context.Context, in *ListWorkflowRequest, opts ...client.CallOption) (*ListWorkflowResponse, error)
	RunWorkflow(ctx context.Context, in *RunWorkflowRequest, opts ...client.CallOption) (FlowRpc_RunWorkflowService, error)
	InspectWorkflow(ctx context.Context, in *InspectWorkflowRequest, opts ...client.CallOption) (*InspectWorkflowResponse, error)
	AbortWorkflow(ctx context.Context, in *AbortWorkflowRequest, opts ...client.CallOption) (*AbortWorkflowResponse, error)
	WatchWorkflow(ctx context.Context, in *WatchWorkflowRequest, opts ...client.CallOption) (FlowRpc_WatchWorkflowService, error)
	StepGet(ctx context.Context, in *StepGetRequest, opts ...client.CallOption) (*StepGetResponse, error)
	StepPut(ctx context.Context, in *StepPutRequest, opts ...client.CallOption) (*StepPutResponse, error)
	StepTrace(ctx context.Context, in *StepTraceRequest, opts ...client.CallOption) (*StepTraceResponse, error)
}

type flowRpcService struct {
	c    client.Client
	name string
}

func NewFlowRpcService(name string, c client.Client) FlowRpcService {
	return &flowRpcService{
		c:    c,
		name: name,
	}
}

func (c *flowRpcService) Register(ctx context.Context, in *RegisterRequest, opts ...client.CallOption) (*RegisterResponse, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.Register", in)
	out := new(RegisterResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowRpcService) Call(ctx context.Context, in *CallRequest, opts ...client.CallOption) (*CallResponse, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.Call", in)
	out := new(CallResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowRpcService) Step(ctx context.Context, in *StepRequest, opts ...client.CallOption) (*StepResponse, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.Step", in)
	out := new(StepResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowRpcService) Pipe(ctx context.Context, opts ...client.CallOption) (FlowRpc_PipeService, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.Pipe", &PipeRequest{})
	stream, err := c.c.Stream(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	return &flowRpcServicePipe{stream}, nil
}

type FlowRpc_PipeService interface {
	Context() context.Context
	SendMsg(interface{}) error
	RecvMsg(interface{}) error
	Close() error
	Send(*PipeRequest) error
	Recv() (*PipeResponse, error)
}

type flowRpcServicePipe struct {
	stream client.Stream
}

func (x *flowRpcServicePipe) Close() error {
	return x.stream.Close()
}

func (x *flowRpcServicePipe) Context() context.Context {
	return x.stream.Context()
}

func (x *flowRpcServicePipe) SendMsg(m interface{}) error {
	return x.stream.Send(m)
}

func (x *flowRpcServicePipe) RecvMsg(m interface{}) error {
	return x.stream.Recv(m)
}

func (x *flowRpcServicePipe) Send(m *PipeRequest) error {
	return x.stream.Send(m)
}

func (x *flowRpcServicePipe) Recv() (*PipeResponse, error) {
	m := new(PipeResponse)
	err := x.stream.Recv(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *flowRpcService) ListWorkflow(ctx context.Context, in *ListWorkflowRequest, opts ...client.CallOption) (*ListWorkflowResponse, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.ListWorkflow", in)
	out := new(ListWorkflowResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowRpcService) RunWorkflow(ctx context.Context, in *RunWorkflowRequest, opts ...client.CallOption) (FlowRpc_RunWorkflowService, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.RunWorkflow", &RunWorkflowRequest{})
	stream, err := c.c.Stream(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(in); err != nil {
		return nil, err
	}
	return &flowRpcServiceRunWorkflow{stream}, nil
}

type FlowRpc_RunWorkflowService interface {
	Context() context.Context
	SendMsg(interface{}) error
	RecvMsg(interface{}) error
	Close() error
	Recv() (*RunWorkflowResponse, error)
}

type flowRpcServiceRunWorkflow struct {
	stream client.Stream
}

func (x *flowRpcServiceRunWorkflow) Close() error {
	return x.stream.Close()
}

func (x *flowRpcServiceRunWorkflow) Context() context.Context {
	return x.stream.Context()
}

func (x *flowRpcServiceRunWorkflow) SendMsg(m interface{}) error {
	return x.stream.Send(m)
}

func (x *flowRpcServiceRunWorkflow) RecvMsg(m interface{}) error {
	return x.stream.Recv(m)
}

func (x *flowRpcServiceRunWorkflow) Recv() (*RunWorkflowResponse, error) {
	m := new(RunWorkflowResponse)
	err := x.stream.Recv(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *flowRpcService) InspectWorkflow(ctx context.Context, in *InspectWorkflowRequest, opts ...client.CallOption) (*InspectWorkflowResponse, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.InspectWorkflow", in)
	out := new(InspectWorkflowResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowRpcService) AbortWorkflow(ctx context.Context, in *AbortWorkflowRequest, opts ...client.CallOption) (*AbortWorkflowResponse, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.AbortWorkflow", in)
	out := new(AbortWorkflowResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowRpcService) WatchWorkflow(ctx context.Context, in *WatchWorkflowRequest, opts ...client.CallOption) (FlowRpc_WatchWorkflowService, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.WatchWorkflow", &WatchWorkflowRequest{})
	stream, err := c.c.Stream(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(in); err != nil {
		return nil, err
	}
	return &flowRpcServiceWatchWorkflow{stream}, nil
}

type FlowRpc_WatchWorkflowService interface {
	Context() context.Context
	SendMsg(interface{}) error
	RecvMsg(interface{}) error
	Close() error
	Recv() (*WatchWorkflowResponse, error)
}

type flowRpcServiceWatchWorkflow struct {
	stream client.Stream
}

func (x *flowRpcServiceWatchWorkflow) Close() error {
	return x.stream.Close()
}

func (x *flowRpcServiceWatchWorkflow) Context() context.Context {
	return x.stream.Context()
}

func (x *flowRpcServiceWatchWorkflow) SendMsg(m interface{}) error {
	return x.stream.Send(m)
}

func (x *flowRpcServiceWatchWorkflow) RecvMsg(m interface{}) error {
	return x.stream.Recv(m)
}

func (x *flowRpcServiceWatchWorkflow) Recv() (*WatchWorkflowResponse, error) {
	m := new(WatchWorkflowResponse)
	err := x.stream.Recv(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *flowRpcService) StepGet(ctx context.Context, in *StepGetRequest, opts ...client.CallOption) (*StepGetResponse, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.StepGet", in)
	out := new(StepGetResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowRpcService) StepPut(ctx context.Context, in *StepPutRequest, opts ...client.CallOption) (*StepPutResponse, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.StepPut", in)
	out := new(StepPutResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowRpcService) StepTrace(ctx context.Context, in *StepTraceRequest, opts ...client.CallOption) (*StepTraceResponse, error) {
	req := c.c.NewRequest(c.name, "FlowRpc.StepTrace", in)
	out := new(StepTraceResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for FlowRpc service
type FlowRpcHandler interface {
	Register(context.Context, *RegisterRequest, *RegisterResponse) error
	Call(context.Context, *CallRequest, *CallResponse) error
	Step(context.Context, *StepRequest, *StepResponse) error
	Pipe(context.Context, FlowRpc_PipeStream) error
	ListWorkflow(context.Context, *ListWorkflowRequest, *ListWorkflowResponse) error
	RunWorkflow(context.Context, *RunWorkflowRequest, FlowRpc_RunWorkflowStream) error
	InspectWorkflow(context.Context, *InspectWorkflowRequest, *InspectWorkflowResponse) error
	AbortWorkflow(context.Context, *AbortWorkflowRequest, *AbortWorkflowResponse) error
	WatchWorkflow(context.Context, *WatchWorkflowRequest, FlowRpc_WatchWorkflowStream) error
	StepGet(context.Context, *StepGetRequest, *StepGetResponse) error
	StepPut(context.Context, *StepPutRequest, *StepPutResponse) error
	StepTrace(context.Context, *StepTraceRequest, *StepTraceResponse) error
}

func RegisterFlowRpcHandler(s server.Server, hdlr FlowRpcHandler, opts ...server.HandlerOption) error {
	type flowRpcImpl interface {
		Register(ctx context.Context, in *RegisterRequest, out *RegisterResponse) error
		Call(ctx context.Context, in *CallRequest, out *CallResponse) error
		Step(ctx context.Context, in *StepRequest, out *StepResponse) error
		Pipe(ctx context.Context, stream server.Stream) error
		ListWorkflow(ctx context.Context, in *ListWorkflowRequest, out *ListWorkflowResponse) error
		RunWorkflow(ctx context.Context, stream server.Stream) error
		InspectWorkflow(ctx context.Context, in *InspectWorkflowRequest, out *InspectWorkflowResponse) error
		AbortWorkflow(ctx context.Context, in *AbortWorkflowRequest, out *AbortWorkflowResponse) error
		WatchWorkflow(ctx context.Context, stream server.Stream) error
		StepGet(ctx context.Context, in *StepGetRequest, out *StepGetResponse) error
		StepPut(ctx context.Context, in *StepPutRequest, out *StepPutResponse) error
		StepTrace(ctx context.Context, in *StepTraceRequest, out *StepTraceResponse) error
	}
	type FlowRpc struct {
		flowRpcImpl
	}
	h := &flowRpcHandler{hdlr}
	endpoints := NewFlowRpcEndpoints()
	for _, ep := range endpoints {
		opts = append(opts, api.WithEndpoint(&ep))
	}
	return s.Handle(s.NewHandler(&FlowRpc{h}, opts...))
}

type flowRpcHandler struct {
	FlowRpcHandler
}

func (h *flowRpcHandler) Register(ctx context.Context, in *RegisterRequest, out *RegisterResponse) error {
	return h.FlowRpcHandler.Register(ctx, in, out)
}

func (h *flowRpcHandler) Call(ctx context.Context, in *CallRequest, out *CallResponse) error {
	return h.FlowRpcHandler.Call(ctx, in, out)
}

func (h *flowRpcHandler) Step(ctx context.Context, in *StepRequest, out *StepResponse) error {
	return h.FlowRpcHandler.Step(ctx, in, out)
}

func (h *flowRpcHandler) Pipe(ctx context.Context, stream server.Stream) error {
	return h.FlowRpcHandler.Pipe(ctx, &flowRpcPipeStream{stream})
}

type FlowRpc_PipeStream interface {
	Context() context.Context
	SendMsg(interface{}) error
	RecvMsg(interface{}) error
	Close() error
	Send(*PipeResponse) error
	Recv() (*PipeRequest, error)
}

type flowRpcPipeStream struct {
	stream server.Stream
}

func (x *flowRpcPipeStream) Close() error {
	return x.stream.Close()
}

func (x *flowRpcPipeStream) Context() context.Context {
	return x.stream.Context()
}

func (x *flowRpcPipeStream) SendMsg(m interface{}) error {
	return x.stream.Send(m)
}

func (x *flowRpcPipeStream) RecvMsg(m interface{}) error {
	return x.stream.Recv(m)
}

func (x *flowRpcPipeStream) Send(m *PipeResponse) error {
	return x.stream.Send(m)
}

func (x *flowRpcPipeStream) Recv() (*PipeRequest, error) {
	m := new(PipeRequest)
	if err := x.stream.Recv(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (h *flowRpcHandler) ListWorkflow(ctx context.Context, in *ListWorkflowRequest, out *ListWorkflowResponse) error {
	return h.FlowRpcHandler.ListWorkflow(ctx, in, out)
}

func (h *flowRpcHandler) RunWorkflow(ctx context.Context, stream server.Stream) error {
	m := new(RunWorkflowRequest)
	if err := stream.Recv(m); err != nil {
		return err
	}
	return h.FlowRpcHandler.RunWorkflow(ctx, m, &flowRpcRunWorkflowStream{stream})
}

type FlowRpc_RunWorkflowStream interface {
	Context() context.Context
	SendMsg(interface{}) error
	RecvMsg(interface{}) error
	Close() error
	Send(*RunWorkflowResponse) error
}

type flowRpcRunWorkflowStream struct {
	stream server.Stream
}

func (x *flowRpcRunWorkflowStream) Close() error {
	return x.stream.Close()
}

func (x *flowRpcRunWorkflowStream) Context() context.Context {
	return x.stream.Context()
}

func (x *flowRpcRunWorkflowStream) SendMsg(m interface{}) error {
	return x.stream.Send(m)
}

func (x *flowRpcRunWorkflowStream) RecvMsg(m interface{}) error {
	return x.stream.Recv(m)
}

func (x *flowRpcRunWorkflowStream) Send(m *RunWorkflowResponse) error {
	return x.stream.Send(m)
}

func (h *flowRpcHandler) InspectWorkflow(ctx context.Context, in *InspectWorkflowRequest, out *InspectWorkflowResponse) error {
	return h.FlowRpcHandler.InspectWorkflow(ctx, in, out)
}

func (h *flowRpcHandler) AbortWorkflow(ctx context.Context, in *AbortWorkflowRequest, out *AbortWorkflowResponse) error {
	return h.FlowRpcHandler.AbortWorkflow(ctx, in, out)
}

func (h *flowRpcHandler) WatchWorkflow(ctx context.Context, stream server.Stream) error {
	m := new(WatchWorkflowRequest)
	if err := stream.Recv(m); err != nil {
		return err
	}
	return h.FlowRpcHandler.WatchWorkflow(ctx, m, &flowRpcWatchWorkflowStream{stream})
}

type FlowRpc_WatchWorkflowStream interface {
	Context() context.Context
	SendMsg(interface{}) error
	RecvMsg(interface{}) error
	Close() error
	Send(*WatchWorkflowResponse) error
}

type flowRpcWatchWorkflowStream struct {
	stream server.Stream
}

func (x *flowRpcWatchWorkflowStream) Close() error {
	return x.stream.Close()
}

func (x *flowRpcWatchWorkflowStream) Context() context.Context {
	return x.stream.Context()
}

func (x *flowRpcWatchWorkflowStream) SendMsg(m interface{}) error {
	return x.stream.Send(m)
}

func (x *flowRpcWatchWorkflowStream) RecvMsg(m interface{}) error {
	return x.stream.Recv(m)
}

func (x *flowRpcWatchWorkflowStream) Send(m *WatchWorkflowResponse) error {
	return x.stream.Send(m)
}

func (h *flowRpcHandler) StepGet(ctx context.Context, in *StepGetRequest, out *StepGetResponse) error {
	return h.FlowRpcHandler.StepGet(ctx, in, out)
}

func (h *flowRpcHandler) StepPut(ctx context.Context, in *StepPutRequest, out *StepPutResponse) error {
	return h.FlowRpcHandler.StepPut(ctx, in, out)
}

func (h *flowRpcHandler) StepTrace(ctx context.Context, in *StepTraceRequest, out *StepTraceResponse) error {
	return h.FlowRpcHandler.StepTrace(ctx, in, out)
}
