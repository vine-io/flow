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
	"context"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vine-io/vine/core/broker/http"
	"github.com/vine-io/vine/core/registry/mdns"
	vserver "github.com/vine-io/vine/core/server"
	"github.com/vine-io/vine/core/server/grpc"
)

func randAddress() string {
	source := rand.NewSource(time.Now().UnixNano())
	return fmt.Sprintf("127.0.0.1:%d", rand.New(source).Int63n(50000)+int64(10000))
}

func testNewServer(t *testing.T, name, address string) *RpcServer {
	scheduler, _ := testNewScheduler(t)

	vbroker := http.NewBroker()
	vbroker.Init()

	reg := mdns.NewRegistry()
	reg.Init()

	s := grpc.NewServer(
		vserver.Name(name),
		vserver.Address(address),
		vserver.Broker(vbroker),
		vserver.Registry(reg),
	)

	err := s.Init()
	if !assert.NoError(t, err, "init vine service") {
		return nil
	}

	server, err := NewRPCServer(s, scheduler)

	assert.NoError(t, err, "create a new rpc server")

	err = s.Start()
	if !assert.NoError(t, err, "start vine service") {
		return nil
	}

	return server
}

func TestNewServer(t *testing.T) {
	name := "server"
	address := randAddress()
	rs := testNewServer(t, name, address)
	defer rs.Stop()
}

func testNewClient(t *testing.T, name, id string, address string) *Client {
	cfg := NewConfig(name, id, address)
	c, err := NewClient(cfg, nil)

	assert.NoError(t, err, "new client")

	return c
}

func TestClientPipe(t *testing.T) {
	name := "client"
	address := randAddress()
	rs := testNewServer(t, name, address)
	defer rs.Stop()

	client := testNewClient(t, name, "1", address)
	session, err := client.NewSession()
	assert.NoError(t, err, "new session")
	session.Close()
}

func TestClientCall(t *testing.T) {
	name := "client-call"
	address := randAddress()
	rs := testNewServer(t, name, address)
	defer rs.Stop()

	Load(&Empty{}, &EmptyEcho{}, &EmptyStep{})

	client := testNewClient(t, name, "3", address)
	pipe, err := client.NewSession()
	if !assert.NoError(t, err, "new session") {
		return
	}
	defer pipe.Close()

	cfg := NewConfig(name, "4", address)
	c, err := NewClient(cfg, nil)
	if !assert.NoError(t, err, "new session") {
		return
	}

	ctx := context.TODO()
	rsp, err := c.Call(ctx, "3", GetTypePkgName(reflect.TypeOf(&EmptyEcho{})), []byte("hello"))
	if !assert.NoError(t, err, "test client call") {
		return
	}

	assert.Equal(t, []byte("hello"), rsp, "they should be equal")
}

func TestClientExecuteWorkflow(t *testing.T) {
	name := "execute-workflow"
	address := randAddress()
	rs := testNewServer(t, name, address)
	defer rs.Stop()

	Load(&Empty{}, &EmptyEcho{}, &EmptyStep{})

	client := testNewClient(t, name, "2", address)
	pipe, err := client.NewSession()
	assert.NoError(t, err, "new session")
	defer pipe.Close()

	items := map[string][]byte{
		"a": []byte("a"),
		"b": []byte("1"),
	}
	entity := &Empty{Name: "empty"}
	step := &EmptyStep{Client: "2"}
	//step2 := &EmptyStep{Client: "2"}

	wf := client.NewWorkflow(WithName("w"), WithId("2")).
		Items(items).
		Entities(entity).
		Steps(StepToWorkStep(step)).
		Build()

	ctx := context.TODO()
	watcher, err := client.ExecuteWorkflow(ctx, wf, true)

	if !assert.NoError(t, err, "execute workflow") {
		return
	}

	for {
		result, err := watcher.Next()
		if err == io.EOF {
			t.Logf("worflow done!")
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		t.Log(result)
	}
}

func TestClientAbortWorkflow(t *testing.T) {
	name := "abort-workflow"
	address := randAddress()
	rs := testNewServer(t, name, address)
	defer rs.Stop()

	Load(&Empty{}, &EmptyEcho{}, &EmptyStep{})

	client := testNewClient(t, name, "2", address)
	pipe, err := client.NewSession()
	assert.NoError(t, err, "new session")
	defer pipe.Close()

	items := map[string][]byte{
		"a": []byte("a"),
		"b": []byte("1"),
	}
	entity := &Empty{Name: "empty"}
	step := &EmptyStep{Client: "2"}

	wf := client.NewWorkflow(WithName("w"), WithId("2")).
		Items(items).
		Entities(entity).
		Steps(StepToWorkStep(step)).
		Build()

	ctx := context.TODO()
	watcher, err := client.ExecuteWorkflow(ctx, wf, true)

	if !assert.NoError(t, err, "execute workflow") {
		return
	}

	a := 0
	abort := false

	for {
		result, err := watcher.Next()
		if err == io.EOF {
			t.Logf("worflow done!")
			return
		}

		a += 1
		if a > 2 && !abort {
			err = client.AbortWorkflow(ctx, "2")
			if !assert.NoError(t, err, "abort workflow") {
				return
			}
			abort = true
		}

		if err != nil {
			t.Fatal(err)
		}
		t.Log(result)
	}
}
