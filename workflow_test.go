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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vine-io/flow/api"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func testNewEtcdClient(t *testing.T) *clientv3.Client {
	conn, err := clientv3.New(clientv3.Config{
		Endpoints:            []string{"127.0.0.1:2379"},
		AutoSyncInterval:     0,
		DialTimeout:          time.Second * 3,
		DialKeepAliveTime:    time.Second * 30,
		DialKeepAliveTimeout: time.Second * 15,
	})

	assert.NoError(t, err, "connect to etcd server")

	return conn
}

func testNewScheduler(t *testing.T) (*Scheduler, func()) {
	conn := testNewEtcdClient(t)
	s, err := NewScheduler(conn, 10)

	cancel := func() {
		conn.Close()
		s.Stop(true)
	}

	assert.NoError(t, err, "create a new Scheduler instance")

	return s, cancel
}

func TestNewScheduler(t *testing.T) {
	_, cancel := testNewScheduler(t)
	defer cancel()
}

func TestExecuteWorkflow(t *testing.T) {
	s, _ := testNewScheduler(t)

	ps := NewPipeSet()

	pipe := testNewPipe(t)

	ps.Add(pipe)
	defer ps.Del(pipe)

	items := map[string][]byte{
		"a": []byte("a"),
		"b": []byte("1"),
	}
	entity := &Empty{Name: "empty"}
	step := &EmptyStep{}

	entities := []*api.Entity{EntityToAPI(entity)}
	echoes := []*api.Echo{}

	ws := StepToAPI(step)
	ws.Clients = map[string]*api.Client{"1": &api.Client{Id: "1"}}
	steps := []*api.Step{ws}
	s.Register(entities, echoes, steps)

	b := NewBuilder(WithId("1"), WithName("test")).
		Items(items).
		Entities(entity).
		Steps(StepToWorkStep(step))
	w := b.Build()

	err := s.ExecuteWorkflow(w, ps)
	if err != nil {
		t.Fatal(err)
	}

	s.Stop(true)
}