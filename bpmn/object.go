package bpmn

type DataObject interface {
	Model
	ModelExtension
}

var _ DataObject = (*DataObjectImpl)(nil)

type DataObjectImpl struct {
	ModelMeta
}

func (o *DataObjectImpl) GetShape() Shape { return DataObjectShape }

func (o *DataObjectImpl) ReadExtensionElement() (*ExtensionElement, error) {
	return &ExtensionElement{}, nil
}

func (o *DataObjectImpl) WriteExtensionElement(elem *ExtensionElement) error {
	return nil
}
