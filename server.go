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
	"sync"

	"github.com/vine-io/flow/api"
	vserver "github.com/vine-io/vine/core/server"
	verrs "github.com/vine-io/vine/lib/errors"
	log "github.com/vine-io/vine/lib/logger"
	"google.golang.org/grpc/peer"
)

var _ api.FlowRpcHandler = (*RpcServer)(nil)

type CallPack struct {
	req []byte
	rsp chan []byte
	ech chan error
}

func NewCall(req []byte) *CallPack {
	return &CallPack{req: req, rsp: make(chan []byte, 1), ech: make(chan error, 1)}
}

func (p *CallPack) Destroy() {
	close(p.rsp)
	close(p.ech)
}

type StepPack struct {
	sa  api.StepAction
	rsp chan []byte
	ech chan error
}

func NewStep(sa api.StepAction) *StepPack {
	return &StepPack{sa: sa, rsp: make(chan []byte, 1), ech: make(chan error, 1)}
}

func (p *StepPack) Destroy() {
	close(p.rsp)
	close(p.ech)
}

type ClientPipe struct {
	Id     string
	pr     *peer.Peer
	stream api.FlowRpc_PipeStream
	callCh chan *CallPack
	stepCh chan *StepPack
	exit   chan struct{}
}

func NewPipe(id string, pr *peer.Peer, stream api.FlowRpc_PipeStream) *ClientPipe {
	return &ClientPipe{
		Id:     id,
		pr:     pr,
		stream: stream,
	}
}

func (p *ClientPipe) Call(pack *CallPack) (<-chan []byte, <-chan error) {
	select {
	case <-p.exit:
		pack.ech <- errors.New("pipe closed")
	case p.callCh <- pack:
	}

	return pack.rsp, pack.ech
}

func (p *ClientPipe) Step(pack *StepPack) (<-chan []byte, <-chan error) {
	select {
	case <-p.exit:
		pack.ech <- errors.New("pipe closed")
	case p.stepCh <- pack:
	}

	return pack.rsp, pack.ech
}

func (p *ClientPipe) process(ctx context.Context) {
	defer p.close()
	for {
		select {
		case <-ctx.Done():
			return
		case value, ok := <-p.callCh:
			if !ok {
				break
			}
			go func(stream api.FlowRpc_PipeStream, pack *CallPack) {
				req := &api.PipeResponse{
					Topic: api.Topic_TopicCall,
					Call:  &api.PipeCallRequest{Data: pack.req},
				}
				err := stream.Send(req)
				if err != nil {
					pack.ech <- err
					return
				}

				rsp, err := stream.Recv()
				if err != nil {
					pack.ech <- err
					return
				}

				if rsp.Call == nil {
					pack.ech <- fmt.Errorf("response is empty")
					return
				}

				if e := rsp.Call.Error; e != "" {
					pack.ech <- errors.New(e)
					return
				}

				pack.rsp <- rsp.Call.Data

			}(p.stream, value)
		case value, ok := <-p.stepCh:
			if !ok {
				break
			}

			go func(stream api.FlowRpc_PipeStream, pack *StepPack) {

				req := &api.PipeResponse{
					Topic: api.Topic_TopicStep,
					Step: &api.PipeStepRequest{
						Action: pack.sa,
					},
				}
				err := stream.Send(req)
				if err != nil {
					pack.ech <- err
					return
				}

				rsp, err := stream.Recv()
				if err != nil {
					pack.ech <- err
					return
				}

				if rsp.Step == nil {
					pack.ech <- fmt.Errorf("response is empty")
					return
				}

				if e := rsp.Step.Error; e != "" {
					pack.ech <- errors.New(e)
					return
				}

				pack.rsp <- rsp.Step.Data

			}(p.stream, value)
		}
	}
}

func (p *ClientPipe) close() {
	close(p.callCh)
	close(p.stepCh)
	close(p.exit)
}

type PipeSet struct {
	sync.RWMutex
	sets map[string]*ClientPipe
}

func (ps *PipeSet) Add(p *ClientPipe) {
	ps.Lock()
	defer ps.Unlock()
	ps.sets[p.Id] = p
}

func (ps *PipeSet) Del(p *ClientPipe) {
	ps.Lock()
	defer ps.Unlock()
	delete(ps.sets, p.Id)
}

func (ps *PipeSet) Get(id string) (*ClientPipe, bool) {
	ps.RLock()
	defer ps.RUnlock()
	p, ok := ps.sets[id]
	return p, ok
}

type RpcServer struct {
	s  vserver.Server
	ps *PipeSet
}

func NewRPCServer(s vserver.Server) (*RpcServer, error) {
	rpc := &RpcServer{s: s}
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

	pack := NewCall(req.Request)
	defer pack.Destroy()
	result, ech := pipe.Call(pack)

	select {
	case <-ctx.Done():
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

	pack := NewStep(req.Action)
	defer pack.Destroy()
	result, ech := pipe.Step(pack)

	select {
	case <-ctx.Done():
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
	pCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	p := NewPipe(req.Id, pr, stream)
	go p.process(pCtx)

	r.ps.Add(p)
	defer r.ps.Del(p)

	select {
	case <-ctx.Done():
		log.Info("client pipe <%s,%s> closed", p.Id, p.pr.Addr.String())
	}

	return nil
}
