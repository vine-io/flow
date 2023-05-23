package schema

import (
	"encoding/xml"
	"io"
	"strings"
)

type Process struct {
	MetaName

	IsExecutable     bool              `xml:"isExecutable,attr,omitempty"`
	ExtensionElement *ExtensionElement `xml:"bpmn2:extensionElements,omitempty"`
	Elements         []*ProcessElement `xml:"-"`
}

func (m *Process) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "bpmn2:process"

	if m.Id != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "id"},
			Value: m.Id,
		})
	}
	if m.Name != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "name"},
			Value: m.Name,
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
	_ = UnmarshalXMLMetaName(d, start, &m.MetaName)

	m.Elements = make([]*ProcessElement, 0)
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch typ := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(typ.Name.Local)
			switch {
			case strings.HasSuffix(name, "extensionElement"):
				if err = d.DecodeElement(m.ExtensionElement, &typ); err != nil {
					return err
				}
			default:
				elem := &ProcessElement{}
				if err = d.DecodeElement(elem, &typ); err != nil {
					return err
				}
				m.Elements = append(m.Elements, elem)
			}
		}
	}

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
		return e.EncodeElement(m.Event, start)
	}
	if m.SequenceFlow != nil {
		return e.EncodeElement(m.SequenceFlow, start)
	}
	if m.Task != nil {
		return e.EncodeElement(m.Task, start)
	}
	if m.Gateway != nil {
		return e.EncodeElement(m.Gateway, start)
	}

	return nil
}

func (m *ProcessElement) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	name := strings.ToLower(start.Name.Local)
	if strings.HasSuffix(name, "event") {
		m.Event = &Event{}
		return d.DecodeElement(m.Event, &start)
	} else if strings.HasSuffix(name, "flow") {
		m.SequenceFlow = &SequenceFlow{}
		return d.DecodeElement(m.SequenceFlow, &start)
	} else if strings.HasSuffix(name, "task") {
		m.Task = &Task{}
		return d.DecodeElement(m.Task, &start)
	} else if strings.HasSuffix(name, "gateway") {
		m.Gateway = &Gateway{}
		return d.DecodeElement(m.Gateway, &start)
	}
	return nil
}

type EventType int

const (
	StartEventType EventType = iota + 1
	EndEventType
	IntermediateThrowEventType
)

type Event struct {
	Type                   EventType               `xml:"-"`
	StartEvent             *StartEvent             `xml:"bpmn2:startEvent"`
	EndEvent               *EndEvent               `xml:"-"`
	IntermediateCatchEvent *IntermediateCatchEvent `xml:"-"`
}

func (m *Event) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if m.StartEvent != nil {
		start.Name.Local = "bpmn2:startEvent"
		return e.EncodeElement(m.StartEvent, start)
	}
	if m.EndEvent != nil {
		start.Name.Local = "bpmn2:endEvent"
		return e.EncodeElement(m.EndEvent, start)
	}
	if m.IntermediateCatchEvent != nil {
		start.Name.Local = "bpmn2:intermediateCatchEvent"
		return e.EncodeElement(m.IntermediateCatchEvent, start)
	}

	return nil
}

func (m *Event) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	name := start.Name.Local
	switch name {
	case "startEvent":
		m.Type = StartEventType
		m.StartEvent = &StartEvent{}
		return d.DecodeElement(m.StartEvent, &start)
	case "endEvent":
		m.Type = EndEventType
		m.EndEvent = &EndEvent{}
		return d.DecodeElement(m.EndEvent, &start)
	case "intermediateCatchEventType":
		m.Type = IntermediateThrowEventType
		m.IntermediateCatchEvent = &IntermediateCatchEvent{}
		return d.DecodeElement(m.IntermediateCatchEvent, &start)
	}
	return nil
}

type StartEvent struct {
	MetaName
	ExtensionElement *ExtensionElement `xml:"bpmn2:extensionElements,omitempty"`
	Out              string            `xml:"bpmn2:outgoing,omitempty"`
}

type IntermediateCatchEvent struct {
	MetaName
	ExtensionElement     *ExtensionElement     `xml:"bpmn2:extensionElements,omitempty"`
	TimerEventDefinition *TimerEventDefinition `xml:"bpmn2:timerEventDefinition"`
	Out                  string                `xml:"bpmn2:incoming,omitempty"`
}

type TimerEventDefinition struct {
}

type EndEvent struct {
	MetaName
	ExtensionElement *ExtensionElement `xml:"bpmn2:extensionElements,omitempty"`
	In               string            `xml:"bpmn2:incoming,omitempty"`
}

type TaskType int

const (
	SampleTaskType TaskType = iota + 1
	ServiceTaskType
	UserTaskType
)

