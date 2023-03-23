// Code generated by proto-gen-validator
// source: github.com/vine-io/flow/api/rpc.proto

package api

import (
	fmt "fmt"
	is "github.com/vine-io/vine/util/is"
)

func (m *ListWorkerRequest) Validate() error {
	return m.ValidateE("")
}

func (m *ListWorkerRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *ListWorkerResponse) Validate() error {
	return m.ValidateE("")
}

func (m *ListWorkerResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *ListRegistryRequest) Validate() error {
	return m.ValidateE("")
}

func (m *ListRegistryRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *ListRegistryResponse) Validate() error {
	return m.ValidateE("")
}

func (m *ListRegistryResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *RegisterRequest) Validate() error {
	return m.ValidateE("")
}

func (m *RegisterRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if len(m.Id) == 0 {
		errs = append(errs, fmt.Errorf("field '%sid' is required", prefix))
	}
	return is.MargeErr(errs...)
}

func (m *RegisterResponse) Validate() error {
	return m.ValidateE("")
}

func (m *RegisterResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *CallRequest) Validate() error {
	return m.ValidateE("")
}

func (m *CallRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if len(m.Id) == 0 {
		errs = append(errs, fmt.Errorf("field '%sid' is required", prefix))
	}
	if len(m.Name) == 0 {
		errs = append(errs, fmt.Errorf("field '%sname' is required", prefix))
	}
	if len(m.Request) == 0 {
		errs = append(errs, fmt.Errorf("field '%srequest' is required", prefix))
	}
	return is.MargeErr(errs...)
}

func (m *CallResponse) Validate() error {
	return m.ValidateE("")
}

func (m *CallResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *StepRequest) Validate() error {
	return m.ValidateE("")
}

func (m *StepRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if int32(m.Action) != 0 {
		if !is.In([]int32{0, 1, 2, 3, 4}, int32(m.Action)) {
			errs = append(errs, fmt.Errorf("field '%saction' must in '[0, 1, 2, 3, 4]'", prefix))
		}
	}
	return is.MargeErr(errs...)
}

func (m *StepResponse) Validate() error {
	return m.ValidateE("")
}

func (m *StepResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *PipeRequest) Validate() error {
	return m.ValidateE("")
}

func (m *PipeRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if int32(m.Topic) != 0 {
		if !is.In([]int32{0, 1, 2, 3}, int32(m.Topic)) {
			errs = append(errs, fmt.Errorf("field '%stopic' must in '[0, 1, 2, 3]'", prefix))
		}
	}
	return is.MargeErr(errs...)
}

func (m *PipeResponse) Validate() error {
	return m.ValidateE("")
}

func (m *PipeResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if int32(m.Topic) != 0 {
		if !is.In([]int32{0, 1, 2, 3}, int32(m.Topic)) {
			errs = append(errs, fmt.Errorf("field '%stopic' must in '[0, 1, 2, 3]'", prefix))
		}
	}
	return is.MargeErr(errs...)
}

func (m *PipeCallRequest) Validate() error {
	return m.ValidateE("")
}

func (m *PipeCallRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *PipeCallResponse) Validate() error {
	return m.ValidateE("")
}

func (m *PipeCallResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *PipeStepRequest) Validate() error {
	return m.ValidateE("")
}

func (m *PipeStepRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if int32(m.Action) != 0 {
		if !is.In([]int32{0, 1, 2, 3, 4}, int32(m.Action)) {
			errs = append(errs, fmt.Errorf("field '%saction' must in '[0, 1, 2, 3, 4]'", prefix))
		}
	}
	return is.MargeErr(errs...)
}

func (m *PipeStepResponse) Validate() error {
	return m.ValidateE("")
}

func (m *PipeStepResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *ListWorkflowRequest) Validate() error {
	return m.ValidateE("")
}

func (m *ListWorkflowRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *ListWorkflowResponse) Validate() error {
	return m.ValidateE("")
}

func (m *ListWorkflowResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *RunWorkflowRequest) Validate() error {
	return m.ValidateE("")
}

func (m *RunWorkflowRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if m.Workflow == nil {
		errs = append(errs, fmt.Errorf("field '%sworkflow' is required", prefix))
	} else {
		errs = append(errs, m.Workflow.ValidateE(prefix+"workflow."))
	}
	return is.MargeErr(errs...)
}

