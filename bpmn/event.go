package bpmn

var _ Element = (*baseEvent)(nil)

type baseEvent struct {
	Id               string
	Name             string
	Incoming         []string
	Outgoing         []string
	ExtensionElement *ExtensionElement
}

func (e *baseEvent) GetShape() Shape {
	return EventShape
}

func (e *baseEvent) GetID() string {
	return e.Id
}

func (e *baseEvent) SetID(id string) {
	e.Id = id
}

func (e *baseEvent) GetName() string {
	return e.Name
}

func (e *baseEvent) SetName(name string) {
	e.Name = name
}

func (e *baseEvent) GetIncoming() []string {
	return e.Incoming
}

func (e *baseEvent) SetIncoming(incoming []string) {
	e.Incoming = incoming
}

func (e *baseEvent) GetOutgoing() []string {
	return e.Outgoing
}

func (e *baseEvent) SetOutgoing(outgoing []string) {
	e.Outgoing = outgoing
}

func (e *baseEvent) GetExtension() *ExtensionElement {
	return nil
}

func (e *baseEvent) SetExtension(elem *ExtensionElement) {}

type StartEvent struct {
	baseEvent
}

func (e *StartEvent) GetShape() Shape {
	return StartEventShape
}

func (e *StartEvent) GetIncoming() []string {
	return nil
}

func (e *StartEvent) GetOutgoing() []string {
	return e.Outgoing
}

type EndEvent struct {
	baseEvent
}

func (e *EndEvent) GetShape() Shape {
	return EndEventShape
}

func (e *EndEvent) GetIncoming() []string {
	return e.Incoming
}

func (e *EndEvent) GetOutgoing() []string {
	return nil
}
