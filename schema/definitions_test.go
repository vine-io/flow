package schema

import (
	"encoding/xml"
	"testing"
)

func TestDefinitions_MarshalXML(t *testing.T) {
	d := &Definitions{Bpmn: "test"}

	out, err := xml.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(out))
}
