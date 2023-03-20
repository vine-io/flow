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

	"github.com/google/uuid"
	"github.com/vine-io/flow/api"
)

type WorkflowBuilder struct {
	spec *api.Workflow
}

func NewBuilder(opts ...Option) *WorkflowBuilder {
	options := NewOptions(opts...)

	if options.Wid == "" {
		options.Wid = uuid.New().String()
	}
	if options.Name == "" {
		options.Name = GetTypePkgName(reflect.TypeOf(&api.Workflow{}))
	}
	if options.Mode == api.WorkflowMode_WM_UNKNOWN {
		options.Mode = api.WorkflowMode_WM_AUTO
	}

	spec := &api.Workflow{
		Option: options,
	}

	return &WorkflowBuilder{spec: spec}
}

func (b *WorkflowBuilder) Entities(entities []Entity) *WorkflowBuilder {
	items := make([]*api.Entity, 0, len(entities))
	for i := range entities {
		e := EntityToAPI(entities[i])
		items = append(items, e)
	}

	return b
}

func (b *WorkflowBuilder) Items(items map[string][]byte) *WorkflowBuilder {
	b.spec.Items = items
	return b
}

func (b *WorkflowBuilder) Steps(steps []Step) *WorkflowBuilder {
	items := make([]*api.WorkflowStep, 0, len(steps))
	for i := range steps {
		step := steps[i]
		metadata := step.Metadata()
		s := &api.WorkflowStep{
			Name:    GetTypePkgName(reflect.TypeOf(step)),
			Uid:     metadata[StepId],
			Entity:  metadata[StepOwner],
			Injects: ExtractFields(step),
		}
		items = append(items, s)
	}
	b.spec.Steps = items

	return b
}

func (b *WorkflowBuilder) Build() *api.Workflow {
	return b.spec.DeepCopy()
}

type Option func(option *api.WorkflowOption)

func NewOptions(opts ...Option) *api.WorkflowOption {
	var options api.WorkflowOption
	for _, o := range opts {
		o(&options)
	}

	return &options
}

func WithName(name string) Option {
	return func(o *api.WorkflowOption) {
		o.Name = name
	}
}

func WithId(id string) Option {
	return func(o *api.WorkflowOption) {
		o.Wid = id
	}
}

func WithMaxRetry(retry int32) Option {
	return func(o *api.WorkflowOption) {
		o.MaxRetries = retry
	}
}

func WithMode(mode api.WorkflowMode) Option {
	return func(o *api.WorkflowOption) {
		o.Mode = mode
	}
}
