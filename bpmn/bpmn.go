package bpmn

type Shape int32

const (
	EventShape Shape = iota + 1
	TaskShape
	GatewayShape
	DataObjectShape
	DataStoreShape
	ProcessShape
)

type Model interface {
	GetShape() Shape
	GetID() string
	SetID(string)
	GetName() string
	SetName(string)
	GetDocument() string
	SetDocument(string)
}

type ExtensionElementReader interface {
	Read() ([]byte, error)
	Close() error
}

type ExtensionElementWriter interface {
	Write([]byte) error
}

type ModelExtension interface {
	ReadExtensionElement() (ExtensionElementWriter, error)
	WriteExtensionElement() (ExtensionElementWriter, error)
}

var _ Model = (*ModelMeta)(nil)

type ModelMeta struct {
	Id       string
	Name     string
	Document string
}

func (m *ModelMeta) GetShape() Shape {
	return 0
}

func (m *ModelMeta) GetID() string {
	return m.Id
}

func (m *ModelMeta) SetID(id string) {
	m.Id = id
}

func (m *ModelMeta) GetName() string {
	return m.Name
}

func (m *ModelMeta) SetName(name string) {
	m.Name = name
}

func (m *ModelMeta) GetDocument() string {
	return m.Document
}

func (m *ModelMeta) SetDocument(document string) {
	m.Document = document
}

type ProcessIO interface {
	GetIn() string
	SetIn(string)
	GetOut() string
	SetOut(string)
}

type MultipartIO interface {
	GetIns() []string
	SetIns([]string)
	GetOuts() []string
	SetOuts([]string)
	SelectOutgoing(ctx *ExecuteCtx, flows []*SequenceFlow) []*SequenceFlow
}

func SetModelIn(m Model, in string) {
	if v1, match := m.(ProcessIO); match {
		v1.SetIn(in)
		return
	}

	if v2, match := m.(MultipartIO); match {
		ints := v2.GetIns()
		if ints == nil {
			ints = []string{}
		}
		ints = append(ints, in)
		v2.SetIns(ints)
		return
	}
}

func SetModelOut(m Model, out string) {
	if v1, match := m.(ProcessIO); match {
		v1.SetOut(out)
		return
	}

	if v2, match := m.(MultipartIO); match {
		outs := v2.GetOuts()
		if outs == nil {
			outs = []string{}
		}
		outs = append(outs, out)
		v2.SetOuts(outs)
		return
	}
}

type Diagram struct {
	Root   *Graph
	graphs map[string]*Graph
	edges  map[string]*Edge
}

type Graph struct {
	shape    Shape
	Ref      string
	Incoming []*Edge
	Outgoing []*Edge
	x        int32
	y        int32
}

type Edge struct {
	From *Graph
	To   *Graph
	Ref  string
}

//
//func BuildDiagram(p *Process) *Diagram {
//	dg := &Diagram{
//		Root:   &Graph{},
//		graphs: map[string]*Graph{},
//		edges:  map[string]*Edge{},
//	}
//
//	traverseDiagram(dg, dg.Root, p, p.start, 0, 0)
//	return dg
//}
//
//func traverseDiagram(dg *Diagram, g *Graph, p *Process, m Model, x, y int32) {
//	if m == nil {
//		return
//	}
//
//	if g == nil {
//		g = &Graph{}
//	}
//	g.shape = m.GetShape()
//	g.Ref = m.GetID()
//
//	if _, ok := dg.graphs[g.Ref]; ok {
//		g.x = x
//	} else {
//		dg.graphs[g.Ref] = g
//		g.x = x
//		g.y = y
//	}
//
//	if v, ok := m.(ProcessIO); ok {
//		outgoing := v.GetOut()
//		if outgoing != "" {
//			g.Outgoing = []*Edge{}
//			sf, ok := p.flows[outgoing]
//			if !ok {
//				return
//			}
//
//			edge := &Edge{
//				From: g,
//				Ref:  sf.GetID(),
//			}
//			dg.edges[edge.Ref] = edge
//			g.Outgoing = append(g.Outgoing, edge)
//
//			next, ok := p.models[sf.Out]
//			if ok {
//				gnext := dg.graphs[next.GetID()]
//				if gnext == nil {
//					gnext = &Graph{
//						Incoming: []*Edge{edge},
//					}
//				} else {
//					gnext.Incoming = append(gnext.Incoming, edge)
//				}
//				edge.To = gnext
//				traverseDiagram(dg, gnext, p, next, x+1, y)
//			}
//		}
//	}
//
//	if v, ok := m.(MultipartIO); ok {
//		outgoings := v.GetOuts()
//		if len(outgoings) > 0 {
//			g.Outgoing = make([]*Edge, 0)
//		}
//		for idx, outgoing := range outgoings {
//			sf, ok := p.flows[outgoing]
//			if !ok {
//				continue
//			}
//
//			edge := &Edge{
//				From: g,
//				Ref:  sf.GetID(),
//			}
//			dg.edges[edge.Ref] = edge
//			g.Outgoing = append(g.Outgoing, edge)
//
//			next, ok := p.models[sf.Out]
//			if ok {
//				gnext := dg.graphs[next.GetID()]
//				if gnext == nil {
//					gnext = &Graph{
//						Incoming: []*Edge{edge},
//					}
//				} else {
//					gnext.Incoming = append(gnext.Incoming, edge)
//				}
//				edge.To = gnext
//				gnext.Incoming = []*Edge{edge}
//				traverseDiagram(dg, gnext, p, next, x+1, y+int32(idx))
//			}
//		}
//	}
//}
