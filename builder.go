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
	json "github.com/json-iterator/go"
	"github.com/vine-io/flow/api"
)

// WorkflowStepBuilder the builder pattern is used to separate the construction of a complex object of its
// representation so that the same construction process can create different representations.
type WorkflowStepBuilder struct {
	step   *api.WorkflowStep
	worker string
	args   map[string]string
}

func NewStepBuilder(step Step, worker string) *WorkflowStepBuilder {
	s := StepToWorkStep(step, worker)
	s.Uid = uuid.New().String()
	return &WorkflowStepBuilder{step: s, args: map[string]string{}}
}

func (b *WorkflowStepBuilder) Arg(k string, v any) *WorkflowStepBuilder {
	if b.args == nil {
		b.args = map[string]string{}
	}

	var vv string
	switch tv := v.(type) {
	case []byte:
		vv = string(tv)
	case string:
		vv = tv
	default:
		if vvv, ok := v.(interface {
			Marshal() ([]byte, error)
		}); ok {
			data, _ := vvv.Marshal()
			vv = string(data)
		} else {
			data, _ := json.Marshal(v)
			vv = string(data)
		}
	}
	b.args[k] = vv
	return b
}

func (b *WorkflowStepBuilder) Args(args map[string]any) *WorkflowStepBuilder {
	for k, v := range args {
		b.Arg(k, v)
	}
	return b
}

func (b *WorkflowStepBuilder) ID(id string) *WorkflowStepBuilder {
	b.step.Uid = id
	return b
}

func (b *WorkflowStepBuilder) Build() *api.WorkflowStep {
	b.step.Args = &api.WorkflowArgs{Args: b.args}
	return b.step.DeepCopy()
}

// WorkflowBuilder the builder pattern is used to separate the construction of a complex object of its
// representation so that the same construction process can create different representations.
type WorkflowBuilder struct {
	spec *api.Workflow
}

// NewBuilder returns a new instance of the WorkflowBuilder struct.
// The builder can be used to construct a Workflow struct with specific options.
func NewBuilder(opts ...Option) *WorkflowBuilder {
	options := NewOptions(opts...)
	spec := &api.Workflow{
		Option:   options,
		Entities: []*api.Entity{},
		Items:    map[string]string{},
		Steps:    []*api.WorkflowStep{},
		StepArgs: map[string]*api.WorkflowArgs{},
	}

	return &WorkflowBuilder{spec: spec}
}

// Entities adds a slice of Entity interface implementations to Workflow struct.
func (b *WorkflowBuilder) Entities(entities ...Entity) *WorkflowBuilder {
	items := make([]*api.Entity, 0, len(entities))
	for i := range entities {
		e := EntityToAPI(entities[i])
		items = append(items, e)
	}
	b.spec.Entities = items

	return b
}

// Items adds a map of key-value pairs to the Workflow struct.
func (b *WorkflowBuilder) Items(items map[string]string) *WorkflowBuilder {
	if b.spec.Items == nil {
		b.spec.Items = map[string]string{}
	}
	for k, v := range items {
		b.spec.Items[k] = v
	}
	return b
}

// Steps adds a slice of Step interface implementations to the Workflow struct.
func (b *WorkflowBuilder) Steps(steps ...*api.WorkflowStep) *WorkflowBuilder {
	items := make([]*api.WorkflowStep, 0, len(steps))
	for i := range steps {
		step := steps[i]
		items = append(items, step)
		b.spec.StepArgs[step.Uid] = step.Args
	}
	b.spec.Steps = items

	return b
}

// Build returns a new Workflow struct based on the current configuration of the WorkflowBuilder.
func (b *WorkflowBuilder) Build() *api.Workflow {
	return b.spec.DeepCopy()
}

// Option represents a configuration option for Workflow struct.
// It provides a way to modify the fields of the Workflow struct a flexible ways.
type Option func(option *api.WorkflowOption)

func NewOptions(opts ...Option) *api.WorkflowOption {
	var options api.WorkflowOption
	for _, o := range opts {
		o(&options)
	}

	if options.Wid == "" {
		options.Wid = uuid.New().String()
	}
	if options.Name == "" {
		options.Name = GetTypePkgName(reflect.TypeOf(&api.Workflow{}))
	}
	if options.Mode == api.WorkflowMode_WM_UNKNOWN {
		options.Mode = api.WorkflowMode_WM_AUTO
	}

	return &options
}

// WithName sets the Name field of *api.WorkflowOption to the specified value.
func WithName(name string) Option {
	return func(o *api.WorkflowOption) {
		o.Name = name
	}
}

// WithId sets the Wid field of *api.WorkflowOption to the specified value.
func WithId(id string) Option {
	return func(o *api.WorkflowOption) {
		o.Wid = id
	}
}

// WithMaxRetry sets the MaxRetries field of *api.WorkflowOption to the specified value.
func WithMaxRetry(retry int32) Option {
	return func(o *api.WorkflowOption) {
		o.MaxRetries = retry
	}
}

// WithMode sets the Mode field of *api.WorkflowOption to the specified value.
func WithMode(mode api.WorkflowMode) Option {
	return func(o *api.WorkflowOption) {
		o.Mode = mode
	}
}
