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
	"sync"

	"github.com/vine-io/flow/api"
	log "github.com/vine-io/vine/lib/logger"
	"google.golang.org/grpc/peer"
)

type CallPack struct {
	ctx   context.Context
	chunk *api.PipeCallRequest
	rsp   chan []byte
	ech   chan error
}

func NewCall(ctx context.Context, chunk *api.PipeCallRequest) *CallPack {
	p := &CallPack{
		ctx:   ctx,
		chunk: chunk,
		rsp:   make(chan []byte, 1),
		ech:   make(chan error, 1),
	}
	return p
}

func (p *CallPack) Destroy() {
	close(p.rsp)
	close(p.ech)
}

type StepPack struct {
	ctx   context.Context
	chunk *api.PipeStepRequest
	rsp   chan []byte
	ech   chan error
}

func NewStep(ctx context.Context, chunk *api.PipeStepRequest) *StepPack {
	p := &StepPack{
		ctx:   ctx,
		chunk: chunk,
		rsp:   make(chan []byte, 1),
		ech:   make(chan error, 1),
	}
	return p
}

func (p *StepPack) Destroy() {
	close(p.rsp)
	close(p.ech)
}

type ClientPipe struct {
	Id string
	pr *peer.Peer

	// sync.RWMutex for revision
	rmu      sync.Mutex
	revision *api.Revision

	stream api.FlowRpc_PipeStream

	// the queue of the call request
	cqueue chan *CallPack
	// sync.RWMutex for cmc
	cmu sync.RWMutex
	// the map of call chancel
	cmc map[string]*CallPack

	// the queue of the step request
	squeue chan *StepPack
	// sync.RWMutex for smc
	smu sync.RWMutex
	// the map of step channel
	smc map[string]*StepPack

	exit chan struct{}
}

func NewPipe(id string, pr *peer.Peer, stream api.FlowRpc_PipeStream) *ClientPipe {
	return &ClientPipe{
		Id:       id,
		pr:       pr,
		revision: api.NewRevision(),
		stream:   stream,
		cmc:      map[string]*CallPack{},
		cqueue:   make(chan *CallPack, 10),
		smc:      map[string]*StepPack{},
		squeue:   make(chan *StepPack, 10),
		exit:     make(chan struct{}, 1),
	}
}

func (p *ClientPipe) Call(pack *CallPack) (<-chan []byte, <-chan error) {
	select {
	case <-p.exit:
		pack.ech <- errors.New("pipe closed")
	case p.cqueue <- pack:
	}

	return pack.rsp, pack.ech
}

func (p *ClientPipe) Step(pack *StepPack) (<-chan []byte, <-chan error) {
	select {
	case <-p.exit:
		pack.ech <- errors.New("pipe closed")
	case p.squeue <- pack:
	}

	return pack.rsp, pack.ech
}

func (p *ClientPipe) Start() {
	go p.receiving()
	p.process()
}

func (p *ClientPipe) process() {
	for {
		select {
		case <-p.exit:
			return

		case pack, ok := <-p.cqueue:
			if !ok {
				break
			}

			revision := p.forward()

			err := p.stream.Send(&api.PipeResponse{
				Topic:    api.Topic_T_CALL,
				Revision: &revision,
				Call:     pack.chunk,
			})
			if err != nil {
				pack.ech <- err
				break
			}

			p.cmu.Lock()
			p.cmc[string(revision.ToBytes())] = pack
			p.cmu.Unlock()

		case pack, ok := <-p.squeue:
			if !ok {
				break
			}

			revision := p.forward()

			err := p.stream.Send(&api.PipeResponse{
				Topic:    api.Topic_T_STEP,
				Revision: &revision,
				Step:     pack.chunk,
			})
			if err != nil {
				pack.ech <- err
				break
			}

			p.smu.Lock()
			p.smc[string(revision.ToBytes())] = pack
			p.smu.Unlock()
		}
	}
}

func (p *ClientPipe) forward() api.Revision {
	p.rmu.Lock()
	defer p.rmu.Unlock()
	p.revision.Add()
	return *p.revision
}

func (p *ClientPipe) handleCall(rsp *api.PipeRequest) {
	revision := string(rsp.Revision.ToBytes())

	p.cmu.RLock()
	pack, ok := p.cmc[revision]
	p.cmu.RUnlock()
	if !ok {
		log.Errorf("call %s response abort, missing receiver", revision)
		return
	}

	defer func() {
		p.cmu.Lock()
		delete(p.cmc, revision)
		p.cmu.Unlock()
	}()

	select {
	case <-pack.ctx.Done():
		pack.ech <- context.Canceled
		log.Errorf("call %s response abort, receiver cancelled", revision)
		return
	default:

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
	log.Debugf("call %s response finished", revision)
}

func (p *ClientPipe) handleStep(rsp *api.PipeRequest) {
	revision := string(rsp.Revision.ToBytes())

	p.smu.RLock()
	pack, ok := p.smc[revision]
	p.smu.RUnlock()
	if !ok {
		log.Errorf("step %s response abort, missing receiver", revision)
		return
	}

	defer func() {
		p.smu.Lock()
		delete(p.smc, revision)
		p.smu.Unlock()
	}()

	select {
	case <-pack.ctx.Done():
		pack.ech <- context.Canceled
		log.Errorf("step %s response abort, receiver cancelled", revision)
		return
	default:
	}

	log.Debugf("step %s response finished", revision)

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

func (p *ClientPipe) receiving() {
	for {
		rsp, err := p.stream.Recv()
		if err == io.EOF {
			log.Infof("pipe %s stop receiving handler", p.Id)
			return
		}

		if err != nil {
			log.Errorf("pipe %s receive exemption: %v", err)
			return
		}

		switch rsp.Topic {
		case api.Topic_T_CALL:
			p.handleCall(rsp)
		case api.Topic_T_STEP:
			p.handleStep(rsp)
		}
	}
}

func (p *ClientPipe) Close() {
	p.close()
}

func (p *ClientPipe) close() {
	close(p.cqueue)
	close(p.squeue)
	close(p.exit)

	p.smu.Lock()
	items := make([]string, 0)
	for k, _ := range p.smc {
		items = append(items, k)
	}
	for _, name := range items {
		delete(p.smc, name)
	}
	p.smu.Unlock()

	p.cmu.Lock()
	items = make([]string, 0)
	for k, _ := range p.cmc {
		items = append(items, k)
	}
	for _, name := range items {
		delete(p.cmc, name)
	}
	p.cmu.Unlock()
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
