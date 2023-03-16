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
	"google.golang.org/grpc/peer"
)

type CallPack struct {
	ctx context.Context
	req []byte
	rsp chan []byte
	ech chan error
}

func NewCall(ctx context.Context, req []byte) *CallPack {
	return &CallPack{ctx: ctx, req: req, rsp: make(chan []byte, 1), ech: make(chan error, 1)}
}

func (p *CallPack) Destroy() {
	close(p.rsp)
	close(p.ech)
}

type StepPack struct {
	ctx context.Context
	sa  api.StepAction
	rsp chan []byte
	ech chan error
}

func NewStep(ctx context.Context, sa api.StepAction) *StepPack {
	return &StepPack{ctx: ctx, sa: sa, rsp: make(chan []byte, 1), ech: make(chan error, 1)}
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
		callCh: make(chan *CallPack, 10),
		stepCh: make(chan *StepPack, 10),
		exit:   make(chan struct{}, 1),
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

func handleCall(stream api.FlowRpc_PipeStream, pack *CallPack) {
	req := &api.PipeResponse{
		Topic: api.Topic_T_CALL,
		Call:  &api.PipeCallRequest{Data: pack.req},
	}

	err := stream.Send(req)
	if err != nil {
		pack.ech <- err
		return
	}

	rch := make(chan *api.PipeRequest)
	defer close(rch)

	go func(out chan *api.PipeRequest, err chan error) {
		rsp, e := stream.Recv()
		if e != nil {
			err <- e
			return
		}

		out <- rsp
	}(rch, pack.ech)

	select {
	case <-pack.ctx.Done():
		pack.ech <- context.Canceled
	case rsp := <-rch:
		if rsp.Call == nil {
			pack.ech <- fmt.Errorf("response is empty")
			return
		}

		if e := rsp.Call.Error; e != "" {
			pack.ech <- errors.New(e)
			return
		}

		pack.rsp <- rsp.Call.Data
	}
}

func (p *ClientPipe) Step(pack *StepPack) (<-chan []byte, <-chan error) {
	select {
	case <-p.exit:
		pack.ech <- errors.New("pipe closed")
	case p.stepCh <- pack:
	}

	return pack.rsp, pack.ech
}

func handleStep(stream api.FlowRpc_PipeStream, pack *StepPack) {
	req := &api.PipeResponse{
		Topic: api.Topic_T_STEP,
		Step: &api.PipeStepRequest{
			Action: pack.sa,
		},
	}

	err := stream.Send(req)
	if err != nil {
		pack.ech <- err
		return
	}

	rch := make(chan *api.PipeRequest)
	defer close(rch)

	go func(out chan *api.PipeRequest, err chan error) {
		rsp, e := stream.Recv()
		if err != nil {
			err <- e
			return
		}

		out <- rsp
	}(rch, pack.ech)

	select {
	case <-pack.ctx.Done():
		pack.ech <- context.Canceled

	case rsp := <-rch:
		if rsp.Step == nil {
			pack.ech <- fmt.Errorf("response is empty")
			return
		}

		if e := rsp.Step.Error; e != "" {
			pack.ech <- errors.New(e)
			return
		}

		pack.rsp <- rsp.Step.Data
	}
}

func (p *ClientPipe) Start() {
	p.process()
}

func (p *ClientPipe) process() {
	for {
		select {
		case <-p.exit:
			return
		case pack, ok := <-p.callCh:
			if !ok {
				break
			}

			go handleCall(p.stream, pack)

		case pack, ok := <-p.stepCh:
			if !ok {
				break
			}

			go handleStep(p.stream, pack)
		}
	}
}

func (p *ClientPipe) Close() {
	p.close()
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

func NewPipeSet() *PipeSet {
	return &PipeSet{sets: map[string]*ClientPipe{}}
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
