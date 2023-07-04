// MIT License
//
// Copyright (c) 2023 Lack
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
	assert.Equal(t, options.MaxRetries, int32(0), "they should be equal")

	options = NewOptions(
		WithId("id"),
		WithName("w"),
		WithMaxRetry(2),
	)

	assert.Equal(t, options.Wid, "id", "they should be equal")
	assert.Equal(t, options.Name, "w", "they should be equal")
	assert.Equal(t, options.MaxRetries, int32(2), "they should be equal")
}

func TestNewBuilder(t *testing.T) {
	b := NewBuilder(WithId("id"),
		WithName("w"),
		WithMaxRetry(2))

	spec := b.spec.Option
	assert.Equal(t, spec.Wid, "id", "they should be equal")
	assert.Equal(t, spec.Name, "w", "they should be equal")
	assert.Equal(t, spec.MaxRetries, int32(2), "they should be equal")
}

func TestWorkflowBuilder(t *testing.T) {
	b := NewBuilder()

	items := map[string]any{
		"a":      "a",
		"b":      "b",
		"entity": &Empty{},
	}
	b.Items(items)

	steps := []*api.WorkflowStep{StepToWorkStep(&TestStep{}, "1")}
	b.Steps(steps...)

	out := b.Build()

	assert.Equal(t, out.Items, items, "they should be equal")
	assert.Equal(t, out.Steps[0].Worker, StepToWorkStep(&TestStep{}, "1").Worker, "they should be equal")
}

func TestWorkflowBuilder_ToBpmn(t *testing.T) {
	b := NewBuilder()

	items := map[string]any{
		"a": "a",
		"b": "b",
	}
	b.Items(items)
	b.Item("entity", &Empty{})
	b.Item("entity1", &Empty{})

	steps := []*api.WorkflowStep{StepToWorkStep(&TestStep{}, "1")}
	b.Steps(steps...)

	d, pros, err := b.ToBpmn()
	if !assert.NoError(t, err) {
		return
	}

	data, err := d.WriteToBytes()
	if !assert.NoError(t, err) {
		return
	}

	t.Log(string(data))
	t.Log(pros)
}
