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

func (g *GatewayImpl) ReadExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}

func (g *GatewayImpl) WriteExtensionElement() (ExtensionElementWriter, error) {
	//TODO implement me
	panic("implement me")
}

func (g *GatewayImpl) GetIns() []string { return g.Incoming }

func (g *GatewayImpl) SetIns(in []string) { g.Incoming = in }

func (g *GatewayImpl) GetOuts() []string { return g.Outgoing }

func (g *GatewayImpl) SetOuts(out []string) { g.Outgoing = out }

type ExclusiveGateway struct {
	GatewayImpl
}

type ParallelGateway struct {
	GatewayImpl
}
