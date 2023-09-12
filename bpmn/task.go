package bpmn

var _ Element = (*Task)(nil)

type Task struct {
	Id                    string
	Name                  string
	Incoming              string
	Outgoing              string
	Extension             *ExtensionElement
	Properties            []*TaskProperty
	DataInputAssociation  []*DataAssociation
	DataOutputAssociation []*DataAssociation
}

type TaskProperty struct {
	Id   string
	Name string
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
	if t.Incoming == "" {
		return nil
	}
	return []string{t.Incoming}
}

func (t *Task) SetIncoming(incoming []string) {
	t.Incoming = incoming[0]
}

func (t *Task) GetOutgoing() []string {
	if t.Outgoing == "" {
		return nil
	}
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

func NewServiceTask(name, typ string) *ServiceTask {
	st := &ServiceTask{}
	st.Name = name
	st.Id = randShapeName(st)
	st.Extension = &ExtensionElement{
		TaskDefinition: &TaskDefinition{
			Type: typ,
		},
		Properties: &Properties{Items: []*Property{}},
		Headers:    &TaskHeaders{Items: []*HeaderItem{}},
	}

	return st
}

func (t *ServiceTask) GetShape() Shape {
	return ServiceTaskShape
}

func (t *ServiceTask) SetHeader(key, value string) {
	t.Extension.Headers.Items = append(t.Extension.Headers.Items, &HeaderItem{
		Name:  key,
		Value: value,
	})
}

func (t *ServiceTask) SetProperty(name, value string) {
	t.Extension.Properties.Items = append(t.Extension.Properties.Items, &Property{
		Name:  name,
		Value: value,
	})
}

type UserTask struct {
	Task
}

func (t *UserTask) GetShape() Shape {
	return UserTaskShape
}
