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
	"github.com/vine-io/apimachinery/runtime"
	"github.com/vine-io/apimachinery/schema"
)

type Entity interface {
	runtime.Object
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
}

var _ Entity = (*Empty)(nil)

type Empty struct{}

func (e *Empty) GetObjectKind() schema.ObjectKind {
	return &schema.EmptyObjectKind
}

func (e *Empty) DeepCopyObject() runtime.Object {
	return &Empty{}
}

func (e *Empty) DeepFromObject(o runtime.Object) {
	*e = *o.DeepCopyObject().(*Empty)
}

func (e *Empty) Marshal() ([]byte, error) {
	return []byte(""), nil
}

func (e *Empty) Unmarshal(data []byte) error {
	return nil
}
