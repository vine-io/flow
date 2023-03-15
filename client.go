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

	"github.com/vine-io/flow/api"
	"github.com/vine-io/vine/core/client"
)

var _ api.FlowRpcService = (*Client)(nil)

type Client struct {
	conn api.FlowRpcClient
}

func (c *Client) NewWorkFlow() {}

func (c *Client) Register(ctx context.Context, in *api.RegisterRequest, opts ...client.CallOption) (*api.RegisterResponse, error) {
	return nil, nil
}

func (c *Client) Call(ctx context.Context, in *api.CallRequest, opts ...client.CallOption) (*api.CallResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Step(ctx context.Context, in *api.StepRequest, opts ...client.CallOption) (*api.StepResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Pipe(ctx context.Context, opts ...client.CallOption) (api.FlowRpc_PipeService, error) {
	//TODO implement me
	panic("implement me")

}
