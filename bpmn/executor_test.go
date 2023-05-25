package bpmn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunner(t *testing.T) {
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

	assert.NoError(t, err)

	runner := NewRunner(d)
	ctx := context.TODO()

	err = runner.Run(ctx)
	assert.NoError(t, err)
}
