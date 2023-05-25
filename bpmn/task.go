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

func (t *TaskImpl) ReadExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TaskImpl) WriteExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TaskImpl) GetIn() string { return t.Incoming }

func (t *TaskImpl) SetIn(in string) { t.Incoming = in }

func (t *TaskImpl) GetOut() string { return t.Outgoing }

func (t *TaskImpl) SetOut(out string) { t.Outgoing = out }

func (t *TaskImpl) Execute(ctx *ExecuteCtx) error {
	fmt.Printf("task %s done!\n", t.GetID())
	return nil
}

type ServiceTask struct {
	TaskImpl
}
