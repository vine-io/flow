package bpmn

import (
	"fmt"
	"testing"
)

func TestNewDefinitionBuilder(t *testing.T) {
	db := NewDefinitionBuilder("test")

	gwId := "gateway_1"
	gw := ExclusiveGateway{}
	gw.SetID(gwId)

	gwId2 := "gateway_2"
	gw2 := ExclusiveGateway{}
	gw2.SetID(gwId2)

	us := ServiceTask{}
	usId := "servicetask_1"
	us.SetID(usId)
	d, err := db.Start().
		AddModel(&ServiceTask{}).
		AddModel(&gw).
		AddModel(&ServiceTask{}).
		AddModel(&gw2).
		End().
		Seek(gwId).
		AddModel(&us).
		Link(usId, gwId2, "", nil).
		Build()

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(d.Beautify())
}
