package bpmn

var _ Element = (*Task)(nil)

type Task struct {
	Id                    string
	Name                  string
	Incoming              string
	Outgoing              string
	Extension             *ExtensionElement
	Properties            []*Property
	DataInputAssociation  []*DataAssociation
	DataOutputAssociation []*DataAssociation
}

type DataAssociation struct {
	Id     string
	Source string
	Target string
}

func (t *Task) GetShape() Shape {
	return TaskShape
}

func (t *Task) GetID() string {
	return t.Id
}

func (t *Task) SetID(id string) {
	t.Id = id
}

func (t *Task) GetName() string {
	return t.Name
}

func (t *Task) SetName(name string) {
	t.Name = name
}

func (t *Task) GetIncoming() []string {
	return []string{t.Incoming}
}

func (t *Task) SetIncoming(incoming []string) {
	t.Incoming = incoming[0]
}

func (t *Task) GetOutgoing() []string {
	return []string{t.Outgoing}
}

func (t *Task) SetOutgoing(outgoing []string) {
	t.Outgoing = outgoing[0]
}

func (t *Task) GetExtension() *ExtensionElement {
	return t.Extension
}

func (t *Task) SetExtension(elem *ExtensionElement) {
	t.Extension = elem
}

type ServiceTask struct {
	Task
}

func (t *ServiceTask) GetShape() Shape {
	return ServiceTaskShape
}

type UserTask struct {
	Task
}

func (t *UserTask) GetShape() Shape {
	return UserTaskShape
}
