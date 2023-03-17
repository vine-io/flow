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
	"sync"

	"github.com/vine-io/flow/api"
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

// Step 表示具有原子性的复杂操作
type Step interface {
	Metadata() map[string]string

	Prepare(ctx context.Context) error

	Commit(ctx context.Context) error

	Rollback(ctx context.Context) error

	Cancel(ctx context.Context) error
}

var _ Step = (*EmptyStep)(nil)

type EmptyStep struct {
	E *Empty `inject:""`
}

func (s *EmptyStep) Metadata() map[string]string {
	return map[string]string{
		StepName: "empty test step",
	}
}

func (s *EmptyStep) Prepare(ctx context.Context) error {
	return nil
}

func (s *EmptyStep) Commit(ctx context.Context) error {
	return nil
}

func (s *EmptyStep) Rollback(ctx context.Context) error {
	return nil
}

func (s *EmptyStep) Cancel(ctx context.Context) error {
	return nil
}

func NewEmptyStep(e *Empty) *EmptyStep {
	return &EmptyStep{E: e}
}
