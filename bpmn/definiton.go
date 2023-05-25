package bpmn

import (
	"bytes"

	"github.com/google/uuid"
)

type Definition struct {
	ModelMeta

	start  *StartEvent
	cur    Model
	flows  map[string]*SequenceFlow
	models map[string]Model
}

/*
 se ---> x ---> st (n1) ---> x ---> ee
         |                   |
         | ---> ut (s1) ---> |
         |                   |
         | ---> ut (s1) ---> |
*/

func (p *Definition) Beautify() string {
	bs := make([]*bytes.Buffer, 0)
	traverse(&bs, p, p.start, 0, 0)
	b := bytes.NewBufferString("")
	for i, item := range bs {
		b.WriteString(item.String())
		if i < len(bs) {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func fillPrefix(c int) string {
	s := ""
	for i := 0; i < c; i++ {
		s += "       "
	}
	return s
}

func traverse(bs *[]*bytes.Buffer, p *Definition, m Model, x, y int) {
	if m == nil {
		return
	}

	if len(*bs) < x+1 {
		b := bytes.NewBufferString("")
		b.WriteString(fillPrefix(y))
		*bs = append(*bs, b)
	}
	b := (*bs)[x]
	switch m.GetShape() {
	case EventShape:
		b.WriteString("--->")
		b.WriteString(" e ")
		y += 1
	case TaskShape:
		b.WriteString("--->")
		b.WriteString(" t ")
		y += 1
	case GatewayShape:
		b.WriteString("--->")
		b.WriteString(" x ")
		y += 1
	}

	if v, ok := m.(ProcessIO); ok {
		outgoing := v.GetOut()
		if outgoing != "" {
			sf, ok := p.flows[outgoing]
			if !ok {
				return
			}

			next, ok := p.models[sf.Out]
			if ok {
				traverse(bs, p, next, x, y)
			}
		}
	}

	if v, ok := m.(MultipartIO); ok {
		outgoings := v.GetOuts()
		for idx, outgoing := range outgoings {
			sf, ok := p.flows[outgoing]
			if !ok {
				continue
			}

			next, ok := p.models[sf.Out]
			if ok {
				traverse(bs, p, next, x+idx, y)
			}
		}
	}
}

type DefinitionBuilder struct {
	*Definition
	err error
}

func NewDefinitionBuilder(name string) *DefinitionBuilder {
	df := &Definition{
		flows:  map[string]*SequenceFlow{},
		models: map[string]Model{},
	}
	df.Id = uuid.New().String()
	df.Name = name
	return &DefinitionBuilder{Definition: df}
}

func (b *DefinitionBuilder) Start() *DefinitionBuilder {
	evt := &StartEvent{}
	evt.SetID(uuid.New().String())
	b.start = evt
	b.models[evt.Id] = evt
	b.cur = evt
	return b
}

func (b *DefinitionBuilder) AddModel(m Model) *DefinitionBuilder {
	if b.err != nil {
		return b
	}

	if m.GetID() == "" {
		m.SetID(uuid.New().String())
	}

	if v, ok := m.(ProcessIO); ok && v.GetOut() != "" {
		fs := b.flows[v.GetOut()]
		fs.In = m.GetID()
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

func (b *DefinitionBuilder) Seek(id string) *DefinitionBuilder {
	if b.err != nil {
		return b
	}
	if m, ok := b.models[id]; ok {
		b.cur = m
	}
	return b
}

func (b *DefinitionBuilder) Link(from, to, name string, ce *ConditionExpression) *DefinitionBuilder {
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

func (b *DefinitionBuilder) ApplyFlow(flowId, name string, ce *ConditionExpression) *DefinitionBuilder {
	if b.err != nil {
		return b
	}
	if flow, ok := b.flows[flowId]; ok {
		flow.Name = name
		flow.Condition = ce
	}
	return b
}

func (b *DefinitionBuilder) Rename(id, name string) *DefinitionBuilder {
	if b.err != nil {
		return b
	}
	if m, ok := b.models[id]; ok {
		m.SetName(name)
	}
	return b
}

func (b *DefinitionBuilder) End() *DefinitionBuilder {
	if b.err != nil {
		return b
	}
	evt := &EndEvent{}
	evt.SetID(uuid.New().String())
	b.AddModel(evt)
	return b
}

func (b *DefinitionBuilder) Build() (*Definition, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.Definition, nil
}
