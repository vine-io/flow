package bpmn

type Gateway interface {
	Model
	ModelExtension
	MultipartIO
}

var _ Gateway = (*GatewayImpl)(nil)

type GatewayImpl struct {
	ModelMeta
	Incoming []string
	Outgoing []string
}

func (g *GatewayImpl) GetShape() Shape { return GatewayShape }

func (g *GatewayImpl) ReadExtensionElement() (*ExtensionElement, error) {
	return &ExtensionElement{}, nil
}

func (g *GatewayImpl) WriteExtensionElement(elem *ExtensionElement) error {
	return nil
}

func (g *GatewayImpl) GetIns() []string { return g.Incoming }

func (g *GatewayImpl) SetIns(in []string) { g.Incoming = in }

func (g *GatewayImpl) GetOuts() []string { return g.Outgoing }

func (g *GatewayImpl) SetOuts(out []string) { g.Outgoing = out }

func (g *GatewayImpl) Execute(ctx *ExecuteCtx) ([]string, error) {
	view := ctx.view
	if len(g.Outgoing) == 0 {
		return nil, nil
	}

	outs := make([]string, 0)
	for _, outgoing := range g.Outgoing {
		flow, ok := view.flows[outgoing]
		if !ok {
			continue
		}

		m, ok := view.models[flow.Out]
		if !ok {
			continue
		}

		outs = append(outs, m.GetID())
	}

	return outs, nil
}

type ExclusiveGateway struct {
	GatewayImpl
}

type InclusiveGateway struct {
	GatewayImpl
}

type ParallelGateway struct {
	GatewayImpl
}
