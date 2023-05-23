package schema

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcess_MarshalXML(t *testing.T) {
	p := &Process{
		MetaName: MetaName{
			Id:   "p1",
			Name: "test",
		},
		IsExecutable: false,
		Elements: []*ProcessElement{
			{
				SequenceFlow: &SequenceFlow{
					MetaName: MetaName{
						Id:   "flow1",
						Name: "flow1",
					},
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
					Type: StartEventType,
					StartEvent: &StartEvent{
						MetaName: MetaName{
							Id:   "event1",
							Name: "event11",
						},
						Out: "flow1",
					},
				},
			},
			{
				Task: &Task{
					Type: UserTaskType,
					UserTask: &UserTask{
						MetaName: MetaName{
							Id:   "task1",
							Name: "task11",
						},
						In: "flow1",
					},
				},
			},
		},
	}

	out, err := xml.MarshalIndent(p, " ", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("\n%v", string(out))

	p1 := &Process{}
	if err = xml.Unmarshal(out, p1); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, p, p1)
}

func TestSequenceEvent_MarshalXML(t *testing.T) {
	event := &Event{
		Type: StartEventType,
		StartEvent: &StartEvent{
			MetaName: MetaName{
				Id:   "event1",
				Name: "event11",
			},
			Out: "flow1",
		},
	}

	out, err := xml.MarshalIndent(event, " ", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(out))

	evt2 := &Event{}
	err = xml.Unmarshal(out, evt2)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, event, evt2)
}

func TestSequenceStartEvent_MarshalXML(t *testing.T) {
	event := &StartEvent{
		MetaName: MetaName{
			Id:   "event1",
			Name: "event11",
		},
		Out: "flow1",
	}

	out, err := xml.MarshalIndent(event, " ", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(out))

	evt2 := &StartEvent{}
	err = xml.Unmarshal(out, evt2)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, event, evt2)
}

func TestSequenceFlow_MarshalXML(t *testing.T) {
	f := &SequenceFlow{
		MetaName: MetaName{
			Id:   "flow1",
			Name: "flow1",
		},
		SourceRef: "event1",
		TargetRef: "task1",
		ConditionExpression: &ConditionExpression{
			Type:  "bpmn:tFormalExpression",
			Value: "=grace&gt;0",
		},
	}

	out, err := xml.MarshalIndent(f, " ", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(out))

	f2 := &SequenceFlow{}
	err = xml.Unmarshal(out, f2)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, f, f2)
}
