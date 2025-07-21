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
	"strings"

	"github.com/google/uuid"
	vserver "github.com/vine-io/vine/core/server"
	verrs "github.com/vine-io/vine/lib/errors"
	log "github.com/vine-io/vine/lib/logger"
	"google.golang.org/grpc/peer"

	"github.com/vine-io/flow/api"
)

type RpcServer struct {
	s         vserver.Server
	ps        *PipeSet
	scheduler *Scheduler

	subs *WorkerSub
}

func NewRPCServer(s vserver.Server, scheduler *Scheduler) (*RpcServer, error) {
	rpc := &RpcServer{
		s:         s,
		ps:        NewPipeSet(),
		scheduler: scheduler,
		subs:      NewWorkSub(),
	}

	err := api.RegisterFlowRpcHandler(s, rpc)
	if err != nil {
		return nil, err
	}

	return rpc, nil
}

func (rs *RpcServer) Id() string {
	return rs.s.Options().Id
}

func (rs *RpcServer) ListWorker(ctx context.Context, req *api.ListWorkerRequest, rsp *api.ListWorkerResponse) error {
	items, err := rs.scheduler.GetWorkers(ctx)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	workers := make([]*api.Worker, len(items))
	for i, item := range items {
		_, ok := rs.ps.Get(item.Id)
		worker := item.DeepCopy()
		worker.Up = ok
		workers[i] = worker
	}

	rsp.Workers = workers
	return nil
}

func (rs *RpcServer) GetWorker(ctx context.Context, req *api.GetWorkerRequest, rsp *api.GetWorkerResponse) error {
	worker, err := rs.scheduler.GetWorker(ctx, req.Id)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	_, ok := rs.ps.Get(req.Id)
	worker.Up = ok

	rsp.Worker = worker
	return nil
}

func (rs *RpcServer) ListRegistry(ctx context.Context, req *api.ListRegistryRequest, rsp *api.ListRegistryResponse) error {
	entities, echoes, steps := rs.scheduler.GetRegistry()
	rsp.Entities = entities
	rsp.Echoes = echoes
	rsp.Steps = steps
	return nil
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

	worker := &api.Worker{
		Id:       req.Id,
		Endpoint: endpoint,
		Attrs:    req.Attrs,
	}
	workers := map[string]*api.Worker{req.Id: worker}

	entities := make([]*api.Entity, len(req.Entities))
	for i := range req.Entities {
		entity := req.Entities[i]
		entity.Workers = workers
		entities[i] = entity
	}

	echoes := make([]*api.Echo, len(req.Echoes))
	for i := range req.Echoes {
		echo := req.Echoes[i]
		echo.Workers = workers
		echoes[i] = echo
	}

	steps := make([]*api.Step, len(req.Steps))
	for i := range req.Steps {
		step := req.Steps[i]
		step.Workers = workers
		steps[i] = step
	}

	err := rs.scheduler.Register(worker, entities, echoes, steps)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), "register worker: %v", err)
	}
	return nil
}

func (rs *RpcServer) WorkHook(ctx context.Context, req *api.WorkHookRequest, stream api.FlowRpc_WorkHookStream) error {
	uid := uuid.New().String()
	rs.subs.Add(uid, stream)

	select {
	case <-ctx.Done():
	}

	rs.subs.Del(uid)
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

	pack := NewCall(ctx, &api.PipeCallRequest{
		Name: req.Name,
		Data: req.Request,
	})
	defer pack.Destroy()
	result, ech := pipe.Call(pack)

	select {
	case <-ctx.Done():
		return verrs.Timeout(rs.Id(), "request timeout")
	case err := <-ech:
		e := api.FromErr(err)
		return verrs.InternalServerError(rs.Id(), "%v", e.Detail)
	case data := <-result:
		rsp.Name = req.Name
		rsp.Data = data
	}

	return nil
}

func (rs *RpcServer) Step(ctx context.Context, req *api.StepRequest, rsp *api.StepResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	pipe, ok := rs.ps.Get(req.Cid)
	if !ok {
		return verrs.PreconditionFailed(rs.Id(), "client %s not exists:", req.Cid)
	}

	pack := NewStep(ctx, &api.PipeStepRequest{
		Name:   req.Name,
		Action: req.Action,
		Items:  req.Items,
	})
	defer pack.Destroy()
	result, ech := pipe.Step(pack)

	select {
	case <-ctx.Done():
		return verrs.Timeout(rs.Id(), "request timeout")
	case err := <-ech:
		e := api.FromErr(err)
		return verrs.InternalServerError(rs.Id(), "%v", e.Detail)
	case data := <-result:
		rsp.Name = req.Name
		rsp.Data = data
	}

	return nil
}

