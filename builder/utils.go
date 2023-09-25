// Copyright 2023 Lack (xingyys@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	"container/list"
	"fmt"

	json "github.com/json-iterator/go"
	"github.com/olive-io/bpmn/schema"
	"github.com/tidwall/btree"
	"github.com/vine-io/pkg/xname"
)

type shapeCoord struct {
	x     schema.Double
	y     schema.Double
	count int
}

func draw(
	startEvent schema.FlowElementInterface,
	elements *btree.Map[string, schema.FlowElementInterface],
	startCoordX, startCoordY schema.Double,
) (
	coordMap map[string]*shapeCoord,
	bpmnShapes []schema.BPMNShape,
	bpmnEdges []schema.BPMNEdge,
) {

	shapes := make(map[string]*schema.BPMNShape)
	flows := make([]*schema.SequenceFlow, 0)
	coordMap = make(map[string]*shapeCoord)

	coordX := schema.Double(150) // x 轴间隔
	coordY := schema.Double(200) // y 轴间隔

	queue := list.New()
	queue.PushBack(startEvent)
	startId, _ := startEvent.Id()
	coordMap[*startId] = &shapeCoord{ // 记录每个元素中点坐标
		x: startCoordX,
		y: startCoordY,
	}

	calConrd := func(width, height, x, y schema.Double) (schema.Double, schema.Double) {
		return x - width/2, y - height/2
	}

	bpmnShapes = make([]schema.BPMNShape, 0)
	bpmnEdges = make([]schema.BPMNEdge, 0)

	// dfs
	for queue.Len() > 0 {

		inner := list.New()
		for item := queue.Front(); item != nil; item = item.Next() {
			elem := item.Value.(schema.FlowElementInterface)
			eid, _ := elem.Id()
			flow, _ := elem.(schema.FlowNodeInterface)

			coord := coordMap[*eid]
			x, y := coord.x, coord.y
			switch tt := flow.(type) {
			case *SubProcessBuilder:
				subShapes, subEdges := tt.Draw(x, y)
				bpmnShapes = append(bpmnShapes, subShapes...)
				bpmnEdges = append(bpmnEdges, subEdges...)
				subShape := bpmnShapes[0]
				shapes[*eid] = &subShape
				x = x + subShape.BoundsField.WidthField/2 + coordX
			default:
				width, height := getFlowSize(elem)
				startX, startY := calConrd(width, height, x, y)

				ds := &schema.BPMNShape{}
				ds.SetId(schema.NewStringP(*eid + "_di"))
				dsElem := schema.QName(*eid)
				ds.SetBpmnElement(&dsElem)
				ds.SetBounds(&schema.Bounds{
					XField:      startX,
					YField:      startY,
					WidthField:  width,
					HeightField: height,
				})
				shapes[*eid] = ds
			}

			for i, outgoing := range *flow.Outgoings() {
				selem, ok := elements.Get(string(outgoing))
				if !ok {
					continue
				}
				dst := selem.(*schema.SequenceFlow).TargetRefField
				target, ok := elements.Get(dst)
				if ok {
					flows = append(flows, selem.(*schema.SequenceFlow))

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
							y:     y + coordY*schema.Double(i),
							count: 0,
						}
					}

					coordMap[dst] = sc
				}
			}

		}

		queue = inner
	}

	for _, item := range shapes {
		bpmnShapes = append(bpmnShapes, *item)
	}
	for _, flow := range flows {
		flowId, _ := flow.Id()
		edge := schema.BPMNEdge{}
		fid := string(*flowId) + "_di"
		edge.SetId(&fid)
		edge.SetBpmnElement((*schema.QName)(flowId))
		waypoints := make([]schema.Point, 0)

		sourceRef := flow.SourceRef()
		source := shapes[*sourceRef]
		bounds := source.Bounds()
		wps := schema.Point{
			XField: bounds.X() + bounds.Width(),
			YField: bounds.Y() + bounds.Height()/2,
		}
		waypoints = append(waypoints, wps)
		targetRef := flow.TargetRef()
		target := shapes[*targetRef]
		bounds = target.Bounds()
		wpt := schema.Point{
			XField: bounds.X(),
			YField: bounds.Y() + bounds.Height()/2,
		}
		if wpt.Y() != wps.Y() {
			x := wpt.X() + (wpt.X()-wps.X())/2
			waypoints = append(waypoints,
				schema.Point{
					XField: x,
					YField: wps.Y(),
				},
				schema.Point{
					XField: x,
					YField: wpt.Y(),
				},
			)
		}
		waypoints = append(waypoints, wpt)
		edge.SetWaypoints(waypoints)
		bpmnEdges = append(bpmnEdges, edge)
	}

	return
}

