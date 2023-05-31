package bpmn

var _ Element = (*Gateway)(nil)

type Gateway struct {
	Id               string
	Name             string
	Incoming         []string
	Outgoing         []string
	ExtensionElement *ExtensionElement
}

func (g *Gateway) GetShape() Shape {
	return GatewayShape
}

func (g *Gateway) GetID() string {
	return g.Id
}

func (g *Gateway) SetID(id string) {
	g.Id = id
}

func (g *Gateway) GetName() string {
	return g.Name
}

func (g *Gateway) SetName(name string) {
	g.Name = name
}

func (g *Gateway) GetIncoming() []string {
	return g.Incoming
}

func (g *Gateway) SetIncoming(incoming []string) {
	g.Incoming = incoming
}

func (g *Gateway) GetOutgoing() []string {
	return g.Outgoing
}

func (g *Gateway) SetOutgoing(outgoing []string) {
	g.Outgoing = outgoing
}

func (g *Gateway) GetExtension() *ExtensionElement {
	return g.ExtensionElement
}

func (g *Gateway) SetExtension(elem *ExtensionElement) {
	g.ExtensionElement = elem
}

type ExclusiveGateway struct {
	Gateway
}

func (g *ExclusiveGateway) GetShape() Shape {
	return ExclusiveGatewayShape
}

type InclusiveGateway struct {
	Gateway
}

func (g *InclusiveGateway) GetShape() Shape {
	return InclusiveGatewayShape
}

type ParallelGateway struct {
	Gateway
}

func (g *ParallelGateway) GetShape() Shape {
	return ParallelGatewayShape
}
