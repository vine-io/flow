package bpmn

import "fmt"

type Task interface {
	Model
	ModelExtension
	ProcessIO
}

var _ Task = (*TaskImpl)(nil)

type TaskImpl struct {
	ModelMeta
	Incoming string
	Outgoing string
}

func (t *TaskImpl) GetShape() Shape { return TaskShape }

func (t *TaskImpl) ReadExtensionElement() (*ExtensionElement, error) {
	return &ExtensionElement{}, nil
}

func (t *TaskImpl) WriteExtensionElement(elem *ExtensionElement) error {
	return nil
}

func (t *TaskImpl) GetIn() string { return t.Incoming }

func (t *TaskImpl) SetIn(in string) { t.Incoming = in }

func (t *TaskImpl) GetOut() string { return t.Outgoing }

func (t *TaskImpl) SetOut(out string) { t.Outgoing = out }

func (t *TaskImpl) Execute(ctx *ExecuteCtx) ([]string, error) {
	fmt.Printf("task %s done!\n", t.Id)

	view := ctx.view
	if t.Outgoing == "" {
		return nil, nil
	}

	flow, ok := view.flows[t.Outgoing]
	if !ok {
		return nil, nil
	}

	m, ok := view.models[flow.Out]
	if !ok {
		return nil, nil
	}

	return []string{m.GetID()}, nil
}

type UserTask struct {
	TaskImpl
}

type ServiceTask struct {
	TaskImpl
}
