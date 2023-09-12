package bpmn

import "strings"

// Builder builds Definitions structure
type Builder struct {
	d   *Definitions
	ptr *Process
	cur string
	err error
}

func NewBuilder(name string) *Builder {
	def := NewDefinitions()
	ptr := NewProcess(name)
	def.elements = []Element{ptr}
	return &Builder{d: def, ptr: ptr}
}

func (b *Builder) Id(id string) *Builder {
	b.ptr.Id = id
	return b
}

func (b *Builder) SetProperty(key, value string) *Builder {
	property := &Property{
		Name:  key,
		Value: value,
	}
	if b.ptr.ExtensionElement.Properties == nil {
		b.ptr.ExtensionElement.Properties = &Properties{Items: make([]*Property, 0)}
	}
	if b.ptr.ExtensionElement.Headers == nil {
		b.ptr.ExtensionElement.Headers = &TaskHeaders{Items: make([]*HeaderItem, 0)}
	}
	if key != "action" && !strings.HasPrefix(key, "__step_mapping") {
		b.ptr.ExtensionElement.Headers.Items = append(b.ptr.ExtensionElement.Headers.Items, &HeaderItem{Name: key})
	}
	b.ptr.ExtensionElement.Properties.Items = append(b.ptr.ExtensionElement.Properties.Items, property)
	return b
}

func (b *Builder) AppendDep(dep string) *Builder {
	if b.ptr.ExtensionElement.Headers == nil {
		b.ptr.ExtensionElement.Headers = &TaskHeaders{Items: make([]*HeaderItem, 0)}
	}

	exists := false
	for idx, item := range b.ptr.ExtensionElement.Headers.Items {
		if item.Name == "__entities" {
			exists = true
			value := item.Value + "," + dep
			b.ptr.ExtensionElement.Headers.Items[idx] = &HeaderItem{Name: item.Name, Value: value}
			break
		}
	}

	if !exists {
		b.ptr.ExtensionElement.Headers.Items = append(b.ptr.ExtensionElement.Headers.Items, &HeaderItem{Name: "__entities", Value: dep})
	}

	return b
}

func (b *Builder) PopProperty() map[string]any {
	properties := map[string]any{}
	if b.ptr == nil || b.ptr.ExtensionElement == nil ||
		b.ptr.ExtensionElement.Properties == nil ||
		len(b.ptr.ExtensionElement.Properties.Items) == 0 {
		return properties
	}

	for _, item := range b.ptr.ExtensionElement.Properties.Items {
		properties[item.Name] = item.Value
	}
	b.ptr.ExtensionElement.Properties.Items = []*Property{}
	return properties
}

func (b *Builder) Start() *Builder {
	event := &StartEvent{}
	event.SetID(randShapeName(event))
	b.ptr.AppendElem(event)
	b.cur = event.GetID()

	return b
}

func (b *Builder) AppendElem(elem Element) *Builder {
	if elem.GetID() == "" {
		elem.SetID(randShapeName(elem))
	}
	b.ptr.AfterInsertElem(b.cur, elem, nil)
	b.cur = elem.GetID()
	return b
}

func (b *Builder) End() *Builder {
	event := &EndEvent{}
	event.SetID(randShapeName(event))
	b.AppendElem(event)

	return b
}

func (b *Builder) Out() (*Definitions, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.d, nil
}
