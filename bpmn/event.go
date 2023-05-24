package bpmn

import "fmt"

type Event interface {
	Model
	ModelExtension
	ProcessIO
}

var _ Event = (*EventImpl)(nil)

type EventImpl struct {
	ModelMeta
	Incoming string
	Outgoing string
}

func (e *EventImpl) GetShape() Shape { return EventShape }

func (e *EventImpl) ReadExtensionElement() (ExtensionElementWriter, error) {
	return nil, fmt.Errorf("not allowed")
}

func (e *EventImpl) WriteExtensionElement() (ExtensionElementWriter, error) {
	return nil, fmt.Errorf("not allowed")
}

func (e *EventImpl) GetIn() string { return e.Incoming }

func (e *EventImpl) SetIn(in string) { e.Incoming = in }

func (e *EventImpl) GetOut() string { return e.Outgoing }

func (e *EventImpl) SetOut(out string) { e.Outgoing = out }

type StartEvent struct {
	EventImpl
}

func (e *StartEvent) GetIn() string { return "" }

func (e *StartEvent) SetIn(string) {}

type EndEvent struct {
	EventImpl
}

func (e *EndEvent) GetOut() string { return "" }

func (e *EndEvent) SetOut(out string) {}
