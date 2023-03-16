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
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/vine-io/flow/api"
	log "github.com/vine-io/vine/lib/logger"
	"go.etcd.io/etcd/client/v3"
)

type Scheduler struct {
	wg   sync.WaitGroup
	pool *ants.Pool

	storage *clientv3.Client

	emu sync.RWMutex
	em  map[string]*Entity

	smu sync.RWMutex
	sm  map[string]*Step

	wmu sync.RWMutex
	wfm map[string]*api.WorkflowSnapshot

	exit chan struct{}
}

func NewScheduler(storage *clientv3.Client) (*Scheduler, error) {
	pool, err := ants.NewPool(100)
	if err != nil {
		return nil, err
	}

	s := &Scheduler{
		pool:    pool,
		storage: storage,
		em:      map[string]*Entity{},
		sm:      map[string]*Step{},
		wfm:     map[string]*api.WorkflowSnapshot{},
		exit:    make(chan struct{}, 1),
	}
	return s, nil
}

func (s *Scheduler) ExecuteWorkflow(w *api.Workflow, ps *PipeSet) error {

	select {
	case <-s.exit:
		return fmt.Errorf("scheduler stopped")
	default:
	}

	s.wg.Add(1)
	err := s.pool.Submit(func() {
		defer s.wg.Done()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var err error

		// prepare
		for i := range w.Steps {
			step := w.Steps[i]

			var out []byte
			out, err = doStep(ctx, ps, step.Client, api.StepAction_SC_PRAPARE)
			if err != nil {
				return
			}

			log.Debugf(string(out))
		}

		committed := make([]*api.WorkflowStep, 0)

		defer func() {
			if err != nil {
				// rollback
				for i := len(committed) - 1; i >= 0; i-- {
					step := committed[i]
					var out []byte
					out, err = doStep(ctx, ps, step.Client, api.StepAction_SC_ROLLBACK)
					if err == nil {
						return
					}
					log.Debugf(string(out))
				}
			}

			// cancel
			for i := len(committed) - 1; i >= 0; i-- {
				step := committed[i]
				var out []byte
				out, err = doStep(ctx, ps, step.Client, api.StepAction_SC_CANCEL)
				if err == nil {
					return
				}
				log.Debugf(string(out))
			}
		}()

		// commit
		for i := range w.Steps {
			step := w.Steps[i]

			var out []byte
			out, err = doStep(ctx, ps, step.Client, api.StepAction_SC_COMMIT)
			if err == nil {
				return
			}

			committed = append(committed, step)
			log.Debugf(string(out))
		}
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) Stop(wait bool) {
	if wait {
		s.wg.Wait()
	}
	close(s.exit)
}

func doStep(ctx context.Context, ps *PipeSet, id string, action api.StepAction) ([]byte, error) {
	pipe, ok := ps.Get(id)
	if !ok {
		return nil, fmt.Errorf("pipe %s not found", id)
	}

	rch, ech := pipe.Step(NewStep(ctx, action))

	select {
	case <-ctx.Done():
		return nil, context.Canceled
	case b := <-rch:
		return b, nil
	case e := <-ech:
		return nil, e
	}
}
