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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	json "github.com/json-iterator/go"
	"github.com/vine-io/flow/api"
	"github.com/vine-io/pkg/xname"
)

// GetTypePkgName returns the package path and kind for object based on reflect.Type.
func GetTypePkgName(p reflect.Type) string {
	switch p.Kind() {
	case reflect.Ptr:
		p = p.Elem()
	}
	return p.PkgPath() + "." + p.Name()
}

type TagKind int

const (
	TagKindCtx TagKind = iota + 1
	TagKindArgs
)

type Tag struct {
	Name     string
	Kind     TagKind
	IsEntity bool
}

func parseFlowTag(text string) (tag *Tag, err error) {
	parts := strings.Split(text, ";")
	tag = &Tag{}
	for _, part := range parts {
		part = strings.Trim(part, " ")
		if len(part) == 0 {
			continue
		}
		if part == "entity" {
			tag.IsEntity = true
		}
		if strings.HasPrefix(part, "ctx:") {
			tag.Kind = TagKindCtx
			tag.Name = strings.TrimPrefix(part, "ctx:")
		}
		if strings.HasPrefix(part, "args:") {
			tag.Kind = TagKindArgs
			tag.Name = strings.TrimPrefix(part, "args:")
		}
	}

	return
}

func setField(vField reflect.Value, value string) error {
	switch vField.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, e := strconv.ParseInt(value, 10, 64)
		if e != nil {
			return e
		}
		vField.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, e := strconv.ParseUint(value, 10, 64)
		if e != nil {
			return e
		}
		vField.SetUint(v)
	case reflect.String:
		vField.SetString(value)
	case reflect.Ptr:
		v := reflect.New(vField.Type().Elem())
		vv := v.Interface()

		var e error
		e = json.Unmarshal([]byte(value), &vv)
		if e != nil {
			return e
		}
		vField.Set(v)
	case reflect.Struct:
		v := reflect.New(vField.Type())
		vv := v.Interface()

		var e error
		e = json.Unmarshal([]byte(value), &vv)
		if e != nil {
			return e
		}
		vField.Set(v)
	}

	return nil
}

func InjectTypeFields(t any, items, args map[string]string, entityData []byte) error {
	typ := reflect.TypeOf(t)
	vle := reflect.ValueOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		vle = vle.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		tField := typ.Field(i)
		if !tField.IsExported() {
			continue
		}

		text, ok := tField.Tag.Lookup("flow")
		if !ok {
			continue
		}

		tag, err := parseFlowTag(text)
		if err != nil {
			continue
		}
		vField := vle.Field(i)
		if tag.IsEntity {
			field := reflect.New(vField.Type())
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}
			obj := field.Interface()
			if _, ok1 := obj.(Entity); ok1 {
				if err = setField(vField, string(entityData)); err != nil {
					return fmt.Errorf("inject to entity field '%s': %v", tField.Name, err)
				}
				continue
			}
			vField.Set(field)
		}

		switch tag.Kind {
		case TagKindCtx:
			value, ok := items[tag.Name]
			if !ok {
				continue
			}

			if err = setField(vField, value); err != nil {
				return fmt.Errorf("inject to context field '%s': %v", tField.Name, err)
			}
		case TagKindArgs:
			value, ok := args[tag.Name]
			if !ok {
				continue
			}

			if err = setField(vField, value); err != nil {
				return fmt.Errorf("inject to args field '%s': %v", tField.Name, err)
			}
		}
	}

	return nil
}

func ExtractTypeField(t any) ([]byte, error) {
	typ := reflect.TypeOf(t)
	v := reflect.ValueOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		v = v.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		tField := typ.Field(i)
		if !tField.IsExported() {
			continue
		}

		text, ok := tField.Tag.Lookup("flow")
		if !ok {
			continue
		}

		tag, err := parseFlowTag(text)
		if err != nil {
			continue
		}
		if tag.IsEntity {
			vField := v.Field(i)
			if vv, ok := vField.Interface().(Entity); ok {
				return json.Marshal(vv)
			}
		}
	}

	return nil, fmt.Errorf("entity field not found")
}

func ExtractFields(t any) []string {
	typ := reflect.TypeOf(t)
	v := reflect.ValueOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		v = v.Elem()
	}

	items := make([]string, 0)
	for i := 0; i < typ.NumField(); i++ {
		tField := typ.Field(i)
		if !tField.IsExported() {
			continue
		}

		text, ok := tField.Tag.Lookup("flow")
		if !ok {
			continue
		}

		tag, err := parseFlowTag(text)
		if err != nil {
			continue
		}
		if tag.Name != "" {
			items = append(items, tag.Name)
		}
	}

	return items
}

func SetTypeEntityField(t any, data []byte) error {
	typ := reflect.TypeOf(t)
	v := reflect.ValueOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		v = v.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		tField := typ.Field(i)
		if !tField.IsExported() {
			continue
		}

		text, ok := tField.Tag.Lookup("flow")
		if !ok {
			continue
		}

		tag, err := parseFlowTag(text)
		if err != nil {
			continue
		}
		if tag.IsEntity {
			vField := v.Field(i)
			if vv, ok := vField.Interface().(Entity); ok {
				return vv.Unmarshal(data)
			}
		}
	}

	return fmt.Errorf("entity field not found")
}

// EntityToAPI returns a new instance *api.Entity based on the specified Entity interface implementation.
func EntityToAPI(entity Entity) *api.Entity {
	e := &api.Entity{
		Kind:            GetTypePkgName(reflect.TypeOf(entity)),
		OwnerReferences: entity.OwnerReferences(),
		Workers:         map[string]*api.Worker{},
		Describe:        entity.Desc(),
	}

	raw, _ := json.Marshal(entity)
	e.Raw = string(raw)

	return e
}

// EchoToAPI returns a new instance *api.Echo based on the specified Echo interface implementation.
func EchoToAPI(echo Echo) *api.Echo {
	e := &api.Echo{
		Name:     GetTypePkgName(reflect.TypeOf(echo)),
		Entity:   GetTypePkgName(echo.Owner()),
		Workers:  map[string]*api.Worker{},
		Describe: echo.Desc(),
	}

	return e
}

// StepToAPI returns a new instance *api.Step based on the specified Step interface implementation.
func StepToAPI(step Step) *api.Step {
	s := &api.Step{
		Name:     GetTypePkgName(reflect.TypeOf(step)),
		Entity:   GetTypePkgName(step.Owner()),
		Workers:  map[string]*api.Worker{},
		Describe: step.Desc(),
	}

	return s
}

// StepToWorkStep returns a new instance *api.WorkflowStep based on the specified Step interface implementation.
func StepToWorkStep(step Step, worker string) *api.WorkflowStep {
	s := &api.WorkflowStep{
		Name:     GetTypePkgName(reflect.TypeOf(step)),
		Uid:      "Step_" + xname.Gen6(),
		Describe: step.Desc(),
		Worker:   worker,
		Entity:   GetTypePkgName(step.Owner()),
		Injects:  ExtractFields(step),
	}

	return s
}

func zeebeEscape(text string) string {
	text = strings.ReplaceAll(text, "/", "__")
	text = strings.ReplaceAll(text, ".", "_")
	return text
}

func zeebeUnEscape(text string) string {
	text = strings.ReplaceAll(text, "__", "/")
	text = strings.ReplaceAll(text, "_", ".")
	return text
}
