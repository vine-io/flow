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

	"github.com/vine-io/flow/api"
	vserver "github.com/vine-io/vine/core/server"
	verrs "github.com/vine-io/vine/lib/errors"
	log "github.com/vine-io/vine/lib/logger"
	"google.golang.org/grpc/peer"
)

var _ api.FlowRpcHandler = (*RpcServer)(nil)

type RpcServer struct {
	s  vserver.Server
	ps *PipeSet
}

func NewRPCServer(s vserver.Server) (*RpcServer, error) {
	rpc := &RpcServer{s: s, ps: NewPipeSet()}
	err := api.RegisterFlowRpcHandler(s, rpc)
	if err != nil {
		return nil, err
	}

	return rpc, nil
}

func (r *RpcServer) Id() string {
	return r.s.Options().Id
}

func (r *RpcServer) Register(ctx context.Context, req *api.RegisterRequest, rsp *api.RegisterResponse) error {
	//TODO implement me
	panic("implement me")
}

func (r *RpcServer) Call(ctx context.Context, req *api.CallRequest, rsp *api.CallResponse) error {
	pipe, ok := r.ps.Get(req.Id)
	if !ok {
		return verrs.PreconditionFailed(r.Id(), "client %s not exists:", req.Id)
	}

	pack := NewCall(ctx, req.Request)
	defer pack.Destroy()
	result, ech := pipe.Call(pack)

	select {
	case <-ctx.Done():
		return verrs.Timeout(r.Id(), "request timeout")
	case rsp.Response = <-result:
	case e := <-ech:
		return verrs.InternalServerError(r.Id(), "%v", e)
	}

	return nil
}

func (r *RpcServer) Step(ctx context.Context, req *api.StepRequest, rsp *api.StepResponse) error {
	pipe, ok := r.ps.Get(req.Id)
	if !ok {
		return verrs.PreconditionFailed(r.Id(), "client %s not exists:", req.Id)
	}

	pack := NewStep(ctx, req.Action)
	defer pack.Destroy()
	result, ech := pipe.Step(pack)

	select {
	case <-ctx.Done():
		return verrs.Timeout(r.Id(), "request timeout")
	case rsp.Response = <-result:
	case e := <-ech:
		return verrs.InternalServerError(r.Id(), "%v", e)
	}

	return nil
}

func (r *RpcServer) Pipe(ctx context.Context, stream api.FlowRpc_PipeStream) error {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return verrs.BadRequest(r.Id(), "the peer of client is empty")
	}

	req, err := stream.Recv()
	if err != nil {
		return verrs.BadRequest(r.Id(), "confirm client info: %v", err)
	}
	p := NewPipe(req.Id, pr, stream)
	defer p.Close()
	go p.Start()

	r.ps.Add(p)
	defer r.ps.Del(p)

	select {
	case <-ctx.Done():
		log.Info("client pipe <%s,%s> closed", p.Id, p.pr.Addr.String())
	}

	return nil
}

func (r *RpcServer) ListWorkflow(ctx context.Context, req *api.ListWorkflowRequest, rsp *api.ListWorkflowResponse) error {
	//TODO implement me
	panic("implement me")
}

func (r *RpcServer) RunWorkflow(ctx context.Context, req *api.RunWorkflowRequest, stream api.FlowRpc_RunWorkflowStream) error {
	//TODO implement me
	panic("implement me")
}

func (r *RpcServer) InspectWorkflow(ctx context.Context, req *api.InspectWorkflowRequest, rsp *api.InspectWorkflowResponse) error {
	//TODO implement me
	panic("implement me")
}

func (r *RpcServer) AbortWorkflow(ctx context.Context, req *api.AbortWorkflowRequest, rsp *api.AbortWorkflowResponse) error {
	//TODO implement me
	panic("implement me")
}

func (r *RpcServer) WatchWorkflow(ctx context.Context, req *api.WatchWorkflowRequest, stream api.FlowRpc_WatchWorkflowStream) error {
	//TODO implement me
	panic("implement me")
}
