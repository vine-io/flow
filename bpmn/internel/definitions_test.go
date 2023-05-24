package internel

import (
	"encoding/xml"
	"testing"
)

func TestDefinitions_MarshalXML(t *testing.T) {
	d := &Definitions{Bpmn: "test"}
	d.Name = "d1"

	out, err := xml.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(out))
}
