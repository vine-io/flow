package bpmn

import (
	"container/list"

	"github.com/tidwall/btree"
)

var _ Element = (*Process)(nil)

type Process struct {
	Id         string
	Name       string
	Executable bool

	ExtensionElement *ExtensionElement
	Elements         *btree.Map[string, Element]
	ObjectReferences *btree.Map[string, *DataReference]
	StoreReferences  *btree.Map[string, *DataReference]
	start            Element
	end              Element
}

type DataReference struct {
	Id  string
	Ref string
}

func NewProcess(name string) *Process {
	process := &Process{
		Id:               "Process_" + randName(),
		Name:             name,
		Executable:       true,
		ExtensionElement: &ExtensionElement{},
		Elements:         &btree.Map[string, Element]{},
		ObjectReferences: &btree.Map[string, *DataReference]{},
		StoreReferences:  &btree.Map[string, *DataReference]{},
	}

	return process
}

func (p *Process) GetShape() Shape {
	return ProcessShape
}

func (p *Process) GetID() string {
	return p.Id
}

func (p *Process) SetID(id string) {
	p.Id = id
}

func (p *Process) GetName() string {
	return p.Name
}

func (p *Process) SetName(name string) {
	p.Name = name
}

func (p *Process) GetIncoming() []string {
	return nil
}

func (p *Process) SetIncoming(incoming []string) {
	return
}

func (p *Process) GetOutgoing() []string {
	return nil
}

func (p *Process) SetOutgoing(outgoing []string) {
	return
}

func (p *Process) GetExtension() *ExtensionElement {
	return p.ExtensionElement
}

func (p *Process) SetExtension(elem *ExtensionElement) {
	p.ExtensionElement = elem
}

func (p *Process) Link(src, dst string, cond *ConditionExpression) *Process {
	source, ok := p.Elements.GetMut(src)
	if !ok {
		return p
	}

	target, ok := p.Elements.GetMut(dst)
	if !ok {
		return p
	}

	fid := "Flow_" + randName()
	flow := &SequenceFlow{
		Id:        fid,
		SourceRef: source.GetID(),
		TargetRef: target.GetID(),
		Condition: cond,
	}
	p.Elements.Set(fid, flow)
	setOutgoing(source, fid)
	setIncoming(target, fid)

	return p
}

func (p *Process) AppendElem(elem Element) *Process {
	p.Elements.Set(elem.GetID(), elem)

	if IsStartEvent(elem) {
		p.start = elem
	}
	if IsEndEvent(elem) {
		p.end = elem
	}

	return p
}

// AfterInsertElem inserts a new element to Process after the specified Element
func (p *Process) AfterInsertElem(src string, dst Element, cond *ConditionExpression) *Process {
	source, ok := p.Elements.GetMut(src)
	if !ok {
		source = p.start
	}
	outs := source.GetOutgoing()
	if !IsGateway(source) && len(outs) > 0 {
		outgoing := outs[0]
		flow, _ := p.Elements.GetMut(outgoing)
		flow.SetIncoming([]string{dst.GetID()})
		dst.SetOutgoing([]string{flow.GetID()})
	}

	p.AppendElem(dst)
	p.Link(src, dst.GetID(), cond)

	return p
}

// BeforeInsertElem inserts a new element to Process before the specified Element
func (p *Process) BeforeInsertElem(dst string, src Element, cond *ConditionExpression) *Process {
	target, ok := p.Elements.GetMut(dst)
	if !ok {
		return p
	}
	ints := target.GetIncoming()
	if !IsGateway(target) && len(ints) > 0 {
		incoming := ints[0]
		flow, _ := p.Elements.GetMut(incoming)
		flow.SetOutgoing([]string{target.GetID()})
		src.SetIncoming([]string{flow.GetID()})
	}

	p.AppendElem(src)
	p.Link(src.GetID(), dst, cond)

	return p
}

type shapeCoord struct {
	x     int64
	y     int64
	count int
}

func (p *Process) Draw() *DiagramPlane {
	plane := &DiagramPlane{
		Id:      p.Id + "_di",
		Element: p.Id,
	}
	if p.start == nil {
		return plane
	}

	shapes := make(map[string]*DiagramShape)
	flows := make([]*SequenceFlow, 0)
	coordMap := make(map[string]*shapeCoord)

	coordX := int64(150)      // x 轴间隔
	coordY := int64(200)      // y 轴间隔
	startCoordX := int64(150) // x 轴初始位置
	startCoordY := int64(200) // y 轴初始位置

	queue := list.New()
	queue.PushBack(p.start)
	coordMap[p.start.GetID()] = &shapeCoord{ // 记录每个元素中点坐标
		x: startCoordX,
		y: startCoordY,
	}

	calConrd := func(width, height, x, y int64) (int64, int64) {
		return x - width/2, y - height/2
	}

	// dfs
	for queue.Len() > 0 {

		inner := list.New()
		for item := queue.Front(); item != nil; item = item.Next() {
			elem := item.Value.(Element)
			id := elem.GetID()

			coord := coordMap[id]
			x, y := coord.x, coord.y
			width, height := elem.GetShape().Size()
			startX, startY := calConrd(int64(width), int64(height), x, y)

			ds := &DiagramShape{
				Id:      id + "_di",
				Element: id,
				Bounds: &DiagramBounds{
					X:      startX,
					Y:      startY,
					Width:  int64(width),
					Height: int64(height),
				},
			}
			shapes[id] = ds

			for i, outgoing := range elem.GetOutgoing() {
				flow, ok := p.Elements.Get(outgoing)
				if !ok {
					continue
				}
				dst := flow.GetOutgoing()[0]
				target, ok := p.Elements.Get(dst)
				if ok {
					flows = append(flows, flow.(*SequenceFlow))

					inner.PushBack(target)
					sc, ok1 := coordMap[dst]
					if ok1 {
						sc.count += 1
						if x+coordX > sc.x {
							sc.x = x + coordX
						}
					} else {
						sc = &shapeCoord{
							x:     x + coordX,
							y:     y + coordY*int64(i),
							count: 0,
						}
					}

					coordMap[dst] = sc
				}
			}

		}

		queue = inner
	}

	plane.Shapes = make([]*DiagramShape, 0)
	for _, item := range shapes {
		plane.Shapes = append(plane.Shapes, item)
	}
	plane.Edges = make([]*DiagramEdge, 0)
	for _, flow := range flows {
		edge := &DiagramEdge{
			Id:      flow.Id + "_di",
			Element: flow.Id,
		}
		waypoints := make([]*DiagramWaypoint, 0)
		source := shapes[flow.SourceRef]
		bounds := source.Bounds
		wps := &DiagramWaypoint{
			X: bounds.X + bounds.Width,
			Y: bounds.Y + bounds.Height/2,
		}
		waypoints = append(waypoints, wps)
		target := shapes[flow.TargetRef]
		bounds = target.Bounds
		wpt := &DiagramWaypoint{
			X: bounds.X,
			Y: bounds.Y + bounds.Height/2,
		}
		if wpt.Y != wps.Y {
			x := wpt.X + (wpt.X-wps.X)/2
			waypoints = append(waypoints,
				&DiagramWaypoint{
					X: x,
					Y: wps.Y,
				},
				&DiagramWaypoint{
					X: x,
					Y: wpt.Y,
				},
			)
		}
		waypoints = append(waypoints, wpt)
		edge.Waypoints = waypoints
		plane.Edges = append(plane.Edges, edge)
	}

	return plane
}
