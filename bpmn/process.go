package bpmn

type Process struct {
	ModelMeta

	flows  map[string]*SequenceFlow
	models map[string]Model
}

func (p *Process) GetShape() Shape { return ProcessShape }

func (p *Process) ReadExtensionElement() (*ExtensionElement, error) {
	return &ExtensionElement{}, nil
}

func (p *Process) WriteExtensionElement(elem *ExtensionElement) error {
	return nil
}
