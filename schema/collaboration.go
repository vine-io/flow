package schema

import "encoding/xml"

type Collaboration struct {
	xml.Name    `xml:"bpmn:collaboration"`
	ID          string       `xml:"id,attr,omitempty"`
	BName       string       `xml:"name,attr,omitempty"`
	Participant *Participant `xml:"bpmn:participant,omitempty"`
}

type Participant struct {
	xml.Name   `xml:"bpmn:participant"`
	ID         string `xml:"id,attr,omitempty"`
	ProcessRef string `xml:"processRef,attr,omitempty"`
}
