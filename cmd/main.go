package main

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
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

type SS struct {
	E ants.Pool
}

func main() {
	e := &SS{}
	typ := reflect.TypeOf(e).Elem()
	vle := reflect.ValueOf(e).Elem()

	fmt.Println(typ.PkgPath() + "." + typ.Name())

	//vle.Field(0).SetString("aaa")
	fmt.Println(vle.Field(0).Type().Name())

	//p, err := ants.NewPool(100,
	//	ants.WithPreAlloc(true),
	//)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer p.Release()
	//
	//wg := sync.WaitGroup{}
	//wg.Add(1)
	//_ = p.Submit(func() {
	//	defer wg.Done()
	//	fmt.Println("hello world")
	//})
	////wg.Wait()
	//fmt.Println(p.Running())
}
