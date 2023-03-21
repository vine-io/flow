package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vine-io/flow/api"
)

func TestNewOptions(t *testing.T) {
	options := NewOptions()
	assert.NotEqual(t, options.Name, "", "they should not be equal")
	assert.NotEqual(t, options.Wid, "", "they should not be equal")
	assert.Equal(t, options.Mode, api.WorkflowMode_WM_AUTO, "they should be equal")
	assert.Equal(t, options.MaxRetries, int32(0), "they should be equal")

	options = NewOptions(
		WithId("id"),
		WithName("w"),
		WithMode(api.WorkflowMode_WM_HYBRID),
		WithMaxRetry(2),
	)

	assert.Equal(t, options.Wid, "id", "they should be equal")
	assert.Equal(t, options.Name, "w", "they should be equal")
	assert.Equal(t, options.Mode, api.WorkflowMode_WM_HYBRID, "they should be equal")
	assert.Equal(t, options.MaxRetries, int32(2), "they should be equal")
}

func TestNewBuilder(t *testing.T) {
	b := NewBuilder(WithId("id"),
		WithName("w"),
		WithMode(api.WorkflowMode_WM_HYBRID),
		WithMaxRetry(2))

	spec := b.spec.Option
	assert.Equal(t, spec.Wid, "id", "they should be equal")
	assert.Equal(t, spec.Name, "w", "they should be equal")
	assert.Equal(t, spec.Mode, api.WorkflowMode_WM_HYBRID, "they should be equal")
	assert.Equal(t, spec.MaxRetries, int32(2), "they should be equal")
}

func TestWorkflowBuilder(t *testing.T) {
	b := NewBuilder()

	items := map[string][]byte{
		"a": []byte("a"),
		"b": []byte("b"),
	}
	b.Items(items)

	entities := []Entity{&Empty{}}
	b.Entities(entities)

	steps := []Step{&EmptyStep{}}
	b.Steps(steps)

	out := b.Build()

	assert.Equal(t, out.Items, items, "they should be equal")
	assert.Equal(t, out.Entities[0], EntityToAPI(&Empty{}), "they should be equal")
	assert.Equal(t, out.Steps[0], StepToWorkStep(&EmptyStep{}), "they should be equal")
}
