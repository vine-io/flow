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
	"sync"

	"github.com/vine-io/flow/api"
	log "github.com/vine-io/vine/lib/logger"
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

type PipeStream interface {
	Context() context.Context
	Send(*api.PipeResponse) error
	Recv() (*api.PipeRequest, error)
	Close() error
}

type MemberPipeHandler func(*api.PipeResponse) (*api.PipeRequest, error)

type MemberPipeStream struct {
	ctx     context.Context
	cancel  context.CancelFunc
	id      string
	queue   chan *api.PipeResponse
	handler MemberPipeHandler
}

func NewMemberPipeStream(ctx context.Context, id string, handler MemberPipeHandler) *MemberPipeStream {
	ctx, cancel := context.WithCancel(ctx)
	s := &MemberPipeStream{
		ctx:     ctx,
		cancel:  cancel,
		id:      id,
		queue:   make(chan *api.PipeResponse, 3),
		handler: handler,
	}
	return s
}

func (s *MemberPipeStream) Context() context.Context {
	return s.ctx
}

func (s *MemberPipeStream) Send(req *api.PipeResponse) error {
	if s.isClosed() {
		return fmt.Errorf("stream closed")
	}
	s.queue <- req
	return nil
}

func (s *MemberPipeStream) Recv() (*api.PipeRequest, error) {

	select {
	case <-s.ctx.Done():
		return nil, io.EOF
	case data := <-s.queue:
		return s.handler(data)
	}
}

func (s *MemberPipeStream) isClosed() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

func (s *MemberPipeStream) Close() error {
	select {
	case <-s.ctx.Done():
		return fmt.Errorf("already closed")
	default:
	}

	s.cancel()
	return nil
}

type Peer struct {
	Server string
	Client string
}

type ClientPipe struct {
	ctx context.Context

	Id string
	pr *Peer

	// sync.RWMutex for revision
	rmu      sync.Mutex
	revision *api.Revision

	stream PipeStream

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

func NewPipe(ctx context.Context, id string, pr *Peer, stream PipeStream) *ClientPipe {
	return &ClientPipe{
		ctx:      ctx,
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
		pack.ech <- api.ErrCancel("pipe closed")
	case p.cqueue <- pack:
		select {
		case <-p.ctx.Done():
			pack.ech <- api.ErrCancel("pipe closed")
		default:

		}
	}

	return pack.rsp, pack.ech
}

func (p *ClientPipe) Step(pack *StepPack) (<-chan []byte, <-chan error) {
	select {
	case <-p.exit:
		pack.ech <- api.ErrCancel("pipe closed")
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
			goto EXIT

		case pack, ok := <-p.cqueue:
			if !ok {
				goto EXIT
			}

			revision := p.forward()

			log.Debugf("send call request to %s in %s", p.Id, revision.Readably())

			err := p.stream.Send(&api.PipeResponse{
				Topic:    api.Topic_T_CALL,
				Revision: revision,
				Call:     pack.chunk,
			})
			if err != nil {
				pack.ech <- err
				goto EXIT
			}

			p.newCallPack(revision.Readably(), pack)
			go func() {
				select {
				case <-p.ctx.Done():
					pack.ech <- api.ErrCancel("pipe closed")
					p.delCallPack(revision.Readably())
				case <-pack.ctx.Done():
				}
			}()

		case pack, ok := <-p.squeue:
			if !ok {
				goto EXIT
			}

			revision := p.forward()

			log.Debugf("send step request to %s in %s", p.Id, revision.Readably())

			err := p.stream.Send(&api.PipeResponse{
				Topic:    api.Topic_T_STEP,
				Revision: revision,
				Step:     pack.chunk,
			})
			if err != nil {
				pack.ech <- err
				goto EXIT
			}

			p.newStepPack(revision.Readably(), pack)
			go func() {
				select {
				case <-p.ctx.Done():
					pack.ech <- api.ErrCancel("pipe closed")
					p.delStepPack(revision.Readably())
				case <-pack.ctx.Done():
				}
			}()
		}
	}

EXIT:
	log.Infof("stop worker pipe <%s,%s> process", p.Id, p.pr.Client)
}

func (p *ClientPipe) forward() *api.Revision {
	p.rmu.Lock()
	defer p.rmu.Unlock()
	p.revision.Add()
	return p.revision.DeepCopy()
}

func (p *ClientPipe) getCallPack(revision string) (*CallPack, bool) {
	p.cmu.RLock()
	pack, ok := p.cmc[revision]
	p.cmu.RUnlock()
	return pack, ok
}

func (p *ClientPipe) newCallPack(revision string, pack *CallPack) {
	p.cmu.Lock()
	p.cmc[revision] = pack
	p.cmu.Unlock()
}

func (p *ClientPipe) delCallPack(revision string) {
	p.cmu.Lock()
	delete(p.cmc, revision)
	p.cmu.Unlock()
}

func (p *ClientPipe) getStepPack(revision string) (*StepPack, bool) {
	p.smu.RLock()
	pack, ok := p.smc[revision]
	p.smu.RUnlock()
	return pack, ok
}

func (p *ClientPipe) newStepPack(revision string, pack *StepPack) {
	p.smu.Lock()
	p.smc[revision] = pack
	p.smu.Unlock()
}

func (p *ClientPipe) delStepPack(revision string) {
	p.smu.Lock()
	delete(p.smc, revision)
	p.smu.Unlock()
}

func (p *ClientPipe) handlePing(rsp *api.PipeRequest) {
	log.Debugf("reply pong response")
	err := p.stream.Send(&api.PipeResponse{Topic: api.Topic_T_PING})
	if err != nil {
		log.Errorf("reply pong response: %v", err)
	}
}

func (p *ClientPipe) handleCall(rsp *api.PipeRequest) {
	revision := rsp.Revision.Readably()

	pack, ok := p.getCallPack(revision)
	if !ok {
		log.Errorf("call %s response abort, missing receiver", revision)
		return
	}

	defer p.delCallPack(revision)

	select {
	case <-pack.ctx.Done():
		pack.ech <- api.ErrCancel("pipe handle call request")
		log.Errorf("call %s response abort, receiver cancelled", revision)
		return
	default:
	}

	if rsp.Call == nil {
		pack.ech <- fmt.Errorf("response is empty")
		return
	}

	if e := rsp.Call.Error; e != "" {
		pack.ech <- api.Parse(e)
		return
	}

	pack.rsp <- rsp.Call.Data
	log.Debugf("call %s response finished", revision)
}

func (p *ClientPipe) handleStep(rsp *api.PipeRequest) {
	revision := rsp.Revision.Readably()

	pack, ok := p.getStepPack(revision)
	if !ok {
		log.Errorf("step %s response abort, missing receiver", revision)
		return
	}

	defer p.delStepPack(revision)

	select {
	case <-pack.ctx.Done():
		pack.ech <- api.ErrCancel("pipe handle step request")
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
		pack.ech <- api.Parse(e)
		return
	}

	pack.rsp <- rsp.Step.Data
}

func (p *ClientPipe) receiving() {
	for {
		rsp, err := p.stream.Recv()
		if err == io.EOF || (err != nil && IsCancel(err)) {
			log.Infof("pipe %s stop receiving handler", p.Id)
			return
		}

		if err != nil {
			log.Errorf("pipe %s receive exception: %+v", p.Id, err)
			return
		}

		log.Debugf("[%s] receive data from %v", rsp.Topic.Readably(), p.pr.Client)

		switch rsp.Topic {
		case api.Topic_T_PING:
			p.handlePing(rsp)
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
