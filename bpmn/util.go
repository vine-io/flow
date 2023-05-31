package bpmn

import (
	"reflect"

	"github.com/beevik/etree"
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
