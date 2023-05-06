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
	"reflect"
	"sync"

	"github.com/vine-io/flow/api"
)

type EchoSet struct {
	sync.RWMutex
	em map[string]*api.Echo
}

func NewEchoSet() *EchoSet {
	return &EchoSet{em: map[string]*api.Echo{}}
}

func (s *EchoSet) Add(echo *api.Echo) {
	s.Lock()
	defer s.Unlock()
	s.em[echo.Name] = echo
}

func (s *EchoSet) Del(echo *api.Echo) {
	s.Lock()
	defer s.Unlock()
	delete(s.em, echo.Name)
}

func (s *EchoSet) Get(name string) (*api.Echo, bool) {
	s.RLock()
	echo, ok := s.em[name]
	s.RUnlock()
	return echo, ok
}

func (s *EchoSet) Contains(name string) bool {
	_, ok := s.Get(name)
	return ok
}

func (s *EchoSet) List() []*api.Echo {
	s.RLock()
	defer s.RUnlock()
	echoes := make([]*api.Echo, 0)
	for _, v := range s.em {
		echoes = append(echoes, v.DeepCopy())
	}
	return echoes
}

// Echo 描述一个具体的请求
type Echo interface {
	// Owner 所属 Entity 信息
	Owner() reflect.Type

	Call(ctx context.Context, data []byte) ([]byte, error)
	// Desc Echo 描述信息
	Desc() string
}

var _ Echo = (*EmptyEcho)(nil)

type EmptyEcho struct{}

func (e *EmptyEcho) Owner() reflect.Type {
	return reflect.TypeOf(new(Empty))
}

func (e *EmptyEcho) Call(ctx context.Context, data []byte) ([]byte, error) {
	return data, nil
}

func (e *EmptyEcho) Desc() string {
	return ""
}