type Task struct {
	Type        TaskType     `xml:"-"`
	SampleTask  *SampleTask  `xml:"-"`
	ServiceTask *ServiceTask `xml:"-"`
	UserTask    *UserTask    `xml:"-"`
}

func (m *Task) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if m.SampleTask != nil {
		start.Name.Local = "bpmn2:task"
		return e.EncodeElement(m.SampleTask, start)
	}
	if m.ServiceTask != nil {
		start.Name.Local = "bpmn2:serviceTask"
		return e.EncodeElement(m.ServiceTask, start)
	}
	if m.UserTask != nil {
		start.Name.Local = "bpmn2:userTask"
		return e.EncodeElement(m.UserTask, start)
	}

	return nil
}

func (m *Task) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	name := start.Name.Local
	switch name {
	case "task":
		m.Type = SampleTaskType
		m.SampleTask = &SampleTask{}
		return d.DecodeElement(m.SampleTask, &start)
	case "serviceTask":
		m.Type = ServiceTaskType
		m.ServiceTask = &ServiceTask{}
		return d.DecodeElement(m.ServiceTask, &start)
	case "userTask":
		m.Type = UserTaskType
		m.UserTask = &UserTask{}
		return d.DecodeElement(m.UserTask, &start)
	}
	return nil
}

type SampleTask struct {
	MetaName
	ExtensionElement *ExtensionElement `xml:"bpmn2:extensionElements,omitempty"`
	In               string            `xml:"bpmn2:incoming,omitempty"`
	Out              string            `xml:"bpmn2:outgoing,omitempty"`
}

type ServiceTask struct {
	MetaName
	ExtensionElement *ExtensionElement `xml:"bpmn2:extensionElements,omitempty"`
	In               string            `xml:"bpmn2:incoming,omitempty"`
	Out              string            `xml:"bpmn2:outgoing,omitempty"`
}

type UserTask struct {
	MetaName
	ExtensionElement *ExtensionElement `xml:"bpmn2:extensionElements,omitempty"`
	In               string            `xml:"bpmn2:incoming,omitempty"`
	Out              string            `xml:"bpmn2:outgoing,omitempty"`
}

type SequenceFlow struct {
	MetaName
	SourceRef           string               `xml:"sourceRef,attr,omitempty"`
	TargetRef           string               `xml:"targetRef,attr,omitempty"`
	ConditionExpression *ConditionExpression `xml:"bpmn2:conditionExpression,omitempty"`
}

type sequenceFlow SequenceFlow

func (m *SequenceFlow) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	f := sequenceFlow(*m)
	start.Name.Local = "bpmn2:sequenceFlow"
	return e.EncodeElement(f, start)
}

func (m *SequenceFlow) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var f sequenceFlow
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "name":
			f.Name = attr.Value
		case "id":
			f.Id = attr.Value
		case "sourceRef":
			f.SourceRef = attr.Value
		case "targetRef":
			f.TargetRef = attr.Value
		}
	}
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch typ := tok.(type) {
		case xml.StartElement:
			switch typ.Name.Local {
			case "conditionExpression":
				c := &ConditionExpression{}
				if err = d.DecodeElement(c, &typ); err != nil {
					return err
				}
				f.ConditionExpression = c
			default:
			}
		}
	}

	*m = SequenceFlow(f)
	return nil
}

type ConditionExpression struct {
	Type  string `xml:"xsi:type,attr,omitempty"`
	Value string `xml:",cdata"`
}

type conditionExpression ConditionExpression

func (m *ConditionExpression) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	c := conditionExpression{}
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			c.Type = attr.Value
		}
	}
	if err := d.DecodeElement(&c, &start); err != nil {
		return err
	}
	*m = ConditionExpression(c)
	return nil
}

type GatewayType int

const (
	ExclusiveGatewayType GatewayType = iota + 1
)

type Gateway struct {
	Type             GatewayType       `xml:"-"`
	ExclusiveGateway *ExclusiveGateway `xml:"-"`
}

func (m *Gateway) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if m.ExclusiveGateway != nil {
		start.Name.Local = "bpmn2:exclusiveGateway"
		return e.EncodeElement(m.ExclusiveGateway, start)
	}

	return nil
}

func (m *Gateway) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	name := start.Name.Local
	switch name {
	case "exclusiveGateway":
		m.Type = ExclusiveGatewayType
		m.ExclusiveGateway = &ExclusiveGateway{}
		return d.DecodeElement(m.ExclusiveGateway, &start)
	}
	return nil
}

type ExclusiveGateway struct {
	MetaName
	Ins  []string `xml:"bpmn2:incoming,omitempty"`
	Outs []string `xml:"bpmn2:outgoing,omitempty"`
}
