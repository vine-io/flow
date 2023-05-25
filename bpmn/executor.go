package bpmn

import (
	"context"
	"sync"
	"sync/atomic"
)

type ExecuteCtx struct {
	context.Context
	view      *View
	vars      map[string]any
	objects   map[string]byte
	header    map[string]string
	parallels map[string]string
}

type View struct {
	models map[string]Model
	flows  map[string]*SequenceFlow
}

type Executor interface {
	Execute(ctx *ExecuteCtx) ([]string, error)
}

type Pool struct {
	size    *atomic.Int32
	up      chan int
	down    chan int
	exit    chan struct{}
	release chan struct{}
	queue   chan func()
	wg      sync.WaitGroup
}

func NewPool(size int32) *Pool {
	psize := &atomic.Int32{}
	psize.Store(size)
	p := &Pool{
		size:    psize,
		up:      make(chan int, 1),
		down:    make(chan int, 1),
		exit:    make(chan struct{}, 1),
		release: make(chan struct{}, 10),
		queue:   make(chan func(), 20),
	}

	go p.process()
	return p
}

func (p *Pool) process() {
	for i := 0; i < int(p.size.Load()); i++ {
		p.wg.Add(1)
		go p.startThread(&p.wg, p.queue, p.release)
	}

	for {
		select {
		case <-p.exit:
			return
		case n := <-p.up:
			p.size.Add(int32(n))

			for i := 0; i < n; i++ {
				p.wg.Add(1)
				go p.startThread(&p.wg, p.queue, p.release)
			}

		case n := <-p.down:
			if int32(n) > p.size.Load() {
				n = int(p.size.Load())
			}
			p.size.Add(-int32(n))

			for i := 0; i < n; i++ {
				p.release <- struct{}{}
			}

		}
	}
}

func (p *Pool) Submit(fn func()) {
	p.queue <- fn
}

func (p *Pool) Scale(size int) {
	cs := int(p.size.Load())
	if size > cs {
		p.up <- size - cs
	} else {
		p.down <- cs - size
	}
}

func (p *Pool) Wait() {
	close(p.exit)
	p.wg.Wait()

	close(p.up)
	close(p.down)
	close(p.release)
	close(p.queue)
}

func (p *Pool) startThread(wg *sync.WaitGroup, queue <-chan func(), ch <-chan struct{}) {
	defer wg.Done()
	for {
		select {
		case <-ch:
			return
		case fn := <-queue:
			fn()
		default:
		}
	}
}

type Scheduler struct {
	*Definition
	view  *View
	pool  *Pool
	queue chan Model
	done  chan struct{}
}

func NewScheduler(d *Definition) *Scheduler {
	view := &View{
		models: d.models,
		flows:  d.flows,
	}

	pool := NewPool(1)
	queue := make(chan Model, 10)
	queue <- d.start

	s := &Scheduler{
		Definition: d,
		view:       view,
		pool:       pool,
		queue:      queue,
		done:       make(chan struct{}, 1),
	}
	return s
}

func (s *Scheduler) Run(ctx context.Context) error {

	bctx := &ExecuteCtx{
		Context: ctx,
		view:    s.view,
		vars:    nil,
		objects: nil,
	}

	var err error
	for {
		select {
		case m, ok := <-s.queue:
			if !ok {
				continue
			}

			executor, ok := m.(Executor)
			if !ok {
				continue
			}

			outgoings, e1 := executor.Execute(bctx)
			if err != nil {
				err = e1
				// TODO: handle error
			}

			if len(outgoings) > 1 {
				s.pool.Scale(len(outgoings))
			}

			if len(outgoings) == 0 {
				s.done <- struct{}{}
			}

			for _, outgoing := range outgoings {
				next := s.models[outgoing]
				if ok {
					s.queue <- next
				}
			}

		case <-s.done:
			s.pool.Scale(0)
			goto DONE
		}
	}
DONE:

	s.stop()

	s.pool.Wait()
	return err
}

func (s *Scheduler) stop() {
	close(s.queue)
	close(s.done)
}

//func (s *Scheduler) execute(ctx *ExecuteCtx, m Model) error {
//	executor, ok := m.(Executor)
//	if !ok {
//		return nil
//	}
//
//	if err := executor.Execute(ctx); err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func (s *Scheduler) next() Generator {
//	if s.queue.Len() == 0 {
//		return nil
//	}
//
//	elem := s.queue.Remove(s.queue.Front())
//	if elem == nil {
//		return nil
//	}
//
//	point := elem.(Generator)
//	return point
//}
