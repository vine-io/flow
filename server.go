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
	"strings"

	"github.com/vine-io/flow/api"
	vserver "github.com/vine-io/vine/core/server"
	verrs "github.com/vine-io/vine/lib/errors"
	log "github.com/vine-io/vine/lib/logger"
	"google.golang.org/grpc/peer"
)

var _ api.FlowRpcHandler = (*RpcServer)(nil)

type RpcServer struct {
	s         vserver.Server
	ps        *PipeSet
	scheduler *Scheduler
}

func NewRPCServer(s vserver.Server, scheduler *Scheduler) (*RpcServer, error) {
	rpc := &RpcServer{s: s, ps: NewPipeSet(), scheduler: scheduler}

	err := api.RegisterFlowRpcHandler(s, rpc)
	if err != nil {
		return nil, err
	}

	return rpc, nil
}

func (rs *RpcServer) Id() string {
	return rs.s.Options().Id
}

func (rs *RpcServer) Register(ctx context.Context, req *api.RegisterRequest, rsp *api.RegisterResponse) error {
	var endpoint string
	pr, ok := peer.FromContext(ctx)
	if ok {
		endpoint = strings.Split(pr.Addr.String(), ":")[0]
	}

	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	cm := map[string]*api.Client{req.Id: {Id: req.Id, Endpoint: endpoint}}

	entities := make([]*api.Entity, 0, len(req.Entities))
	for i := range req.Entities {
		entity := req.Entities[i]
		entity.Clients = cm
		entities[i] = entity
	}

	echoes := make([]*api.Echo, 0, len(req.Echoes))
	for i := range req.Echoes {
		echo := req.Echoes[i]
		echo.Clients = cm
		echoes[i] = echo
	}

	steps := make([]*api.Step, 0, len(req.Steps))
	for i := range req.Steps {
		step := req.Steps[i]
		step.Clients = cm
		steps[i] = step
	}

	rs.scheduler.Register(entities, echoes, steps)
	return nil
}

func (rs *RpcServer) Call(ctx context.Context, req *api.CallRequest, rsp *api.CallResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	pipe, ok := rs.ps.Get(req.Id)
	if !ok {
		return verrs.PreconditionFailed(rs.Id(), "client %s not exists:", req.Id)
	}

	pack := NewCall(ctx, req.Request)
	defer pack.Destroy()
	result, ech := pipe.Call(pack)

	select {
	case <-ctx.Done():
		return verrs.Timeout(rs.Id(), "request timeout")
	case e := <-ech:
		return verrs.InternalServerError(rs.Id(), "%v", e)
	case rsp.Response = <-result:
	}

	return nil
}

func (rs *RpcServer) Step(ctx context.Context, req *api.StepRequest, rsp *api.StepResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	pipe, ok := rs.ps.Get(req.Id)
	if !ok {
		return verrs.PreconditionFailed(rs.Id(), "client %s not exists:", req.Id)
	}

	pack := NewStep(ctx, req.Action)
	defer pack.Destroy()
	result, ech := pipe.Step(pack)

	select {
	case <-ctx.Done():
		return verrs.Timeout(rs.Id(), "request timeout")
	case e := <-ech:
		return verrs.InternalServerError(rs.Id(), "%v", e)
	case rsp.Response = <-result:
	}

	return nil
}

func (rs *RpcServer) Pipe(ctx context.Context, stream api.FlowRpc_PipeStream) error {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return verrs.BadRequest(rs.Id(), "the peer of client is empty")
	}

	req, err := stream.Recv()
	if err != nil {
		return verrs.BadRequest(rs.Id(), "confirm client info: %v", err)
	}
	p := NewPipe(req.Id, pr, stream)
	defer p.Close()
	go p.Start()

	rs.ps.Add(p)
	defer rs.ps.Del(p)

	select {
	case <-ctx.Done():
		log.Info("client pipe <%s,%s> closed", p.Id, p.pr.Addr.String())
	}

	return nil
}

func (rs *RpcServer) ListWorkflow(ctx context.Context, req *api.ListWorkflowRequest, rsp *api.ListWorkflowResponse) error {
	snapshots := rs.scheduler.WorkflowSnapshots()
	rsp.Snapshots = snapshots
	return nil
}

func (rs *RpcServer) RunWorkflow(ctx context.Context, req *api.RunWorkflowRequest, stream api.FlowRpc_RunWorkflowStream) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	err := rs.scheduler.ExecuteWorkflow(req.Workflow, rs.ps)
	if err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	if !req.Watch {
		return nil
	}

	ch, err := rs.scheduler.WatchWorkflow(ctx, req.Workflow.Option.String())
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case result := <-ch:
			rsp := &api.RunWorkflowResponse{Result: result}
			err = stream.Send(rsp)
			if err != nil {
				return err
			}
		}
	}
}

func (rs *RpcServer) InspectWorkflow(ctx context.Context, req *api.InspectWorkflowRequest, rsp *api.InspectWorkflowResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	w, err := rs.scheduler.InspectWorkflow(ctx, req.Wid)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	rsp.Workflow = w
	return nil
}

func (rs *RpcServer) AbortWorkflow(ctx context.Context, req *api.AbortWorkflowRequest, rsp *api.AbortWorkflowResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	w, ok := rs.scheduler.GetWorkflow(req.Wid)
	if !ok {
		return verrs.BadRequest(rs.Id(), "workflow not found")
	}

	w.Cancel()
	return nil
}

func (rs *RpcServer) WatchWorkflow(ctx context.Context, req *api.WatchWorkflowRequest, stream api.FlowRpc_WatchWorkflowStream) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	ch, err := rs.scheduler.WatchWorkflow(ctx, req.Wid)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case result := <-ch:
			rsp := &api.WatchWorkflowResponse{Result: result}
			err = stream.Send(rsp)
			if err != nil {
				return err
			}
		}
	}
}

func (rs *RpcServer) StepGet(ctx context.Context, req *api.StepGetRequest, rsp *api.StepGetResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	data, err := rs.scheduler.StepGet(ctx, req.Wid, req.Key)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	rsp.Value = data
	return nil
}

func (rs *RpcServer) StepPut(ctx context.Context, req *api.StepPutRequest, rsp *api.StepPutResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	err := rs.scheduler.StepPut(ctx, req.Wid, req.Key, req.Value)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}
	return nil
}

func (rs *RpcServer) StepTrace(ctx context.Context, req *api.StepTraceRequest, rsp *api.StepTraceResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	err := rs.scheduler.StepTrace(ctx, req.Wid, req.Step, req.Text)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	return nil
}
