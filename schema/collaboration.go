package schema

type Collaboration struct {
	MetaName
	Participant *Participant `xml:"bpmn:participant,omitempty"`
}

type Participant struct {
	MetaName
	ProcessRef string `xml:"processRef,attr,omitempty"`
}
