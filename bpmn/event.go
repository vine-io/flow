package bpmn

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

func (e *EventImpl) ReadExtensionElement() (*ExtensionElement, error) {
	return &ExtensionElement{}, nil
}

func (e *EventImpl) WriteExtensionElement(elem *ExtensionElement) error {
	return nil
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

func (e *StartEvent) Yield(ctx *ExecuteCtx, view *View) ([]string, bool) {
	if e.Outgoing == "" {
		return nil, false
	}

	flow, ok := view.flows[e.Outgoing]
	if !ok {
		return nil, false
	}

	m, ok := view.models[flow.Out]
	if !ok {
		return nil, false
	}

	return []string{m.GetID()}, true
}

func (e *StartEvent) Execute(ctx *ExecuteCtx) ([]string, error) {
	view := ctx.view
	if e.Outgoing == "" {
		return nil, nil
	}

	flow, ok := view.flows[e.Outgoing]
	if !ok {
		return nil, nil
	}

	m, ok := view.models[flow.Out]
	if !ok {
		return nil, nil
	}

	return []string{m.GetID()}, nil
}

type EndEvent struct {
	EventImpl
}

func (e *EndEvent) GetOut() string { return "" }

func (e *EndEvent) SetOut(out string) {}

func (e *EndEvent) Execute(ctx *ExecuteCtx) ([]string, error) {
	return []string{}, nil
}
