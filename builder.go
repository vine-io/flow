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

type IStepBuilder interface {
	Id() string
	Entity() Entity
	Build() (any, error)
}

// WorkflowStepBuilder the builder pattern is used to separate the construction of a complex object of its
// representation so that the same construction process can create different representations.
type WorkflowStepBuilder struct {
	step   *api.WorkflowStep
	worker string
	entity Entity
}

func NewStepBuilder(step Step, worker string, entity Entity) *WorkflowStepBuilder {
	s := StepToWorkStep(step, worker)
	s.EntityId = entity.GetEID()
	if s.Uid != "" {
		s.Uid = "Step_" + HashName(GetTypePkgName(reflect.TypeOf(step))) + "_" + HashName(entity.GetEID())
	}
	return &WorkflowStepBuilder{step: s, worker: worker, entity: entity}
}

func (b *WorkflowStepBuilder) SetId(id string) *WorkflowStepBuilder {
	b.step.Uid = id
	return b
}

func (b *WorkflowStepBuilder) Id() string {
	return b.step.Uid
}

func (b *WorkflowStepBuilder) Entity() Entity {
	return b.entity
}

func (b *WorkflowStepBuilder) Build() (any, error) {
	return b.step.DeepCopy(), nil
}

// SubWorkflowBuilder the builder pattern is used to separate the construction of a complex object of its
// representation so that the same construction sub process can create different representations.
type SubWorkflowBuilder struct {
	options  *api.WorkflowOption
	entities map[string]string
	items    map[string]string
	steps    []IStepBuilder
	entity   Entity
}

// NewSubWorkflowBuilder returns a new instance of the SubWorkflowBuilder struct.
// The builder can be used to construct a Workflow struct with specific options.
func NewSubWorkflowBuilder(entity Entity, opts ...Option) *SubWorkflowBuilder {
	options := NewOptions(opts...)
	wb := &SubWorkflowBuilder{
		options:  options,
		entities: map[string]string{},
		items:    map[string]string{},
		steps:    []IStepBuilder{},
		entity:   entity,
	}

	return wb
}

//func FromSpec(spec *api.Workflow) *WorkflowBuilder {
//	return &WorkflowBuilder{spec: spec}
//}

// Items adds a map of key-value pairs to the Workflow struct.
func (b *SubWorkflowBuilder) Items(items map[string]any) *SubWorkflowBuilder {
	for k, v := range items {
		b.Item(k, v)
	}
	return b
}

