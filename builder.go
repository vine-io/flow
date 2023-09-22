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
	"strings"

	json "github.com/json-iterator/go"
	"github.com/olive-io/bpmn/schema"
	"github.com/vine-io/flow/api"
	"github.com/vine-io/flow/builder"
	"github.com/vine-io/pkg/xname"
)

// WorkflowStepBuilder the builder pattern is used to separate the construction of a complex object of its
// representation so that the same construction process can create different representations.
type WorkflowStepBuilder struct {
	step   *api.WorkflowStep
	worker string
	entity Entity
}

func NewStepBuilder(step Step, worker string, entity Entity) *WorkflowStepBuilder {
	s := StepToWorkStep(step, worker)
	if s.Uid != "" {
		s.Uid = "Step_" + HashName(GetTypePkgName(reflect.TypeOf(step)))
	}
	s.EntityId = entity.GetEID()
	return &WorkflowStepBuilder{step: s, worker: worker, entity: entity}
}

func (b *WorkflowStepBuilder) ID(id string) *WorkflowStepBuilder {
	b.step.Uid = id
	return b
}

func (b *WorkflowStepBuilder) Build() *api.WorkflowStep {
	return b.step.DeepCopy()
}

// WorkflowBuilder the builder pattern is used to separate the construction of a complex object of its
// representation so that the same construction process can create different representations.
type WorkflowBuilder struct {
	spec *api.Workflow
}

// NewWorkFlowBuilder returns a new instance of the WorkflowBuilder struct.
// The builder can be used to construct a Workflow struct with specific options.
func NewWorkFlowBuilder(opts ...Option) *WorkflowBuilder {
	options := NewOptions(opts...)
	spec := &api.Workflow{
		Option:   options,
		Entities: map[string]string{},
		Items:    map[string]string{},
		Steps:    []*api.WorkflowStep{},
	}

	return &WorkflowBuilder{spec: spec}
}

func FromSpec(spec *api.Workflow) *WorkflowBuilder {
	return &WorkflowBuilder{spec: spec}
}

// Items adds a map of key-value pairs to the Workflow struct.
func (b *WorkflowBuilder) Items(items map[string]any) *WorkflowBuilder {
	for k, v := range items {
		b.Item(k, v)
	}
	return b
}

// Item adds a key-value pairs to the Workflow struct.
func (b *WorkflowBuilder) Item(key string, value any) *WorkflowBuilder {
	if b.spec.Items == nil {
		b.spec.Items = map[string]string{}
	}

	var vv string
	switch tv := value.(type) {
	case []byte:
		vv = string(tv)
	case string:
		vv = tv
	default:
		data, _ := json.Marshal(value)
		vv = string(data)
	}
	b.spec.Items[key] = vv
	return b
}

// Step adds Step interface implementations to the Workflow struct.
func (b *WorkflowBuilder) Step(sb *WorkflowStepBuilder) *WorkflowBuilder {
	if b.spec.Steps == nil {
		b.spec.Steps = make([]*api.WorkflowStep, 0)
	}
	if b.spec.Entities == nil {
		b.spec.Entities = map[string]string{}
	}
	step := sb.Build()
	b.spec.Steps = append(b.spec.Steps, step)
	vv, _ := json.Marshal(sb.entity)
	b.spec.Entities[step.Uid] = string(vv)

	return b
}

// Steps adds a slice of Step interface implementations to the Workflow struct.
func (b *WorkflowBuilder) Steps(sbs ...*WorkflowStepBuilder) *WorkflowBuilder {
	for i := range sbs {
		sb := sbs[i]
		b.Step(sb)
	}
	return b
}

// Build returns a new Workflow struct based on the current configuration of the WorkflowBuilder.
func (b *WorkflowBuilder) Build() *api.Workflow {
	return b.spec.DeepCopy()
}

func (b *WorkflowBuilder) ToProcessDefinitions() (*schema.Definitions, map[string]string, map[string]string, error) {
	wf := b.spec.DeepCopy()

	pb := builder.NewProcessDefinitionsBuilder(b.spec.Option.Name)
	pb.Id(wf.Option.Wid)
	pb.Start()
	for key, item := range wf.Items {
		keyText := OliveEscape(key)
		pb.SetProperty(keyText, item)
	}
	dataObjects := make(map[string]string, 0)
	for key, value := range wf.Entities {
		dataObjects[key] = value
	}

	mappingPrefix := "__step_mapping__"
	for idx, step := range wf.Steps {
		task := builder.NewServiceTaskBuilder(step.Describe, "dr-service")
		task.SetId(step.Uid)

		name := OliveEscape(step.Name)
		task.SetHeader("stepName", name)
		task.SetHeader("worker", step.Worker)
		task.SetHeader("entity", step.Entity)
		task.SetHeader("entity_id", step.EntityId)
		task.SetHeader("describe", step.Describe)
		task.SetHeader("injects", strings.Join(step.Injects, ","))

		pb.SetProperty(mappingPrefix+step.Uid, step.Worker)

		// 最后一个步骤设置为结束任务
		if idx == len(wf.Steps)-1 {
			task.SetHeader("completed", "true")
		}
		task.SetProperty(step.Uid+"___result", "true")
		task.SetProperty(mappingPrefix+step.Uid, "")
		task.SetProperty("action", "")
		pb.AppendElem(task.Out())
	}
	pb.SetProperty("action", api.StepAction_SC_PREPARE.Readably())
	pb.End()

	properties := pb.PopProperty()
	d, err := pb.ToDefinitions()
	if err != nil {
		return nil, nil, nil, err
	}
	return d, dataObjects, properties, nil
}

func (b *WorkflowBuilder) ToSubProcessDefinitions() (*schema.Definitions, map[string]string, map[string]string, error) {
	wf := b.spec.DeepCopy()

	pb := builder.NewSubProcessDefinitionsBuilder(b.spec.Option.Name)
	pb.Id(wf.Option.Wid)
	pb.Start()
	for key, item := range wf.Items {
		keyText := OliveEscape(key)
		pb.SetProperty(keyText, item)
	}
	dataObjects := make(map[string]string, 0)
	for key, value := range wf.Entities {
		dataObjects[key] = value
	}

	mappingPrefix := "__step_mapping__"
	for idx, step := range wf.Steps {
		task := builder.NewServiceTaskBuilder(step.Describe, "dr-service")
		task.SetId(step.Uid)

		name := OliveEscape(step.Name)
		task.SetHeader("stepName", name)
		task.SetHeader("worker", step.Worker)
		task.SetHeader("entity", step.Entity)
		task.SetHeader("entity_id", step.EntityId)
		task.SetHeader("describe", step.Describe)
		task.SetHeader("injects", strings.Join(step.Injects, ","))

		pb.SetProperty(mappingPrefix+step.Uid, step.Worker)

		// 最后一个步骤设置为结束任务
		if idx == len(wf.Steps)-1 {
			task.SetHeader("completed", "true")
		}
		task.SetProperty(step.Uid+"___result", "true")
		task.SetProperty(mappingPrefix+step.Uid, "")
		task.SetProperty("action", "")
		pb.AppendElem(task.Out())
	}
	pb.SetProperty("action", api.StepAction_SC_PREPARE.Readably())
	pb.End()

	properties := pb.PopProperty()
	d, err := pb.ToDefinitions()
	if err != nil {
		return nil, nil, nil, err
	}
	return d, dataObjects, properties, nil
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
		options.Wid = "WF_" + xname.Gen6()
	}
	if options.Name == "" {
		options.Name = GetTypePkgName(reflect.TypeOf(&api.Workflow{}))
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
