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
	"sync"

	json "github.com/json-iterator/go"
	"github.com/vine-io/flow/api"
)

type EntitySet struct {
	sync.RWMutex
	em map[string]*api.Entity
}

func NewEntitySet() *EntitySet {
	return &EntitySet{em: map[string]*api.Entity{}}
}

func (s *EntitySet) Add(entity *api.Entity) {
	s.Lock()
	defer s.Unlock()
	s.em[entity.Kind] = entity
}

func (s *EntitySet) Del(entity *api.Entity) {
	s.Lock()
	defer s.Unlock()
	delete(s.em, entity.Kind)
}

func (s *EntitySet) Get(kind string) (*api.Entity, bool) {
	s.RLock()
	entity, ok := s.em[kind]
	s.RUnlock()
	return entity, ok
}

func (s *EntitySet) Contains(name string) bool {
	_, ok := s.Get(name)
	return ok
}

func (s *EntitySet) List() []*api.Entity {
	s.RLock()
	defer s.RUnlock()
	entities := make([]*api.Entity, 0)
	for _, v := range s.em {
		entities = append(entities, v.DeepCopy())
	}
	return entities
}

// Entity 描述工作流中的具体资源，是工作流中的执行单元
type Entity interface {
	// OwnerReferences Entity 之间的依赖信息
	OwnerReferences() []*api.OwnerReference
	// Marshal Entity 序列化
	Marshal() ([]byte, error)
	// Unmarshal Entity 反序列化
	Unmarshal(data []byte) error
	// String Entity 说明
	String() string
}

var _ Entity = (*Empty)(nil)

type Empty struct{}

func (e *Empty) OwnerReferences() []*api.OwnerReference {
	return nil
}

func (e *Empty) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *Empty) Unmarshal(data []byte) error {
	if len(data) == 0 {
		*e = Empty{}
		return nil
	}
	return json.Unmarshal(data, e)
}

func (e *Empty) String() string {
	return "empty entity"
}
