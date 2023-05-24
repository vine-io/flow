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

func (s *DataStoreImpl) ReadExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}

func (s *DataStoreImpl) WriteExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}
