package bpmn

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseXML(t *testing.T) {
	xmlText, err := os.ReadFile("../testdata/order-process-4.bpmn")
	assert.NoError(t, err)
	d, err := FromXML(string(xmlText))
	if !assert.NoError(t, err) {
		return
	}

	t.Log(d)

	out, err := d.WriteToBytes()
	if !assert.NoError(t, err) {
		return
	}

	t.Log(string(out))
}

func TestBuilder(t *testing.T) {
	b := NewBuilder("test")
	out, err := b.Start().
		AppendElem(NewServiceTask("testTask", "test_service")).
		End().
		Out()

	if !assert.NoError(t, err) {
		return
	}

	data, err := out.AutoLayout().WriteToBytes()
	if !assert.NoError(t, err) {
		return
	}

	t.Log(string(data))
}
