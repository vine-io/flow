package bpmn

var _ Element = (*SequenceFlow)(nil)

type SequenceFlow struct {
	Id               string
	Name             string
	SourceRef        string
	TargetRef        string
	Condition        *ConditionExpression
	ExtensionElement *ExtensionElement
}

type ConditionExpression struct {
	Type  string
	Value string
}

func (f *SequenceFlow) GetShape() Shape {
	return FlowShape
}

func (f *SequenceFlow) GetID() string {
	return f.Id
}

func (f *SequenceFlow) SetID(id string) {
	f.Id = id
}

func (f *SequenceFlow) GetName() string {
	return f.Name
}

func (f *SequenceFlow) SetName(name string) {
	f.Name = name
}

func (f *SequenceFlow) GetIncoming() []string {
	return []string{f.SourceRef}
}

func (f *SequenceFlow) SetIncoming(incoming []string) {
	f.SourceRef = incoming[0]
}

func (f *SequenceFlow) GetOutgoing() []string {
	return []string{f.TargetRef}
}

func (f *SequenceFlow) SetOutgoing(outgoing []string) {
	f.TargetRef = outgoing[0]
}

func (f *SequenceFlow) GetExtension() *ExtensionElement {
	return f.ExtensionElement
}

func (f *SequenceFlow) SetExtension(elem *ExtensionElement) {
	f.ExtensionElement = elem
}
