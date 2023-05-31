package bpmn

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseXML(t *testing.T) {
	xmlText, err := os.ReadFile("../testdata/order-process-4.bpmn")
	assert.NoError(t, err)
	d, err := ParseXML(string(xmlText))
	if !assert.NoError(t, err) {
		return
	}

	t.Log(d)

	out, err := d.WriteToBytes()
	if !assert.NoError(t, err) {
		return
	}

	t.Log(string(out))

	assert.Equal(t, string(xmlText), string(out))
}

func TestEvent_Marshal(t *testing.T) {
	//se := StartEvent{baseEvent: baseEvent{
	//	Id:       "event_axf1o",
	//	Name:     "event_1",
	//	Incoming: []string{"flow_1"},
	//}}
	//
	//out, err := xml.Marshal(se)
	//assert.NoError(t, err)
	//
	//t.Log(string(out))
	//
	//soe := StartEvent{}
	//err = xml.Unmarshal(out, &soe)
	//assert.NoError(t, err)
	//
	//assert.Equal(t, se, soe)
	//
	//e := EndEvent{baseEvent: baseEvent{
	//	Id:       "event_axf1o",
	//	Name:     "event_1",
	//	Incoming: []string{"flow_1"},
	//}}
	//
	//out, err = xml.MarshalIndent(e, " ", "  ")
	//assert.NoError(t, err)
	//
	//t.Log(string(out))
	//
	//oe := EndEvent{}
	//err = xml.Unmarshal(out, &oe)
	//assert.NoError(t, err)
	//
	//assert.Equal(t, e, oe)
}