// Item adds a key-value pairs to the Workflow struct.
func (b *SubWorkflowBuilder) Item(key string, value any) *SubWorkflowBuilder {
	if b.items == nil {
		b.items = map[string]string{}
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
	b.items[key] = vv
	return b
}

// Step adds Step interface implementations to the Workflow struct.
func (b *SubWorkflowBuilder) Step(sb IStepBuilder) *SubWorkflowBuilder {
	if b.steps == nil {
		b.steps = make([]IStepBuilder, 0)
	}
	if b.entities == nil {
		b.entities = map[string]string{}
	}
	b.steps = append(b.steps, sb)
	vv, _ := json.Marshal(sb.Entity())
	b.entities[sb.Id()] = string(vv)

	return b
}

// Steps adds a slice of Step interface implementations to the Workflow struct.
func (b *SubWorkflowBuilder) Steps(sbs ...IStepBuilder) *SubWorkflowBuilder {
	for i := range sbs {
		sb := sbs[i]
		b.Step(sb)
	}
	return b
}

type subProcessWorkflow struct {
	builder     *builder.SubProcessBuilder
	dataObjects map[string]string
	properties  map[string]string
}

func (b *SubWorkflowBuilder) Id() string {
	return b.options.Wid
}

func (b *SubWorkflowBuilder) Entity() Entity {
	return b.entity
}

func (b *SubWorkflowBuilder) Build() (any, error) {
	pb := builder.NewSubProcessDefinitionsBuilder(b.options.Name)
	pb.Id(b.options.Wid)
	pb.Start()
	for key, item := range b.items {
		keyText := OliveEscape(key)
		pb.SetProperty(keyText, item)
	}
	dataObjects := make(map[string]string, 0)
	for key, value := range b.entities {
		dataObjects[key] = value
	}

	mappingPrefix := "__step_mapping__"
	for idx := range b.steps {
		sb := b.steps[idx]
		out, err := sb.Build()
		if err != nil {
			return nil, err
		}
		if step, ok := out.(*api.WorkflowStep); ok {
			task := builder.NewServiceTaskBuilder(step.Describe, "dr-service")
			task.SetId(step.Uid)

			name := OliveEscape(step.Name)
			task.SetHeader("stepName", name)
			task.SetHeader("worker", step.Worker)
			task.SetHeader("entity", step.Entity)
			task.SetHeader("entityId", step.EntityId)
			task.SetHeader("describe", step.Describe)
			task.SetHeader("injects", strings.Join(step.Injects, ","))

			pb.SetProperty(mappingPrefix+step.Uid, step.Worker)

			task.SetProperty(mappingPrefix+step.Uid, "")
			task.SetProperty("action", "")
			pb.AppendElem(task.Out())
		}
	}
	pb.SetProperty("entityId", b.entity.GetEID())
	pb.SetProperty("entity", GetTypePkgName(reflect.TypeOf(b.entity)))
	pb.SetProperty("action", api.StepAction_SC_PREPARE.Readably())
	pb.End()

	properties := pb.PopProperty()
	process := pb.Out()

	out := &subProcessWorkflow{
		builder:     process,
		dataObjects: dataObjects,
		properties:  properties,
	}

	return out, nil
}

// WorkflowBuilder the builder pattern is used to separate the construction of a complex object of its
// representation so that the same construction process can create different representations.
type WorkflowBuilder struct {
	options  *api.WorkflowOption
	entities map[string]string
	items    map[string]string
	entity   Entity
	steps    []IStepBuilder
}

// NewWorkFlowBuilder returns a new instance of the WorkflowBuilder struct.
// The builder can be used to construct a Workflow struct with specific options.
func NewWorkFlowBuilder(opts ...Option) *WorkflowBuilder {
	options := NewOptions(opts...)
	wb := &WorkflowBuilder{
		options:  options,
		entities: map[string]string{},
		items:    map[string]string{},
		steps:    []IStepBuilder{},
	}

	return wb
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
	if b.items == nil {
		b.items = map[string]string{}
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
	b.items[key] = vv
	return b
}

// Step adds Step interface implementations to the Workflow struct.
func (b *WorkflowBuilder) Step(sb IStepBuilder) *WorkflowBuilder {
	if b.steps == nil {
		b.steps = make([]IStepBuilder, 0)
	}
	if b.entities == nil {
		b.entities = map[string]string{}
	}
	b.steps = append(b.steps, sb)
	vv, _ := json.Marshal(sb.Entity())
	b.entities[sb.Id()] = string(vv)

	return b
}

// Steps adds a slice of Step interface implementations to the Workflow struct.
func (b *WorkflowBuilder) Steps(sbs ...IStepBuilder) *WorkflowBuilder {
	for i := range sbs {
		sb := sbs[i]
		b.Step(sb)
	}
	return b
}

func (b *WorkflowBuilder) ToProcessDefinitions() (*schema.Definitions, map[string]string, map[string]string, error) {

	pb := builder.NewProcessDefinitionsBuilder(b.options.Name)
	pb.Id(b.options.Wid)
	pb.Start()
	for key, item := range b.items {
		keyText := OliveEscape(key)
		pb.SetProperty(keyText, item)
	}
	dataObjects := make(map[string]string)
	for key, value := range b.entities {
		dataObjects[key] = value
	}

	properties := make(map[string]string)
	mappingPrefix := "__step_mapping__"
	for idx := range b.steps {
		sb := b.steps[idx]
		out, err := sb.Build()
		if err != nil {
			return nil, nil, nil, err
		}
		if step, ok := out.(*api.WorkflowStep); ok {
			task := builder.NewServiceTaskBuilder(step.Describe, "dr-service")
			task.SetId(step.Uid)

			name := OliveEscape(step.Name)
			task.SetHeader("stepName", name)
			task.SetHeader("worker", step.Worker)
			task.SetHeader("entity", step.Entity)
			task.SetHeader("entityId", step.EntityId)
			task.SetHeader("describe", step.Describe)
			task.SetHeader("injects", strings.Join(step.Injects, ","))

			pb.SetProperty(mappingPrefix+step.Uid, step.Worker)

			task.SetProperty(mappingPrefix+step.Uid, "")
			task.SetProperty("action", "")
			pb.AppendElem(task.Out())
		}
		if sp, ok := out.(*subProcessWorkflow); ok {
			pb.AppendElem(sp.builder)
			for key, ds := range sp.dataObjects {
				if _, ok1 := dataObjects[key]; !ok1 {
					dataObjects[key] = ds
				}
			}
			for key, property := range sp.properties {
				if _, ok1 := properties[key]; !ok1 {
					properties[key] = property
				}
			}
		}
	}
	pb.SetProperty("action", api.StepAction_SC_PREPARE.Readably())
	pb.End()

	for key, property := range pb.PopProperty() {
		properties[key] = property
	}

	d, err := pb.ToDefinitions()
	if err != nil {
		return nil, nil, nil, err
	}
	return d, dataObjects, properties, nil
}

func (b *WorkflowBuilder) ToSubProcessDefinitions() (*schema.Definitions, map[string]string, map[string]string, error) {
	pb := builder.NewSubProcessDefinitionsBuilder(b.options.Name)
	pb.Id(b.options.Wid)
	pb.Start()
	for key, item := range b.items {
		keyText := OliveEscape(key)
		pb.SetProperty(keyText, item)
	}
	dataObjects := make(map[string]string, 0)
	for key, value := range b.entities {
		dataObjects[key] = value
	}

	properties := make(map[string]string)
	mappingPrefix := "__step_mapping__"
	for idx := range b.steps {
		sb := b.steps[idx]
		out, err := sb.Build()
		if err != nil {
			return nil, nil, nil, err
		}
		if step, ok := out.(*api.WorkflowStep); ok {
			task := builder.NewServiceTaskBuilder(step.Describe, "dr-service")
			task.SetId(step.Uid)

			name := OliveEscape(step.Name)
			task.SetHeader("stepName", name)
			task.SetHeader("worker", step.Worker)
			task.SetHeader("entity", step.Entity)
			task.SetHeader("entityId", step.EntityId)
			task.SetHeader("describe", step.Describe)
			task.SetHeader("injects", strings.Join(step.Injects, ","))

			pb.SetProperty(mappingPrefix+step.Uid, step.Worker)

			task.SetProperty(mappingPrefix+step.Uid, "")
			task.SetProperty("action", "")
			pb.AppendElem(task.Out())
		}
		if sp, ok := out.(*subProcessWorkflow); ok {
			pb.AppendElem(sp.builder)
			for key, ds := range sp.dataObjects {
				if _, ok1 := dataObjects[key]; !ok1 {
					dataObjects[key] = ds
				}
			}
			for key, property := range sp.properties {
				if _, ok1 := properties[key]; !ok1 {
					properties[key] = property
				}
			}
		}
	}
	pb.SetProperty("entityId", b.entity.GetEID())
	pb.SetProperty("entity", GetTypePkgName(reflect.TypeOf(b.entity)))
	pb.SetProperty("action", api.StepAction_SC_PREPARE.Readably())
	pb.End()

	for key, property := range pb.PopProperty() {
		properties[key] = property
	}
	d := pb.ToDefinitions()
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
