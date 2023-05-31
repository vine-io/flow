package bpmn

var _ Element = (*DataStore)(nil)

type DataStore struct {
	Id               string
	Name             string
	ExtensionElement *ExtensionElement
}

func (s *DataStore) GetShape() Shape {
	return DataStoreShape
}

func (s *DataStore) GetID() string {
	return s.Id
}

func (s *DataStore) SetID(id string) {
	s.Id = id
}

func (s *DataStore) GetName() string {
	return s.Name
}

func (s *DataStore) SetName(name string) {
	s.Name = name
}

func (s *DataStore) GetIncoming() []string {
	return nil
}

func (s *DataStore) SetIncoming(incoming []string) {
	return
}

func (s *DataStore) GetOutgoing() []string {
	return nil
}

func (s *DataStore) GetExtension() *ExtensionElement {
	return s.ExtensionElement
}

func (s *DataStore) SetOutgoing(outgoing []string) {
	return
}

func (s *DataStore) SetExtension(elem *ExtensionElement) {
	s.ExtensionElement = elem
}
