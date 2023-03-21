package flow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vine-io/flow/api"
)

func testNewPipe(t *testing.T) *ClientPipe {
	id := "1"
	peer := &Peer{
		Server: "localhost",
		Client: "localhost",
	}

	s := NewMemberPipeStream(context.TODO(), id, func(in *api.PipeResponse) (*api.PipeRequest, error) {
		out := &api.PipeRequest{
			Id:       id,
			Topic:    in.Topic,
			Revision: in.Revision,
		}
		switch in.Topic {
		case api.Topic_T_CALL:
			data := in.Call
			out.Call = &api.PipeCallResponse{
				Name: data.Name,
				Data: data.Data,
			}
		case api.Topic_T_STEP:
			data := in.Step
			out.Step = &api.PipeStepResponse{
				Name: data.Name,
				Data: []byte("test"),
			}
		}
		return out, nil
	})

	p := NewPipe(id, peer, s)
	go p.Start()

	return p
}

func TestNewPipe(t *testing.T) {
	testNewPipe(t)
}

func TestPipeClose(t *testing.T) {
	p := testNewPipe(t)
	p.Close()
}

func TestPipeCall(t *testing.T) {
	p := testNewPipe(t)
	defer p.Close()

	ctx := context.TODO()
	chunk := &api.PipeCallRequest{
		Name: "test",
		Data: []byte("hello"),
	}

	bch, ech := p.Call(NewCall(ctx, chunk))
	select {
	case e := <-ech:
		t.Fatal(e)
	case b := <-bch:
		assert.Equal(t, string(b), "hello", "they should be equal")
	}
}

func TestPipeStep(t *testing.T) {
	p := testNewPipe(t)
	defer p.Close()

	ctx := context.TODO()
	chunk := &api.PipeStepRequest{
		Wid:    "",
		Name:   "test",
		Action: 0,
		Items:  nil,
		Entity: nil,
	}

	bch, ech := p.Step(NewStep(ctx, chunk))
	select {
	case e := <-ech:
		t.Fatal(e)
	case b := <-bch:
		assert.Equal(t, string(b), "test", "they should be equal")
	}
}
