// Copyright 2023 Lack (xingyys@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import "github.com/olive-io/bpmn/schema"

type ServiceTaskBuilder struct {
	id       string
	name     string
	taskType string
	task     *schema.ServiceTask
}

func NewServiceTaskBuilder(name, taskType string) *ServiceTaskBuilder {
	task := schema.DefaultServiceTask()
	task.ExtensionElementsField = &schema.ExtensionElements{
		TaskHeaderField: &schema.TaskHeader{Header: make([]*schema.Item, 0)},
		PropertiesField: &schema.Properties{Property: make([]*schema.Item, 0)},
	}
	b := &ServiceTaskBuilder{
		id:       *randShapeName(&task),
		name:     name,
		taskType: taskType,
		task:     &task,
	}

	return b
}

func (b *ServiceTaskBuilder) SetId(id string) *ServiceTaskBuilder {
	b.id = id
	return b
}

func (b *ServiceTaskBuilder) SetHeader(name string, value any) *ServiceTaskBuilder {
	vv, vt := parseItemValue(value)
	b.task.ExtensionElementsField.TaskHeaderField.Header = append(b.task.ExtensionElementsField.TaskHeaderField.Header, &schema.Item{
		Name:  name,
		Value: vv,
		Type:  vt,
	})
	return b
}

func (b *ServiceTaskBuilder) SetProperty(name string, value any) *ServiceTaskBuilder {
	vv, vt := parseItemValue(value)
	b.task.ExtensionElementsField.PropertiesField.Property = append(b.task.ExtensionElementsField.PropertiesField.Property, &schema.Item{
		Name:  name,
		Value: vv,
		Type:  vt,
	})
	return b
}

func (b *ServiceTaskBuilder) Out() *schema.ServiceTask {
	b.task.SetId(&b.id)
	b.task.SetName(&b.name)
	b.task.ExtensionElementsField.TaskDefinitionField = &schema.TaskDefinition{Type: b.taskType}
	return b.task
}
