package bpmn

type Process struct {
	ModelMeta

	flows  map[string]*SequenceFlow
	models map[string]Model
}

func (p *Process) GetShape() Shape { return ProcessShape }

func (p *Process) ReadExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Process) WriteExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}