func (rs *RpcServer) Pipe(ctx context.Context, stream api.FlowRpc_PipeStream) error {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return verrs.BadRequest(rs.Id(), "the peer of worker is empty")
	}
	endpoint := pr.Addr.String()

	log.Debugf("discovery new pipe connect")

	req, err := stream.Recv()
	if err != nil {
		return verrs.BadRequest(rs.Id(), "confirm worker info: %v", err)
	}
	if req.Id == "" || req.Topic != api.Topic_T_CONN {
		return verrs.BadRequest(rs.Id(), "invalid request data")
	}

	pipe, ok := rs.ps.Get(req.Id)
	if ok && pipe.pr.Client == endpoint {
		return verrs.Conflict(rs.Id(), "worker <%s,%s> already registered", req.Id, endpoint)
	}

	log.Infof("receive pipe connect request from <%s,%s>", req.Id, endpoint)
	log.Debugf("send pipe reply to <%s,%s>", req.Id, endpoint)

	err = stream.Send(&api.PipeResponse{
		Topic: api.Topic_T_CONN,
	})
	if err != nil {
		err = fmt.Errorf("reply to worker: %v", err)
		log.Error(err)
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	log.Infof("establish a new pipe channel <%s,%s>", req.Id, endpoint)

	cpr := &Peer{
		Server: rs.s.Options().Address,
		Client: endpoint,
	}

	p := NewPipe(ctx, req.Id, cpr, stream)
	defer p.Close()
	go p.Start()

	rs.ps.Add(p)
	defer rs.ps.Del(p)

	worker, _ := rs.scheduler.GetWorker(ctx, req.Id)
	if worker != nil {
		rs.subs.Pub(&api.WorkHookResult{
			Action: api.HookAction_HA_UP,
			Worker: worker,
		})
		defer rs.subs.Pub(&api.WorkHookResult{
			Action: api.HookAction_HA_DOWN,
			Worker: worker,
		})
	}

	select {
	case <-ctx.Done():
		log.Infof("worker pipe <%s,%s> closed", p.Id, endpoint)
	}

	return nil
}

func (rs *RpcServer) ListWorkflowInstance(ctx context.Context, req *api.ListWorkflowInstanceRequest, rsp *api.ListWorkflowInstanceResponse) error {
	snapshots := rs.scheduler.GetWorkflowInstances()
	rsp.Snapshots = snapshots
	return nil
}

func (rs *RpcServer) ExecuteWorkflowInstance(ctx context.Context, req *api.ExecuteWorkflowInstanceRequest, stream api.FlowRpc_ExecuteWorkflowInstanceStream) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	err := rs.scheduler.ExecuteWorkflowInstance(req.Id, req.Name, req.Definitions, req.DataObjects, req.Properties, rs.ps)
	if err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	if !req.Watch {
		return nil
	}

	ch, err := rs.scheduler.WatchWorkflowInstance(ctx, req.Id)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case result, ok := <-ch:
			if !ok {
				return stream.Close()
			}
			rsp := &api.ExecuteWorkflowInstanceResponse{Result: result}
			err = stream.Send(rsp)
			if err != nil {
				return err
			}
		}
	}
}

func (rs *RpcServer) InspectWorkflowInstance(ctx context.Context, req *api.InspectWorkflowInstanceRequest, rsp *api.InspectWorkflowInstanceResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	w, err := rs.scheduler.InspectWorkflowInstance(ctx, req.Wid)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	rsp.Workflow = w
	return nil
}

func (rs *RpcServer) AbortWorkflowInstance(ctx context.Context, req *api.AbortWorkflowInstanceRequest, rsp *api.AbortWorkflowInstanceResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	w, ok := rs.scheduler.GetWorkflowInstance(req.Wid)
	if !ok {
		return verrs.BadRequest(rs.Id(), "workflow not found")
	}

	w.Abort()
	return nil
}

func (rs *RpcServer) PauseWorkflowInstance(ctx context.Context, req *api.PauseWorkflowInstanceRequest, rsp *api.PauseWorkflowInstanceResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	w, ok := rs.scheduler.GetWorkflowInstance(req.Wid)
	if !ok {
		return verrs.BadRequest(rs.Id(), "workflow not found")
	}

	w.Pause()
	return nil
}

func (rs *RpcServer) ResumeWorkflowInstance(ctx context.Context, req *api.ResumeWorkflowInstanceRequest, rsp *api.ResumeWorkflowInstanceResponse) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	w, ok := rs.scheduler.GetWorkflowInstance(req.Wid)
	if !ok {
		return verrs.BadRequest(rs.Id(), "workflow not found")
	}

	w.Resume()
	return nil
}

func (rs *RpcServer) WatchWorkflowInstance(ctx context.Context, req *api.WatchWorkflowInstanceRequest, stream api.FlowRpc_WatchWorkflowInstanceStream) error {
	if err := req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	ch, err := rs.scheduler.WatchWorkflowInstance(ctx, req.Wid)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	for {
		select {
		case <-ctx.Done():
			return stream.Close()
		case result, ok := <-ch:
			if !ok {
				return stream.Close()
			}
			rsp := &api.WatchWorkflowInstanceResponse{Result: result}
			err = stream.Send(rsp)
			if err != nil {
				return err
			}
		}
	}
}

func (rs *RpcServer) HandleServiceErr(ctx context.Context, req *api.HandleServiceErrRequest, rsp *api.HandleServiceErrResponse) (err error) {
	if err = req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	err = rs.scheduler.HandleServiceErr(ctx, *req.Req)
	return
}

func (rs *RpcServer) ListInteractive(ctx context.Context, req *api.ListInteractiveRequest, rsp *api.ListInteractiveResponse) (err error) {
	if err = req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	rsp.Interactive, err = rs.scheduler.ListInteractive(ctx, req.Pid)
	return
}

func (rs *RpcServer) CommitInteractive(ctx context.Context, req *api.CommitInteractiveRequest, rsp *api.CommitInteractiveResponse) (err error) {
	if err = req.Validate(); err != nil {
		return verrs.BadRequest(rs.Id(), err.Error())
	}

	err = rs.scheduler.CommitInteractive(ctx, req.Pid, req.Sid, req.Properties)
	return
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

	err := rs.scheduler.StepTrace(ctx, req.TraceLog)
	if err != nil {
		return verrs.InternalServerError(rs.Id(), err.Error())
	}

	return nil
}

func (rs *RpcServer) Stop() error {
	rs.scheduler.Stop(true)
	return nil
}