func randName() string {
	return xname.Gen(xname.C(7), xname.Lowercase(), xname.Digit())
}

func randShapeName(elem schema.FlowNodeInterface) *string {
	prefix := ""
	switch elem.(type) {
	case schema.ProcessInterface:
		prefix = "SubProcess"
	case schema.SubProcessInterface:
		prefix = "subSubProcess"
	case schema.EventInterface:
		prefix = "Event"
	case schema.TaskInterface:
		prefix = "Activity"
	default:
		prefix = "Activity"
	}

	id := prefix + "_" + randName()
	return &id
}

func setIncoming(elem schema.FlowNodeInterface, incoming string) {
	if IsEndEvent(elem) {
		return
	}
	ints := elem.Incomings()
	if ints == nil {
		incomings := make([]schema.QName, 0)
		ints = &incomings
	}
	if IsGateway(elem) {
		*ints = append(*ints, schema.QName(incoming))
	} else {
		incomings := []schema.QName{schema.QName(incoming)}
		ints = &incomings
	}
	elem.SetIncomings(*ints)
}

func setOutgoing(elem schema.FlowNodeInterface, outgoing string) {
	if IsEndEvent(elem) {
		return
	}
	outs := elem.Outgoings()
	if outs == nil {
		outgoings := make([]schema.QName, 0)
		outs = &outgoings
	}
	if IsGateway(elem) {
		*outs = append(*outs, schema.QName(outgoing))
	} else {
		outgoings := []schema.QName{schema.QName(outgoing)}
		outs = &outgoings
	}
	elem.SetOutgoings(*outs)
}

func IsGateway(elem schema.FlowElementInterface) bool {
	switch elem.(type) {
	case schema.ExclusiveGatewayInterface, schema.InclusiveGatewayInterface, schema.ParallelGatewayInterface:
		return true
	default:
		return false
	}
}

func IsStartEvent(elem schema.FlowElementInterface) bool {
	switch elem.(type) {
	case schema.StartEventInterface:
		return true
	default:
		return false
	}
}

func IsEndEvent(elem schema.FlowElementInterface) bool {
	switch elem.(type) {
	case schema.EndEventInterface:
		return true
	default:
		return false
	}
}

func getFlowSize(s schema.FlowElementInterface) (schema.Double, schema.Double) {
	switch s.(type) {
	case schema.SubProcessInterface:
		return 350, 200
	case schema.TaskInterface:
		return 100, 80
	case schema.StartEventInterface, schema.EndEventInterface, schema.CatchEventInterface:
		return 36, 36
	case schema.GatewayInterface:
		return 50, 50
	case schema.CollaborationInterface:
		return 850, 260
	case schema.DataObjectInterface, schema.DataStoreInterface:
		return 50, 50
	default:
		return 50, 50
	}
}

func parseItemValue(v any) (string, schema.ItemType) {
	switch tt := v.(type) {
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", tt), schema.ItemTypeInteger
	case float64, float32:
		return fmt.Sprintf("%f", tt), schema.ItemTypeFloat
	case string:
		return tt, schema.ItemTypeString
	case []byte:
		return string(tt), schema.ItemTypeString
	case *[]byte:
		return string(*tt), schema.ItemTypeString
	case bool:
		if tt {
			return "true", schema.ItemTypeBoolean
		}
		return "false", schema.ItemTypeBoolean
	default:
		data, _ := json.Marshal(v)
		return string(data), schema.ItemTypeObject
	}
}
