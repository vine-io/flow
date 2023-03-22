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
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vine-io/vine/core/broker/http"
	"github.com/vine-io/vine/core/registry/mdns"
	vserver "github.com/vine-io/vine/core/server"
	"github.com/vine-io/vine/core/server/grpc"
)

const (
	cid     = "1"
	name    = "vine.flow"
	address = "127.0.0.1:44325"
)

func testNewServer(t *testing.T) *RpcServer {
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
	testNewServer(t)
}

func testNewClient(t *testing.T) *Client {

	cfg := NewConfig(name, cid, address)
	c, err := NewClient(cfg)

	assert.NoError(t, err, "new client")

	return c
}

func TestClientPipe(t *testing.T) {
	server := testNewServer(t)
	defer server.Stop()

	client := testNewClient(t)
	session, err := client.NewSession()
	assert.NoError(t, err, "new session")
	defer session.Close()
}

func TestClientCall(t *testing.T) {
	server := testNewServer(t)
	defer server.Stop()

	Load(&Empty{}, &EmptyEcho{}, &EmptyStep{})

	client := testNewClient(t)
	pipe, err := client.NewSession()
	assert.NoError(t, err, "new session")
	defer pipe.Close()

	cfg := NewConfig(name, "2", address)
	c, err := NewClient(cfg)
	assert.NoError(t, err, "new session")

	ctx := context.TODO()
	rsp, err := c.Call(ctx, "1", GetTypePkgName(reflect.TypeOf(&EmptyEcho{})), []byte("hello"))
	if !assert.NoError(t, err, "test client call") {
		return
	}

	assert.Equal(t, []byte("hello"), rsp, "they should be equal")
}

func TestClientExecuteWorkflow(t *testing.T) {
	server := testNewServer(t)
	defer server.Stop()

	Load(&Empty{}, &EmptyEcho{}, &EmptyStep{})

	client := testNewClient(t)
	pipe, err := client.NewSession()
	assert.NoError(t, err, "new session")
	defer pipe.Close()

	items := map[string][]byte{
		"a": []byte("a"),
		"b": []byte("1"),
	}
	entity := &Empty{Name: "empty"}
	step := &EmptyStep{}

	wf := pipe.NewWorkflow(WithName("w"), WithId("1")).
		Items(items).
		Entities(entity).
		Steps(step).
		Build()

	ctx := context.TODO()
	watcher, err := pipe.ExecuteWorkflow(ctx, wf, true)

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
