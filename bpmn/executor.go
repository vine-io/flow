package bpmn

import (
	"container/list"
	"context"
)

type ExecuteCtx struct {
	context.Context
	vars    map[string]any
	objects map[string]byte
}

type Executor interface {
	Execute(ctx *ExecuteCtx) error
}

type Runner struct {
	*Definition
	queue *list.List
}

func NewRunner(d *Definition) *Runner {
	queue := list.New()
	queue.PushBack(d.start)
	runner := &Runner{
		Definition: d,
		queue:      queue,
	}
	return runner
}

func (r *Runner) Run(ctx context.Context) error {

	bctx := &ExecuteCtx{
		Context: ctx,
		vars:    nil,
		objects: nil,
	}

	m := r.next(bctx)
	for m != nil {
		select {
		case <-ctx.Done():
		default:
		}

		_ = r.execute(bctx, m)
		m = r.next(bctx)
	}

	return nil
}

func (r *Runner) execute(ctx *ExecuteCtx, m Model) error {
	executor, ok := m.(Executor)
	if !ok {
		return nil
	}

	if err := executor.Execute(ctx); err != nil {
		return err
	}

	return nil
}

func (r *Runner) next(ctx *ExecuteCtx) Model {
	if r.queue.Len() == 0 {
		return nil
	}

	elem := r.queue.Remove(r.queue.Front())
	if elem == nil {
		return nil
	}

	point := elem.(Model)
	if v, ok := point.(ProcessIO); ok {
		outgoing := v.GetOut()
		if outgoing == "" {
			return point
		}
		flow, ok := r.flows[outgoing]
		if !ok {
			return point
		}

		m, ok := r.models[flow.Out]
		if ok {
			r.queue.PushBack(m)
		}
	}

	if v, ok := point.(MultipartIO); ok {
		outgoings := v.GetOuts()
		flows := make([]*SequenceFlow, 0)
		for _, outgoing := range outgoings {
			flow, ok := r.flows[outgoing]
			if ok {
				flows = append(flows, flow)
			}
		}
		flows = v.SelectOutgoing(ctx, flows)
		for _, flow := range flows {
			m, ok := r.models[flow.Out]
			if ok {
				r.queue.PushBack(m)
			}
		}
	}

	return point
}
