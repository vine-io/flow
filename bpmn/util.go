package bpmn

import (
	"reflect"

	"github.com/beevik/etree"
	"github.com/vine-io/pkg/xname"
)

func getAttr(attrs []etree.Attr, name string) (string, bool) {
	for _, attr := range attrs {
		if attr.FullKey() == name {
			return attr.Value, true
		}
	}
	return "", false
}

func getKind(v any) string {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name()
}

func randName() string {
	return xname.Gen(xname.C(7), xname.Lowercase(), xname.Digit())
}

func randShapeName(elem Element) string {
	prefix := ""
	switch elem.GetShape() {
	case ProcessShape:
		prefix = "Process"
	case StartEventShape, EndEventShape:
		prefix = "EndEvent"
	case FlowShape:
		prefix = "Flow"
	case TaskShape, ServiceTaskShape, UserTaskShape:
		prefix = "Activity"
	default:
		prefix = "Activity"
	}

	return prefix + "_" + randName()
}

func setIncoming(elem Element, incoming string) {
	if IsEndEvent(elem) {
		return
	}
	ints := elem.GetIncoming()
	if ints == nil {
		ints = make([]string, 0)
	}
	if IsGateway(elem) {
		ints = append(ints, incoming)
	} else {
		ints = []string{incoming}
	}
	elem.SetIncoming(ints)
}

func setOutgoing(elem Element, outgoing string) {
	if IsEndEvent(elem) {
		return
	}
	outs := elem.GetOutgoing()
	if outs == nil {
		outs = make([]string, 0)
	}
	if IsGateway(elem) {
		outs = append(outs, outgoing)
	} else {
		outs = []string{outgoing}
	}
	elem.SetOutgoing(outs)
}

func IsGateway(elem Element) bool {
	switch elem.GetShape() {
	case ExclusiveGatewayShape, InclusiveGatewayShape, ParallelGatewayShape:
		return true
	default:
		return false
	}
}

func IsStartEvent(elem Element) bool {
	return elem.GetShape() == StartEventShape
}

func IsEndEvent(elem Element) bool {
	return elem.GetShape() == EndEventShape
}
