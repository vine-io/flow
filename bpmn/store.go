package bpmn

type DataStore interface {
	Model
	ModelExtension
}

var _ DataStore = (*DataStoreImpl)(nil)

type DataStoreImpl struct {
	ModelMeta
}

func (s *DataStoreImpl) GetShape() Shape { return DataStoreShape }

func (s *DataStoreImpl) ReadExtensionElement() (*ExtensionElement, error) {
	return &ExtensionElement{}, nil
}

func (s *DataStoreImpl) WriteExtensionElement(elem *ExtensionElement) error {
	return nil
}
