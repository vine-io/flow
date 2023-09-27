// Code generated by proto-gen-deepcopy. DO NOT EDIT.
// source: github.com/vine-io/flow/api/flow.proto

package api

import ()

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *Error) DeepCopyInto(out *Error) {
	*out = *in
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new Error.
func (in *Error) DeepCopy() *Error {
	if in == nil {
		return nil
	}
	out := new(Error)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *OwnerReference) DeepCopyInto(out *OwnerReference) {
	*out = *in
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new OwnerReference.
func (in *OwnerReference) DeepCopy() *OwnerReference {
	if in == nil {
		return nil
	}
	out := new(OwnerReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *Worker) DeepCopyInto(out *Worker) {
	*out = *in
	if in.Attrs != nil {
		in, out := &in.Attrs, &out.Attrs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new Worker.
func (in *Worker) DeepCopy() *Worker {
	if in == nil {
		return nil
	}
	out := new(Worker)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *Entity) DeepCopyInto(out *Entity) {
	*out = *in
	if in.OwnerReferences != nil {
		in, out := &in.OwnerReferences, &out.OwnerReferences
		*out = make([]*OwnerReference, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(OwnerReference)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.Workers != nil {
		in, out := &in.Workers, &out.Workers
		*out = make(map[string]*Worker, len(*in))
		for key, val := range *in {
			var outVal *Worker
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(Worker)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new Entity.
func (in *Entity) DeepCopy() *Entity {
	if in == nil {
		return nil
	}
	out := new(Entity)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *Echo) DeepCopyInto(out *Echo) {
	*out = *in
	if in.Workers != nil {
		in, out := &in.Workers, &out.Workers
		*out = make(map[string]*Worker, len(*in))
		for key, val := range *in {
			var outVal *Worker
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(Worker)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new Echo.
func (in *Echo) DeepCopy() *Echo {
	if in == nil {
		return nil
	}
	out := new(Echo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *Step) DeepCopyInto(out *Step) {
	*out = *in
	if in.Injects != nil {
		in, out := &in.Injects, &out.Injects
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Workers != nil {
		in, out := &in.Workers, &out.Workers
		*out = make(map[string]*Worker, len(*in))
		for key, val := range *in {
			var outVal *Worker
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(Worker)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new Step.
func (in *Step) DeepCopy() *Step {
	if in == nil {
		return nil
	}
	out := new(Step)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *Revision) DeepCopyInto(out *Revision) {
	*out = *in
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new Revision.
func (in *Revision) DeepCopy() *Revision {
	if in == nil {
		return nil
	}
	out := new(Revision)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *WorkflowOption) DeepCopyInto(out *WorkflowOption) {
	*out = *in
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new WorkflowOption.
func (in *WorkflowOption) DeepCopy() *WorkflowOption {
	if in == nil {
		return nil
	}
	out := new(WorkflowOption)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *WorkflowStatus) DeepCopyInto(out *WorkflowStatus) {
	*out = *in
	if in.Option != nil {
		in, out := &in.Option, &out.Option
		*out = new(WorkflowOption)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new WorkflowStatus.
func (in *WorkflowStatus) DeepCopy() *WorkflowStatus {
	if in == nil {
		return nil
	}
	out := new(WorkflowStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *WorkflowStep) DeepCopyInto(out *WorkflowStep) {
	*out = *in
	if in.Injects != nil {
		in, out := &in.Injects, &out.Injects
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Logs != nil {
		in, out := &in.Logs, &out.Logs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Stages != nil {
		in, out := &in.Stages, &out.Stages
		*out = make([]*WorkflowStepStage, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(WorkflowStepStage)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new WorkflowStep.
func (in *WorkflowStep) DeepCopy() *WorkflowStep {
	if in == nil {
		return nil
	}
	out := new(WorkflowStep)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *WorkflowStepStage) DeepCopyInto(out *WorkflowStepStage) {
	*out = *in
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *Workflow) DeepCopyInto(out *Workflow) {
	*out = *in
	if in.Option != nil {
		in, out := &in.Option, &out.Option
		*out = new(WorkflowOption)
		(*in).DeepCopyInto(*out)
	}
	if in.Entities != nil {
		in, out := &in.Entities, &out.Entities
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Steps != nil {
		in, out := &in.Steps, &out.Steps
		*out = make([]*WorkflowStep, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(WorkflowStep)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(WorkflowStatus)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new Workflow.
func (in *Workflow) DeepCopy() *Workflow {
	if in == nil {
		return nil
	}
	out := new(Workflow)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *WorkflowSnapshot) DeepCopyInto(out *WorkflowSnapshot) {
	*out = *in
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new WorkflowSnapshot.
func (in *WorkflowSnapshot) DeepCopy() *WorkflowSnapshot {
	if in == nil {
		return nil
	}
	out := new(WorkflowSnapshot)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *TraceLog) DeepCopyInto(out *TraceLog) {
	*out = *in
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *BpmnTrace) DeepCopyInto(out *BpmnTrace) {
	*out = *in
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *WorkflowWatchResult) DeepCopyInto(out *WorkflowWatchResult) {
	*out = *in
}

// DeepCopy is an auto-generated deepcopy function, copying the receiver, creating a new WorkflowWatchResult.
func (in *WorkflowWatchResult) DeepCopy() *WorkflowWatchResult {
	if in == nil {
		return nil
	}
	out := new(WorkflowWatchResult)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *Interactive) DeepCopyInto(out *Interactive) {
	*out = *in
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = make([]*Property, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(Property)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *Property) DeepCopyInto(out *Property) {
	*out = *in
}

// DeepCopyInto is an auto-generated deepcopy function, coping the receiver, writing into out. in must be no-nil.
func (in *WorkHookResult) DeepCopyInto(out *WorkHookResult) {
	*out = *in
	if in.Worker != nil {
		in, out := &in.Worker, &out.Worker
		*out = new(Worker)
		(*in).DeepCopyInto(*out)
	}
}
