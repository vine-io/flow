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
	"encoding/binary"
	"math"
)

// revBytesLen is the byte length of a normal revision.
// First 8 bytes is the revision.main in big-endian format. The 9th byte
// is a '_'. The last 8 bytes is the revision.sub in big-endian format.
const (
	revBytesLen       = 8 + 1 + 8
	markedRevBytesLen = revBytesLen + 1
)

func (m *Revision) Add() {
	if m.Main < math.MaxUint64 {
		m.Sub += 1
		return
	}

	m.Main += 1
	m.Sub = 0
}

func (m *Revision) IsZone() bool {
	return m.Main == 0 && m.Sub == 0
}

func (m *Revision) GreaterThan(b *Revision) bool {
	if m.Main > b.Main {
		return true
	}
	if m.Main < b.Main {
		return false
	}
	return m.Sub > b.Sub
}

func (m *Revision) ToBytes() []byte {
	b := make([]byte, revBytesLen, markedRevBytesLen)
	binary.BigEndian.PutUint64(b, m.Main)
	b[8] = '_'
	binary.BigEndian.PutUint64(b[9:], m.Sub)
	return b
}

func BytesToRev(bytes []byte) Revision {
	return Revision{
		Main: binary.BigEndian.Uint64(bytes[0:8]),
		Sub:  binary.BigEndian.Uint64(bytes[9:]),
	}
}
