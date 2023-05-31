package bpmn

var _ Element = (*DataObject)(nil)

type DataObject struct {
	Id               string
	Name             string
	ExtensiveElement *ExtensionElement
}

func (o *DataObject) GetShape() Shape {
	return DataObjectShape
}

func (o *DataObject) GetID() string {
	return o.Id
}

func (o *DataObject) SetID(id string) {
	o.Id = id
}

func (o *DataObject) GetName() string {
	return o.Name
}

func (o *DataObject) SetName(name string) {
	o.Name = name
}

func (o *DataObject) GetIncoming() []string {
	return nil
}

func (o *DataObject) SetIncoming(incoming []string) {
	return
}

func (o *DataObject) GetOutgoing() []string {
	return nil
}

func (o *DataObject) SetOutgoing(outgoing []string) {
	return
}

func (o *DataObject) GetExtension() *ExtensionElement {
	return o.ExtensiveElement
}

func (o *DataObject) SetExtension(elem *ExtensionElement) {
	o.ExtensiveElement = elem
}
