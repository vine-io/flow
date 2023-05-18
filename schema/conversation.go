package schema

import "encoding/xml"

type Conversation struct {
	xml.Name `xml:"bpmn:conversation"`
}
