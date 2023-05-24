package internel

import "encoding/xml"

type MetaName struct {
	Id   string `xml:"id,attr,omitempty"`
	Name string `xml:"name,attr,omitempty"`
}

func UnmarshalXMLMetaName(d *xml.Decoder, start xml.StartElement, m *MetaName) error {
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "id":
			m.Id = attr.Value
		case "name":
			m.Name = attr.Value
		}
	}

	return nil
}

type Definitions struct {
	MetaName
	Bpmn             string `xml:"xmlns:bpmn,attr,omitempty"`
	Bpmndi           string `xml:"xmlns:bpmndi,attr,omitempty"`
	Xsi              string `xml:"xmlns:xsi,attr,omitempty"`
	DC               string `xml:"xmlns:dc,attr,omitempty"`
	DI               string `xml:"xmlns:di,attr,omitempty"`
	Modeler          string `xml:"xmlns:modeler,attr,omitempty"`
	TargetNamespace  string `xml:"targetNamespace,attr,omitempty"`
	Exporter         string `xml:"exporter,attr,omitempty"`
	ExporterVersions string `xml:"exporterVersions,attr,omitempty"`

	// bpmn collaboration section
	Collaborations []*Collaboration `xml:"bpmn2:collaborations,omitempty"`

	Conversations []*Conversation `xml:"bpmn2:conversation,omitempty"`

	// bpmn process section
	Process *Process `xml:"bpmn2:process"`

	// bpmn diagram section
	Diagram *Diagram `xml:"bpmndi:diagram"`
}

type definitions Definitions

func (m *Definitions) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "bpmn:definitions"

	d := definitions(*m)
	return e.EncodeElement(d, start)
}

func (m *Definitions) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return nil
}
