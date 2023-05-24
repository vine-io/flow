package bpmn

import (
	"fmt"
	"testing"
)

func TestNewProcessBuilder(t *testing.T) {
	pb := NewProcessBuilder("test")

	gwId := "gateway_1"
	gw := ExclusiveGateway{}
	gw.SetID(gwId)

	gwId2 := "gateway_2"
	gw2 := ExclusiveGateway{}
	gw2.SetID(gwId2)

	us := ServiceTask{}
	usId := "servicetask_1"
	us.SetID(usId)
	p, err := pb.Start().
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

	fmt.Println(p.Beautify())
}
