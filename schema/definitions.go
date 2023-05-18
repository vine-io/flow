package schema

import "encoding/xml"

type Definitions struct {
	xml.Name         `xml:"bpmn:definitions,omitempty"`
	Bpmn             string `xml:"xmlns:bpmn,attr,omitempty"`
	Bpmndi           string `xml:"xmlns:bpmndi,attr,omitempty"`
	Xsi              string `xml:"xmlns:xsi,attr,omitempty"`
	DC               string `xml:"xmlns:dc,attr,omitempty"`
	DI               string `xml:"xmlns:di,attr,omitempty"`
	Modeler          string `xml:"xmlns:modeler,attr,omitempty"`
	ID               string `xml:"id,attr,omitempty"`
	DName            string `xml:"name,attr,omitempty"`
	TargetNamespace  string `xml:"targetNamespace,attr,omitempty"`
	Exporter         string `xml:"exporter,attr,omitempty"`
	ExporterVersions string `xml:"exporterVersions,attr,omitempty"`

	// bpmn collaboration section
	Collaborations []*Collaboration `xml:"bpmn:collaborations,omitempty"`

	Conversations []*Conversation `xml:"bpmn:conversation,omitempty"`

	// bpmn process section
	Process *Process `xml:"bpmn:process"`

	// bpmn diagram section
	Diagram *Diagram `xml:"bpmndi:diagram"`
}

//func (m *Definitions) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
//	return nil
//}
//
//func (m *Definitions) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
//	return nil
//}
