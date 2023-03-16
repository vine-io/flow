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

package api

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

// TestRevision tests that revision could be encoded to and decoded from
// bytes slice. Moreover, the lexicographical order of its byte slice representation
// follows the order of (main, sub).
func TestRevision(t *testing.T) {
	tests := []Revision{
		// order in (main, sub)
		{},
		{Main: 1, Sub: 0},
		{Main: 1, Sub: 1},
		{Main: 2, Sub: 0},
		{Main: math.MaxUint64, Sub: math.MaxUint64},
	}

	bs := make([][]byte, len(tests))
	for i, tt := range tests {
		b := tt.ToBytes()
		bs[i] = b

		if grev := BytesToRev(b); !reflect.DeepEqual(grev, tt) {
			t.Errorf("#%d: revision = %+v, want %+v", i, grev, tt)
		}
	}

	for i := 0; i < len(tests)-1; i++ {
		if bytes.Compare(bs[i], bs[i+1]) >= 0 {
			t.Errorf("#%d: %v (%+v) should be smaller than %v (%+v)", i, bs[i], tests[i], bs[i+1], tests[i+1])
		}
	}
}
