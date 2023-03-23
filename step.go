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
	"reflect"
	"sync"

	"github.com/vine-io/flow/api"
	log "github.com/vine-io/vine/lib/logger"
)

type StepSet struct {
	sync.RWMutex
	sm map[string]*api.Step
}

func NewStepSet() *StepSet {
	return &StepSet{sm: map[string]*api.Step{}}
}

func (s *StepSet) Add(step *api.Step) {
	s.Lock()
	defer s.Unlock()
	s.sm[step.Name] = step
}

func (s *StepSet) Del(step *api.Step) {
	s.Lock()
	defer s.Unlock()
	delete(s.sm, step.Name)
}

func (s *StepSet) Get(name string) (*api.Step, bool) {
	s.RLock()
	step, ok := s.sm[name]
	s.RUnlock()
	return step, ok
}

func (s *StepSet) Contains(name string) bool {
	_, ok := s.Get(name)
	return ok
}

func (s *StepSet) List() []*api.Step {
	s.RLock()
	defer s.RUnlock()
	steps := make([]*api.Step, 0)
	for _, v := range s.sm {
		steps = append(steps, v.DeepCopy())
	}
	return steps
}

// Step 表示具有原子性的复杂操作
type Step interface {
	Metadata() map[string]string

	Prepare(ctx *PipeSessionCtx) error

	Commit(ctx *PipeSessionCtx) error

	Rollback(ctx *PipeSessionCtx) error

	Cancel(ctx *PipeSessionCtx) error
}

var _ Step = (*EmptyStep)(nil)

type EmptyStep struct {
	Client string
	E      *Empty `flow:"entity"`
	A      string `flow:"name:a"`
	B      int32  `flow:"name:b"`
	C      string `flow:"name:c"`
}

func (s *EmptyStep) Metadata() map[string]string {
	id := s.Client
	if id == "" {
		id = "1"
	}
	return map[string]string{
		StepName:   "empty test step",
		StepWorker: id,
		StepOwner:  GetTypePkgName(reflect.TypeOf(&Empty{})),
	}
}

func (s *EmptyStep) Prepare(ctx *PipeSessionCtx) error {
	log.Infof("a = %v, b = %v", s.A, s.B)
	err := ctx.Put(ctx, "c", "ok")
	if err != nil {
		return err
	}
	return nil
}

func (s *EmptyStep) Commit(ctx *PipeSessionCtx) error {
	s.E.Name = "committed"
	log.Infof("commit")
	log.Infof("c = %v", s.C)
	return nil
}

func (s *EmptyStep) Rollback(ctx *PipeSessionCtx) error {
	s.E.Name = "rollback"
	log.Infof("rollback")
	return nil
}

func (s *EmptyStep) Cancel(ctx *PipeSessionCtx) error {
	s.E.Name = "cancel"
	log.Infof("cancel")
	return nil
}

func NewEmptyStep(e *Empty) *EmptyStep {
	return &EmptyStep{E: e}
}
