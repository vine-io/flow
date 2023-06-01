package bpmn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcess(t *testing.T) {
	p := NewProcess("test")

	start := &StartEvent{}
	start.Id = "StartEvent_" + randName()
	p.AppendElem(start)

	task := &UserTask{}
	task.Id = "ServiceTask_" + randName()
	p.AfterInsertElem(start.Id, task, nil)

	endEvent := &EndEvent{}
	endEvent.Id = "EndEvent_" + randName()
	p.AfterInsertElem(task.Id, endEvent, nil)

	plane := p.Draw()
	assert.Equal(t, len(plane.Shapes), 3)
	assert.Equal(t, len(plane.Edges), 2)

	t.Log(plane)
}
