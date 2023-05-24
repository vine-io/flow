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

func (o *DataObjectImpl) ReadExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}

func (o *DataObjectImpl) WriteExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}
