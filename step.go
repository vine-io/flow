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
	// Owner Step 所属 Entity
	Owner() reflect.Type

	Prepare(ctx *PipeSessionCtx) error

	Commit(ctx *PipeSessionCtx) (map[string]any, error)

	Rollback(ctx *PipeSessionCtx) error

	Cancel(ctx *PipeSessionCtx) error
	// Desc Step 描述信息
	Desc() string
}

var _ Step = (*TestStep)(nil)

type TestStep struct {
	E *Empty `flow:"ctx:entity"`
	B int32  `flow:"ctx:b"`
	C string `flow:"ctx:c"`
	d int32
}

func (s *TestStep) Owner() reflect.Type {
	return reflect.TypeOf(&Empty{})
}

func (s *TestStep) Prepare(ctx *PipeSessionCtx) error {
	log.Infof("b = %v", s.B)
	err := ctx.Put(ctx, "c", "ok")
	if err != nil {
		return err
	}
	s.d = 3
	return nil
}

func (s *TestStep) Commit(ctx *PipeSessionCtx) (map[string]any, error) {
	log.Infof("commit")
	log.Infof("c = %v, d = %v", s.C, s.d)
	return map[string]any{"b": 12}, nil
}

func (s *TestStep) Rollback(ctx *PipeSessionCtx) error {
	log.Infof("rollback")
	return nil
}

func (s *TestStep) Cancel(ctx *PipeSessionCtx) error {
	log.Infof("cancel")
	return nil
}

func (s *TestStep) Desc() string {
	return ""
}

var _ Step = (*EmptyStep)(nil)

type EmptyStep struct{}

func (s *EmptyStep) Owner() reflect.Type { return reflect.TypeOf(&Empty{}) }

func (s *EmptyStep) Prepare(ctx *PipeSessionCtx) error { return nil }

func (s *EmptyStep) Commit(ctx *PipeSessionCtx) (out map[string]any, err error) { return }

func (s *EmptyStep) Rollback(ctx *PipeSessionCtx) error { return nil }

func (s *EmptyStep) Cancel(ctx *PipeSessionCtx) error { return nil }

func (s *EmptyStep) Desc() string { return "empty step" }

type CellStep struct {
	EmptyStep
}
