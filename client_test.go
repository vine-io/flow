package flow

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type InjectData struct {
	Field1 string `json:"field1"`
}

type MyTestEcho struct {
	EmptyEcho

	Data *InjectData `inject:""`
}

type MyTestStep struct {
	EmptyStep

	Data *InjectData `inject:""`
}

func TestClientStore_Populate(t *testing.T) {
	store := NewClientStore()

	data := &InjectData{Field1: "field1"}

	if err := store.Provides(data); err != nil {
		t.Fatal(err)
	}

	store.Load(&MyTestEcho{}, &MyTestStep{})

	ev, err := store.PopulateEcho(GetTypePkgName(reflect.TypeOf(new(MyTestEcho))))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, ev.(*MyTestEcho).Data, data)

	sv, err := store.PopulateStep(GetTypePkgName(reflect.TypeOf(new(MyTestStep))))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, sv.(*MyTestStep).Data, data)
}
