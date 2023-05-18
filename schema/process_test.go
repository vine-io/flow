package schema

import (
	"encoding/xml"
	"testing"
)

func TestProcess_MarshalXML(t *testing.T) {
	p := &Process{
		ID:           "p1",
		BName:        "test",
		IsExecutable: false,
		Elements: []*ProcessElement{
			{
				SequenceFlow: &SequenceFlow{
					ID:        "flow1",
					BName:     "flow1",
					SourceRef: "event1",
					TargetRef: "task1",
					ConditionExpression: &ConditionExpression{
						Type:  "bpmn:tFormalExpression",
						Value: "=grace&gt;0",
					},
				},
			},
			{
				Event: &Event{
					Type:  StartEventType,
					ID:    "event1",
					BName: "event11",
					Ins:   []string{""},
					Outs:  []string{"flow1"},
				},
			},
			{
				Task: &Task{
					Type:  UserTaskType,
					ID:    "task1",
					BName: "task11",
					In:    "flow1",
					Out:   "",
				},
			},
		},
	}

	out, err := xml.MarshalIndent(p, " ", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("\n%v", string(out))
}
