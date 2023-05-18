package schema

import (
	"encoding/xml"
)

type Process struct {
	xml.Name         `xml:"bpmn:process"`
	ID               string            `xml:"id,attr,omitempty"`
	BName            string            `xml:"name,attr,omitempty"`
	IsExecutable     bool              `xml:"isExecutable,attr,omitempty"`
	ExtensionElement *ExtensionElement `xml:"bpmn:extensionElements,omitempty"`
	Elements         []*ProcessElement `xml:"-"`
}

func (m *Process) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "bpmn:process"

	if m.ID != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "id"},
			Value: m.ID,
		})
	}
	if m.BName != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "name"},
			Value: m.BName,
		})
	}
	if m.IsExecutable {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "isExecutable"},
			Value: "true",
		})
	}

	if err := e.EncodeToken(start); err != nil {
		return err
	}

	for _, elem := range m.Elements {
		if err := e.Encode(elem); err != nil {
			return err
		}
	}

	return e.EncodeToken(start.End())
}

func (m *Process) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return nil
}

type ExtensionElement struct {
}

type ProcessElement struct {
	Event        *Event        `xml:"-"`
	SequenceFlow *SequenceFlow `xml:"-"`
	Task         *Task         `xml:"-"`
	Gateway      *Gateway      `xml:"-"`
}

func (m *ProcessElement) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if m.Event != nil {
		return e.Encode(m.Event)
	}
	if m.SequenceFlow != nil {
		return e.Encode(m.SequenceFlow)
	}
	if m.Task != nil {
		return e.Encode(m.Task)
	}
	if m.Gateway != nil {
		return e.Encode(m.Gateway)
	}

	return nil
}

func (m *ProcessElement) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	//typ, ok := getAttr(start.Attr, "type")
	//if !ok {
	//	typ = "file"
	//}
	//a.Source = &DomainDiskSource{}
	//if typ == "file" {
	//	a.Source.File = &DomainDiskSourceFile{}
	//} else if typ == "block" {
	//	a.Source.Block = &DomainDiskSourceBlock{}
	//} else if typ == "network" {
	//	a.Source.Network = &DomainDiskSourceNetwork{}
	//} else if typ == "dir" {
	//	a.Source.Dir = &DomainDiskSourceDir{}
	//} else if typ == "volume" {
	//	a.Source.Volume = &DomainDiskSourceVolume{}
	//} else if typ == "nvme" {
	//	a.Source.NVME = &DomainDiskSourceNVME{}
	//} else if typ == "vhostuser" {
	//	a.Source.VHostUser = &DomainDiskSourceVHostUser{}
	//}
	//disk := domainDisk(*a)
	//err := d.DecodeElement(&disk, &start)
	//if err != nil {
	//	return err
	//}
	//*a = DomainDisk(disk)
	return nil
}

type EventType int

const (
	StartEventType EventType = iota + 1
	EndEventType
	IntermediateThrowEventType
)

type Event struct {
	Type  EventType `xml:"-"`
	ID    string    `xml:"id,attr,omitempty"`
	BName string    `xml:"name,attr,omitempty"`
	Ins   []string  `xml:"bpmn:incoming,omitempty"`
	Outs  []string  `xml:"bpmn:outgoing,omitempty"`
}

type event Event

func (m *Event) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	evt := event(*m)
	switch m.Type {
	case StartEventType:
		start.Name.Local = "bpmn:startEvent"
	case EndEventType:
		start.Name.Local = "bpmn:endEvent"
	case IntermediateThrowEventType:
		start.Name.Local = "bpmn:intermediateThrowEventType"
	default:
	}

	return e.EncodeElement(evt, start)
}

type TaskType int

const (
	DefaultTaskType TaskType = iota + 1
	ServiceTaskType
	UserTaskType
)

type Task struct {
	Type  TaskType `xml:"-"`
	ID    string   `xml:"id,attr,omitempty"`
	BName string   `xml:"name,attr,omitempty"`
	In    string   `xml:"bpmn:incoming,omitempty"`
	Out   string   `xml:"bpmn:outgoing,omitempty"`
}

type task Task

func (m *Task) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	t := task(*m)
	switch m.Type {
	case DefaultTaskType:
		start.Name.Local = "bpmn:task"
	case ServiceTaskType:
		start.Name.Local = "bpmn:serviceTask"
	case UserTaskType:
		start.Name.Local = "bpmn:userTask"
	default:
	}

	return e.EncodeElement(t, start)
}

type SequenceFlow struct {
	xml.Name            `xml:"bpmn:sequenceFlow"`
	ID                  string               `xml:"id,attr,omitempty"`
	BName               string               `xml:"name,attr,omitempty"`
	SourceRef           string               `xml:"sourceRef,attr,omitempty"`
	TargetRef           string               `xml:"targetRef,attr,omitempty"`
	ConditionExpression *ConditionExpression `xml:"bpmn:conditionExpression,omitempty"`
}

type ConditionExpression struct {
	xml.Name `xml:"bpmn:conditionExpression,omitempty"`
	Type     string `xml:"xsi:type,attr,omitempty"`
	Value    string `xml:",cdata"`
}

type GatewayType int

const (
	ExclusiveGatewayType GatewayType = iota + 1
)

type Gateway struct {
	Type  GatewayType `xml:"-"`
	ID    string      `xml:"id,attr,omitempty"`
	BName string      `xml:"name,attr,omitempty"`
	Ins   []string    `xml:"bpmn:incoming,omitempty"`
	Outs  []string    `xml:"bpmn:outgoing,omitempty"`
}

type gateway Gateway

func (m *Gateway) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	gw := gateway(*m)
	switch m.Type {
	case ExclusiveGatewayType:
		start.Name.Local = "bpmn:exclusiveGateway"
	default:
	}

	return e.EncodeElement(gw, start)
}