func (m *RunWorkflowResponse) Validate() error {
	return m.ValidateE("")
}

func (m *RunWorkflowResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *InspectWorkflowRequest) Validate() error {
	return m.ValidateE("")
}

func (m *InspectWorkflowRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if len(m.Wid) == 0 {
		errs = append(errs, fmt.Errorf("field '%swid' is required", prefix))
	}
	return is.MargeErr(errs...)
}

func (m *InspectWorkflowResponse) Validate() error {
	return m.ValidateE("")
}

func (m *InspectWorkflowResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *AbortWorkflowRequest) Validate() error {
	return m.ValidateE("")
}

func (m *AbortWorkflowRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *AbortWorkflowResponse) Validate() error {
	return m.ValidateE("")
}

func (m *AbortWorkflowResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *PauseWorkflowRequest) Validate() error {
	return m.ValidateE("")
}

func (m *PauseWorkflowRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *PauseWorkflowResponse) Validate() error {
	return m.ValidateE("")
}

func (m *PauseWorkflowResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *ResumeWorkflowRequest) Validate() error {
	return m.ValidateE("")
}

func (m *ResumeWorkflowRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *ResumeWorkflowResponse) Validate() error {
	return m.ValidateE("")
}

func (m *ResumeWorkflowResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *WatchWorkflowRequest) Validate() error {
	return m.ValidateE("")
}

func (m *WatchWorkflowRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if len(m.Wid) == 0 {
		errs = append(errs, fmt.Errorf("field '%swid' is required", prefix))
	}
	if len(m.Cid) == 0 {
		errs = append(errs, fmt.Errorf("field '%scid' is required", prefix))
	}
	return is.MargeErr(errs...)
}

func (m *WatchWorkflowResponse) Validate() error {
	return m.ValidateE("")
}

func (m *WatchWorkflowResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *StepGetRequest) Validate() error {
	return m.ValidateE("")
}

func (m *StepGetRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if len(m.Wid) == 0 {
		errs = append(errs, fmt.Errorf("field '%swid' is required", prefix))
	}
	if len(m.Step) == 0 {
		errs = append(errs, fmt.Errorf("field '%sstep' is required", prefix))
	}
	if len(m.Key) == 0 {
		errs = append(errs, fmt.Errorf("field '%skey' is required", prefix))
	}
	return is.MargeErr(errs...)
}

func (m *StepGetResponse) Validate() error {
	return m.ValidateE("")
}

func (m *StepGetResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *StepPutRequest) Validate() error {
	return m.ValidateE("")
}

func (m *StepPutRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if len(m.Wid) == 0 {
		errs = append(errs, fmt.Errorf("field '%swid' is required", prefix))
	}
	if len(m.Step) == 0 {
		errs = append(errs, fmt.Errorf("field '%sstep' is required", prefix))
	}
	if len(m.Key) == 0 {
		errs = append(errs, fmt.Errorf("field '%skey' is required", prefix))
	}
	if len(m.Value) == 0 {
		errs = append(errs, fmt.Errorf("field '%svalue' is required", prefix))
	}
	return is.MargeErr(errs...)
}

func (m *StepPutResponse) Validate() error {
	return m.ValidateE("")
}

func (m *StepPutResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}

func (m *StepTraceRequest) Validate() error {
	return m.ValidateE("")
}

func (m *StepTraceRequest) ValidateE(prefix string) error {
	errs := make([]error, 0)
	if len(m.Wid) == 0 {
		errs = append(errs, fmt.Errorf("field '%swid' is required", prefix))
	}
	if len(m.Step) == 0 {
		errs = append(errs, fmt.Errorf("field '%sstep' is required", prefix))
	}
	if len(m.Text) == 0 {
		errs = append(errs, fmt.Errorf("field '%stext' is required", prefix))
	}
	return is.MargeErr(errs...)
}

func (m *StepTraceResponse) Validate() error {
	return m.ValidateE("")
}

func (m *StepTraceResponse) ValidateE(prefix string) error {
	errs := make([]error, 0)
	return is.MargeErr(errs...)
}
