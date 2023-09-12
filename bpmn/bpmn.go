package bpmn

import (
	"fmt"

	"github.com/beevik/etree"
)

const (
	TargetNamespace = "http://bpmn.io/schema/bpmn"
	BpmnModel       = "http://www.omg.org/spec/BPMN/20100524/MODEL"
	BpmnDI          = "http://www.omg.org/spec/BPMN/20100524/DI"
	BpmnDC          = "http://www.omg.org/spec/DD/20100524/DC"
	BpmnSpecDI      = "http://www.omg.org/spec/DD/20100524/DI"
	XSI             = "http://www.w3.org/2001/XMLSchema-instance"
	Olive           = "http://olive.io/spec/BPMN/MODEL"
)

type Shape int32

const (
	ProcessShape Shape = iota + 1
	CollaborationShape
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

func (s Shape) Size() (int32, int32) {
	switch s {
	case EventShape, StartEventShape, EndEventShape:
		return 36, 36
	case TaskShape, ServiceTaskShape, UserTaskShape:
		return 100, 80
	case GatewayShape, ExclusiveGatewayShape, InclusiveGatewayShape, ParallelGatewayShape:
		return 50, 50
	case CollaborationShape:
		return 850, 260
	case SubProcessShape:
		return 350, 200
	case DataObjectShape, DataStoreShape:
		return 50, 50
	default:
		return 50, 50
	}
}

type ExtensionElement struct {
	Properties     *Properties
	Headers        *TaskHeaders
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

type TaskHeaders struct {
	Items []*HeaderItem
}

type HeaderItem struct {
	Name  string
	Value string
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
	Bpmn            string
	BpmnDI          string
	DC              string
	DI              string
	XSI             string
	Olive           string
	TargetNamespace string
	Id              string

	elements []Element

	Diagram *Diagram
}

func FromXML(text string) (*Definitions, error) {
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

func NewDefinitions() *Definitions {
	def := &Definitions{
		Bpmn:            BpmnModel,
		BpmnDI:          BpmnDI,
		DC:              BpmnDC,
		DI:              BpmnSpecDI,
		XSI:             XSI,
		Olive:           Olive,
		TargetNamespace: TargetNamespace,
		Id:              "Definitions_" + randName(),
		elements:        []Element{},
		Diagram:         &Diagram{Id: "Diagram_" + randName()},
	}

	return def
}

func (d *Definitions) NewProcess(p *Process) *Definitions {
	d.elements = append(d.elements, p)
	return d
}

func (d *Definitions) DefaultProcess() (*Process, error) {
	for _, elem := range d.elements {
		if elem.GetShape() == ProcessShape {
			return elem.(*Process), nil
		}
	}
	return nil, fmt.Errorf("not found Process")
}

func (d *Definitions) AutoLayout() *Definitions {
	planes := make([]*DiagramPlane, 0)

	for _, elem := range d.elements {
		switch elem.GetShape() {
		case ProcessShape:
			plane := elem.(*Process).Draw()
			planes = append(planes, plane)
		}
	}

	d.Diagram.Planes = planes
	return d
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
	Id     string
	Planes []*DiagramPlane
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
	X      int64
	Y      int64
	Width  int64
	Height int64
}

type DiagramWaypoint struct {
	X int64
	Y int64
}
