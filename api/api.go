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
	"encoding/json"
	"math"
	"strings"
)

// revBytesLen is the byte length of a normal revision.
// First 8 bytes is the revision.main in big-endian format. The 9th byte
// is a '_'. The last 8 bytes is the revision.sub in big-endian format.
const (
	revBytesLen       = 8 + 1 + 8
	markedRevBytesLen = revBytesLen + 1
)

func NewRevision() *Revision {
	return &Revision{Main: 0, Sub: 0}
}

func (m *Revision) Add() {
	if m.Sub < math.MaxUint64 {
		m.Sub += 1
		return
	}
	m.Sub = 0
	m.Main += 1
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

func (m StepAction) Readably() string {
	switch m {
	case StepAction_SC_PREPARE:
		return "prepare"
	case StepAction_SC_COMMIT:
		return "commit"
	case StepAction_SC_ROLLBACK:
		return "rollback"
	case StepAction_SC_CANCEL:
		return "cancel"
	default:
		return "unknown"
	}
}

func (m *StepAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Readably())
}

func (m *StepAction) UnmarshalJSON(data []byte) error {
	switch strings.Trim(string(data), `"`) {
	case "prepare":
		*m = StepAction_SC_PREPARE
	case "commit":
		*m = StepAction_SC_COMMIT
	case "rollback":
		*m = StepAction_SC_ROLLBACK
	case "cancel":
		*m = StepAction_SC_CANCEL
	default:
		*m = StepAction_SA_UNKNOWN
	}
	return nil
}

func (m WorkflowMode) Readably() string {
	switch m {
	case WorkflowMode_WM_ABORT:
		return "abort"
	case WorkflowMode_WM_AUTO:
		return "auto"
	case WorkflowMode_WM_MANUAL:
		return "manual"
	case WorkflowMode_WM_HYBRID:
		return "hybrid"
	default:
		return "unknown"
	}
}

func (m *WorkflowMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Readably())
}

func (m *WorkflowMode) UnmarshalJSON(data []byte) error {
	switch strings.Trim(string(data), `"`) {
	case "abort":
		*m = WorkflowMode_WM_ABORT
	case "auto":
		*m = WorkflowMode_WM_AUTO
	case "manual":
		*m = WorkflowMode_WM_MANUAL
	case "hybrid":
		*m = WorkflowMode_WM_HYBRID
	default:
		*m = WorkflowMode_WM_UNKNOWN
	}
	return nil
}

func (m WorkflowState) Readably() string {
	switch m {
	case WorkflowState_SW_RUNNING:
		return "running"
	case WorkflowState_SW_ROLLBACK:
		return "rollback"
	case WorkflowState_SW_CANCEL:
		return "cancel"
	case WorkflowState_SW_SUCCESS:
		return "success"
	case WorkflowState_SW_WARN:
		return "warn"
	case WorkflowState_SW_FAILED:
		return "failed"
	default:
		return "unknown"
	}
}

func (m *WorkflowState) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Readably())
}

func (m *WorkflowState) UnmarshalJSON(data []byte) error {
	switch strings.Trim(string(data), `"`) {
	case "running":
		*m = WorkflowState_SW_RUNNING
	case "rollback":
		*m = WorkflowState_SW_ROLLBACK
	case "cancel":
		*m = WorkflowState_SW_CANCEL
	case "success":
		*m = WorkflowState_SW_SUCCESS
	case "warn":
		*m = WorkflowState_SW_WARN
	case "failed":
		*m = WorkflowState_SW_FAILED
	default:
		*m = WorkflowState_SW_UNKNOWN
	}
	return nil
}
