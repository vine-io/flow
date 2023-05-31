package bpmn

import (
	"github.com/beevik/etree"
)

type Shape int32

const (
	ProcessShape Shape = iota + 1
	SubProcessShape
	FlowShape
	EventShape
	StartEventShape
	EndEventShape
	TaskShape
	ServiceTaskShape
	UserTaskShape
	GatewayShape
	ExclusiveGatewayShape
	InclusiveGatewayShape
	ParallelGatewayShape
	DataObjectShape
	DataStoreShape
)

type ExtensionElement struct {
	Properties     *Properties
	TaskDefinition *TaskDefinition
	IOMapping      *IOMapping
}

type TaskDefinition struct {
	Type    string
	Retries string
}

type IOMapping struct {
	Input  []*Mapping
	Output []*Mapping
}

type Mapping struct {
	Source string
	Target string
}

type Properties struct {
	Items []*Property
}

type Property struct {
	Name  string
	Value string
}

func (e *ExtensionElement) UnmarshalEXML(start *etree.Element) error {
	return nil
}

type Element interface {
	GetShape() Shape
	GetID() string
	SetID(id string)
	GetName() string
	SetName(name string)
	GetIncoming() []string
	SetIncoming(incoming []string)
	GetOutgoing() []string
	SetOutgoing(outgoing []string)
	GetExtension() *ExtensionElement
	SetExtension(elem *ExtensionElement)
}

type Definitions struct {
	Bpmn   string
	BpmnDI string
	DC     string
	DI     string
	Zeebe  string
	Id     string
	Name   string

	elements []Element

	Diagram *Diagram
}

func (d *Definitions) WriteToBytes() ([]byte, error) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	root := doc.CreateElement("")
	if err := Serialize(d, root); err != nil {
		return nil, err
	}
	doc.AddChild(root)
	settings := etree.NewIndentSettings()
	settings.Spaces = 2
	doc.IndentWithSettings(settings)
	return doc.WriteToBytes()
}

func (d *Definitions) WriteToFile(path string) error {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	root := doc.CreateElement("")
	if err := Serialize(d, root); err != nil {
		return err
	}
	doc.AddChild(root)
	settings := etree.NewIndentSettings()
	settings.Spaces = 2
	doc.IndentWithSettings(settings)
	return doc.WriteToFile(path)
}

type Diagram struct {
	Id    string
	Plane *DiagramPlane
}

type DiagramPlane struct {
	Id      string
	Element string
	Shapes  []*DiagramShape
	Edges   []*DiagramEdge
}

type DiagramShape struct {
	Id      string
	Element string
	Label   *DiagramLabel
	Bounds  *DiagramBounds
}

type DiagramEdge struct {
	Id        string
	Element   string
	Waypoints []*DiagramWaypoint
	Label     *DiagramLabel
}

type DiagramLabel struct {
	Bounds *DiagramBounds
}

type DiagramBounds struct {
	X      string
	Y      string
	Width  string
	Height string
}

type DiagramWaypoint struct {
	X string
	Y string
}

func ParseXML(text string) (*Definitions, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(text); err != nil {
		return nil, err
	}

	out, err := Deserialize(doc.Root())
	if err != nil {
		return nil, err
	}
	definitions := out.(*Definitions)

	return definitions, nil
}
