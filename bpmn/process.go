package bpmn

import (
	"github.com/tidwall/btree"
)

var _ Element = (*Process)(nil)

type Process struct {
	Id               string
	Name             string
	ExtensionElement *ExtensionElement

	Elements         *btree.Map[string, Element]
	ObjectReferences *btree.Map[string, *DataReference]
	StoreReferences  *btree.Map[string, *DataReference]
	start            Element
	end              Element
}

type DataReference struct {
	Id  string
	Ref string
}

func (p *Process) GetShape() Shape {
	return ProcessShape
}

func (p *Process) GetID() string {
	return p.Id
}

func (p *Process) SetID(id string) {
	p.Id = id
}

func (p *Process) GetName() string {
	return p.Name
}

func (p *Process) SetName(name string) {
	p.Name = name
}

func (p *Process) GetIncoming() []string {
	return nil
}

func (p *Process) SetIncoming(incoming []string) {
	return
}

func (p *Process) GetOutgoing() []string {
	return nil
}

func (p *Process) SetOutgoing(outgoing []string) {
	return
}

func (p *Process) GetExtension() *ExtensionElement {
	return p.ExtensionElement
}

func (p *Process) SetExtension(elem *ExtensionElement) {
	p.ExtensionElement = elem
}
