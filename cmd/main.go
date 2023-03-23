package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	json "github.com/json-iterator/go"
	"github.com/vine-io/flow"
	"github.com/vine-io/flow/api"
)

var sum int32

func myFunc(i interface{}) {
	n := i.(int32)
	atomic.AddInt32(&sum, n)
	fmt.Printf("run with %d\n", n)
}

func demoFunc() {
	time.Sleep(10 * time.Millisecond)
	fmt.Println("Hello World!")
}

var _ flow.Entity = (*TestEntity)(nil)

type TestEntity struct {
	Name string `json:"name"`
}

func (t TestEntity) Metadata() map[string]string {
	//TODO implement me
	panic("implement me")
}

func (t TestEntity) OwnerReferences() []*api.OwnerReference {
	//TODO implement me
	panic("implement me")
}

func (t TestEntity) Marshal() ([]byte, error) {
	return json.Marshal(&t)
}

func (t *TestEntity) Unmarshal(data []byte) error {
	return json.Unmarshal(data, t)
}

type Person struct {
	Name string `json:"name"`
}

type SS struct {
	I      int32       `flow:"name:i"`
	Text   string      `flow:"name:text"`
	Entity *TestEntity `flow:"entity"`
	Person *Person     `flow:"name:person"`
}

type Tag struct {
	Name     string
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
		if strings.HasPrefix(part, "name:") {
			tag.Name = strings.TrimPrefix(part, "name:")
		}
	}

	return
}

func setField(vField reflect.Value, value []byte) {
	switch vField.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, _ := strconv.ParseInt(string(value), 10, 64)
		vField.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, _ := strconv.ParseUint(string(value), 10, 64)
		vField.SetUint(v)
	case reflect.String:
		vField.SetString(string(value))
	case reflect.Ptr:
		v := reflect.New(vField.Type().Elem())
		vv := v.Interface()
		e := json.Unmarshal([]byte(value), &vv)
		if e == nil {
			vField.Set(v)
		}
	case reflect.Struct:
		v := reflect.New(vField.Type())
		vv := v.Interface()
		e := json.Unmarshal([]byte(value), &vv)
		if e == nil {
			vField.Set(v)
		}
	}
}

func main() {
	//e := &SS{}
	//typ := reflect.TypeOf(e).Elem()
	//vle := reflect.ValueOf(e).Elem()
	//if typ.Kind() == reflect.Ptr {
	//	typ = typ.Elem()
	//	vle = vle.Elem()
	//}
	//
	//fmt.Println(typ.PkgPath() + "." + typ.Name())
	//
	////vle.Field(0).SetString("aaa")
	////fmt.Println(vle.Field(0).Type().Name())
	//
	//data, _ := json.Marshal(&TestEntity{Name: "tt"})
	//pd, _ := json.Marshal(&Person{Name: "p1"})
	//m := map[string]string{"i": "1", "text": "hello", "person": string(pd)}
	//for i := 0; i < typ.NumField(); i++ {
	//	tField := typ.Field(i)
	//	if !tField.IsExported() {
	//		continue
	//	}
	//
	//	text, ok := tField.Tag.Lookup("flow")
	//	if !ok {
	//		continue
	//	}
	//
	//	tag, err := parseFlowTag(text)
	//	if err != nil {
	//		continue
	//	}
	//	vField := vle.Field(i)
	//	if tag.IsEntity {
	//		if _, ok := vField.Interface().(flow.Entity); ok {
	//			setField(vField, data)
	//		}
	//	}
	//	value, ok := m[tag.Name]
	//	if !ok {
	//		continue
	//	}
	//
	//	setField(vField, []byte(value))
	//}
	//
	//b, _ := json.Marshal(e)
	//fmt.Println(string(b))

	p := atomic.Bool{}
	p.Store(true)
	l := sync.Mutex{}
	cond := sync.NewCond(&l)

	go func() {
		time.Sleep(time.Second * 1)
		p.Swap(false)
		cond.L.Lock()
		cond.Signal()
		cond.L.Unlock()
	}()

	fmt.Println("wait")
	n := 0
	for p.Load() {
		cond.L.Lock()
		cond.Wait()
		cond.L.Unlock()
		n += 1
	}
	fmt.Println("done! n =", n)
}
