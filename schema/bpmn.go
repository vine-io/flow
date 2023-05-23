package schema

import (
	"fmt"

	"github.com/google/uuid"
)

type Shape int32

const (
	EventShape Shape = iota + 1
	TaskShape
	GatewayShape
	DataObjectShape
	DataStoreShape
	ProcessShape
)

type Model interface {
	GetShape() Shape
	GetID() string
	SetID(string)
	GetName() string
	SetName(string)
}

type ExtensionElementReader interface {
	Read() ([]byte, error)
	Close() error
}

type ExtensionElementWriter interface {
	Write([]byte) error
}

type ModelExtension interface {
	ReadExtensionElement() (ExtensionElementWriter, error)
	WriteExtensionElement() (ExtensionElementWriter, error)
}

var _ Model = (*ModelMeta)(nil)

type ModelMeta struct {
	Id   string
	Name string
}

func (m *ModelMeta) GetShape() Shape {
	return 0
}

func (m *ModelMeta) GetID() string {
	return m.Id
}

func (m *ModelMeta) SetID(id string) {
	m.Id = id
}

func (m *ModelMeta) GetName() string {
	return m.Name
}

func (m *ModelMeta) SetName(name string) {
	m.Name = name
}

type DataObject interface {
	Model
	ModelExtension
}

type DataStore interface {
	Model
	ModelExtension
}

type ProcessIO interface {
	GetIn() string
	SetIn(string)
	GetOut() string
	SetOut(string)
}

type MultipartIO interface {
	GetIns() []string
	SetIns([]string)
	GetOuts() []string
	SetOuts([]string)
}

type Event interface {
	Model
	ModelExtension
	ProcessIO
}

func SetModelIn(m Model, in string) {
	if v1, match := m.(ProcessIO); match {
		v1.SetIn(in)
		return
	}

	if v2, match := m.(MultipartIO); match {
		ints := v2.GetIns()
		if ints == nil {
			ints = []string{}
		}
		ints = append(ints, in)
		v2.SetIns(ints)
		return
	}
}

func SetModelOut(m Model, out string) {
	if v1, match := m.(ProcessIO); match {
		v1.SetOut(out)
		return
	}

	if v2, match := m.(MultipartIO); match {
		outs := v2.GetOuts()
		if outs == nil {
			outs = []string{}
		}
		outs = append(outs, out)
		v2.SetOuts(outs)
		return
	}
}

var _ Event = (*StartEvent)(nil)

type StartEvent struct {
	ModelMeta
	Output string
}

func (e *StartEvent) GetShape() Shape {
	return EventShape
}

func (e *StartEvent) ReadExtensionElement() (ExtensionElementWriter, error) {
	return nil, fmt.Errorf("not allowed")
}

func (e *StartEvent) WriteExtensionElement() (ExtensionElementWriter, error) {
	return nil, fmt.Errorf("not allowed")
}

func (e *StartEvent) GetIn() string { return "" }

func (e *StartEvent) SetIn(string) {}

func (e *StartEvent) GetOut() string {
	return e.Output
}

func (e *StartEvent) SetOut(out string) {
	e.Output = out
}

var _ Event = (*EndEvent)(nil)

type EndEvent struct {
	ModelMeta
	Incoming string
}

func (e *EndEvent) GetShape() Shape {
	return EventShape
}

func (e *EndEvent) ReadExtensionElement() (ExtensionElementWriter, error) {
	return nil, fmt.Errorf("not allowed")
}

func (e *EndEvent) WriteExtensionElement() (ExtensionElementWriter, error) {
	return nil, fmt.Errorf("not allowed")
}

func (e *EndEvent) GetIn() string { return e.Incoming }

func (e *EndEvent) SetIn(in string) { e.Incoming = in }

func (e *EndEvent) GetOut() string { return "" }

func (e *EndEvent) SetOut(out string) {}

type Task interface {
	Model
	ModelExtension
	ProcessIO
}

type Gateway interface {
	Model
	ModelExtension
	MultipartIO
}

type SequenceFlow struct {
	ModelMeta
	In        string
	Out       string
	Condition *ConditionExpression
}

type ConditionExpression struct {
	Type       string
	Expression string
}

type Process struct {
	ModelMeta

	start  *StartEvent
	cur    Model
	flows  map[string]*SequenceFlow
	models map[string]Model
}

type ProcessBuilder struct {
	*Process
	err error
}

func NewProcessBuilder(name string) *ProcessBuilder {
	p := &Process{
		flows:  map[string]*SequenceFlow{},
		models: map[string]Model{},
	}
	p.Id = uuid.New().String()
	p.Name = name
	return &ProcessBuilder{Process: p}
}

func (b *ProcessBuilder) Start() *ProcessBuilder {
	evt := &StartEvent{}
	evt.SetID(uuid.New().String())
	b.start = evt
	b.models[evt.Id] = evt
	b.cur = evt
	return b
}

func (b *ProcessBuilder) AddModel(m Model) *ProcessBuilder {
	if b.err != nil {
		return b
	}

	fid := uuid.New().String()
	flow := &SequenceFlow{
		In:  b.cur.GetID(),
		Out: m.GetID(),
	}
	flow.Id = fid
	b.flows[flow.Id] = flow

	SetModelOut(b.cur, fid)
	SetModelIn(m, fid)

	b.models[m.GetID()] = m
	b.cur = m
	return b
}

func (b *ProcessBuilder) Seek(id string) *ProcessBuilder {
	if b.err != nil {
		return b
	}
	if m, ok := b.models[id]; ok {
		b.cur = m
	}
	return b
}

func (b *ProcessBuilder) Link(from, to, name string, ce *ConditionExpression) *ProcessBuilder {
	if b.err != nil {
		return b
	}

	fid := uuid.New().String()
	fs, ok := b.models[from]
	if !ok {
		return b
	}
	SetModelOut(fs, fid)

	ts, ok := b.models[to]
	if !ok {
		return b
	}
	SetModelIn(ts, fid)

	flow := &SequenceFlow{
		In:        fs.GetID(),
		Out:       ts.GetID(),
		Condition: ce,
	}
	flow.Id = fid
	flow.Name = name
	b.flows[flow.Id] = flow

	return b
}

func (b *ProcessBuilder) ApplyFlow(flowId, name string, ce *ConditionExpression) *ProcessBuilder {
	if b.err != nil {
		return b
	}
	if flow, ok := b.flows[flowId]; ok {
		flow.Name = name
		flow.Condition = ce
	}
	return b
}

func (b *ProcessBuilder) Rename(id, name string) *ProcessBuilder {
	if b.err != nil {
		return b
	}
	if m, ok := b.models[id]; ok {
		m.SetName(name)
	}
	return b
}

func (b *ProcessBuilder) End() *ProcessBuilder {
	if b.err != nil {
		return b
	}
	evt := &EndEvent{}
	evt.SetID(uuid.New().String())
	b.AddModel(evt)
	return b
}

func (b *ProcessBuilder) Build() (*Process, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.Process, nil
}

type Diagram struct {
	Root *Graph
}

type Graph struct {
	Ref  string
	Outs []*Edge
}

type Edge struct {
	From *Graph
	To   *Graph

	ConditionExpression string
}
