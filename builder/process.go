package builder

import (
	"strings"

	"github.com/olive-io/bpmn/schema"
	"github.com/tidwall/btree"
)

// ProcessDefinitionsBuilder builds Process Definitions structure
type ProcessDefinitionsBuilder struct {
	d   *schema.Definitions
	ptr *ProcessBuilder
	cur string
	err error
}

func NewProcessDefinitionsBuilder(name string) *ProcessDefinitionsBuilder {
	def := schema.DefaultDefinitions()
	ptr := NewProcessBuilder(name)
	return &ProcessDefinitionsBuilder{d: &def, ptr: ptr}
}

func (b *ProcessDefinitionsBuilder) Id(id string) *ProcessDefinitionsBuilder {
	b.ptr.Id = id
	return b
}

func (b *ProcessDefinitionsBuilder) SetProperty(key, value string) *ProcessDefinitionsBuilder {
	property := &schema.Item{
		Name:  key,
		Value: value,
	}
	if b.ptr.ExtensionElement.PropertiesField == nil {
		b.ptr.ExtensionElement.PropertiesField = &schema.Properties{Property: make([]*schema.Item, 0)}
	}
	if b.ptr.ExtensionElement.TaskHeaderField == nil {
		b.ptr.ExtensionElement.TaskHeaderField = &schema.TaskHeader{Header: make([]*schema.Item, 0)}
	}
	if key != "action" && !strings.HasPrefix(key, "__step_mapping") {
		b.ptr.ExtensionElement.TaskHeaderField.Header = append(b.ptr.ExtensionElement.TaskHeaderField.Header, &schema.Item{Name: key})
	}
	b.ptr.ExtensionElement.PropertiesField.Property = append(b.ptr.ExtensionElement.PropertiesField.Property, property)
	return b
}

func (b *ProcessDefinitionsBuilder) PopProperty() map[string]string {
	properties := map[string]string{}
	if b.ptr == nil || b.ptr.ExtensionElement == nil ||
		b.ptr.ExtensionElement.PropertiesField == nil ||
		len(b.ptr.ExtensionElement.PropertiesField.Property) == 0 {
		return properties
	}

	for _, item := range b.ptr.ExtensionElement.PropertiesField.Property {
		properties[item.Name] = item.Value
	}
	b.ptr.ExtensionElement.PropertiesField.Property = []*schema.Item{}
	return properties
}

func (b *ProcessDefinitionsBuilder) Start() *ProcessDefinitionsBuilder {
	event := schema.DefaultStartEvent()
	id := randShapeName(&event)
	event.SetId(id)
	b.ptr.AppendElem(&event)
	b.cur = *id

	return b
}

func (b *ProcessDefinitionsBuilder) AppendElem(elem schema.FlowNodeInterface) *ProcessDefinitionsBuilder {
	id, found := elem.Id()
	if !found {
		id = randShapeName(elem)
		elem.SetId(id)
	}
	b.ptr.AfterInsertElem(b.cur, elem, nil)
	b.cur = *id
	return b
}

func (b *ProcessDefinitionsBuilder) End() *ProcessDefinitionsBuilder {
	event := schema.DefaultEndEvent()
	event.SetId(randShapeName(&event))
	b.AppendElem(&event)

	return b
}

func (b *ProcessDefinitionsBuilder) ToDefinitions() (*schema.Definitions, error) {
	if b.err != nil {
		return nil, b.err
	}

	proc := b.ptr.Out()
	shapes, edges := b.ptr.Draw(100, 150)
	plane := &schema.BPMNPlane{}
	pid := proc.IdField
	planeId := schema.QName(*pid)
	plane.SetBpmnElement(&planeId)
	plane.SetId(schema.NewStringP(*pid + "_di"))
	plane.BPMNShapeFields = shapes
	plane.BPMNEdgeFields = edges
	diagram := &schema.BPMNDiagram{}
	diagram.BPMNPlaneField = plane
	b.d.DiagramField = diagram
	b.d.SetProcesses([]schema.Process{*proc})

	return b.d, nil
}

type ProcessBuilder struct {
	Id   string
	Name string

	ExtensionElement *schema.ExtensionElements
	Elements         *btree.Map[string, schema.FlowElementInterface]
	Objects          *btree.Map[string, *schema.DataObject]
	StoreReferences  *btree.Map[string, *schema.DataStoreReference]
	start            schema.FlowElementInterface
	end              schema.FlowElementInterface
}

func NewProcessBuilder(name string) *ProcessBuilder {
	extensionElements := schema.DefaultExtensionElements()
	process := &ProcessBuilder{
		Id:               "Process_" + randName(),
		Name:             name,
		ExtensionElement: &extensionElements,
		Elements:         &btree.Map[string, schema.FlowElementInterface]{},
		Objects:          &btree.Map[string, *schema.DataObject]{},
		StoreReferences:  &btree.Map[string, *schema.DataStoreReference]{},
	}

	return process
}

func (p *ProcessBuilder) SetExtension(elem *schema.ExtensionElements) {
	p.ExtensionElement = elem
}

func (p *ProcessBuilder) Link(src, dst string, cond *schema.AnExpression) *ProcessBuilder {
	source, ok := p.Elements.GetMut(src)
	if !ok {
		return p
	}
	sourceRef, ok := source.(schema.FlowNodeInterface)
	if !ok {
		return p
	}

	target, ok := p.Elements.GetMut(dst)
	if !ok {
		return p
	}
	targetRef, ok := target.(schema.FlowNodeInterface)
	if !ok {
		return p
	}

	sourceRefId, _ := source.Id()
	targetRefId, _ := target.Id()

	fid := "Flow_" + randName()
	flow := &schema.SequenceFlow{
		SourceRefField:           *sourceRefId,
		TargetRefField:           *targetRefId,
		ConditionExpressionField: cond,
	}
	flow.SetId(&fid)
	p.Elements.Set(fid, flow)
	setOutgoing(sourceRef, fid)
	setIncoming(targetRef, fid)

	return p
}

func (p *ProcessBuilder) AppendElem(elem schema.FlowElementInterface) *ProcessBuilder {
	id, _ := elem.Id()

	p.Elements.Set(*id, elem)

	if IsStartEvent(elem) {
		p.start = elem
	}
	if IsEndEvent(elem) {
		p.end = elem
	}

	return p
}

// AfterInsertElem inserts a new element to Process after the specified Element
func (p *ProcessBuilder) AfterInsertElem(src string, dst schema.FlowNodeInterface, cond *schema.AnExpression) *ProcessBuilder {
	source, ok := p.Elements.GetMut(src)
	if !ok {
		source = p.start
	}
	sourceRef, ok := source.(schema.FlowNodeInterface)
	if !ok {
		return p
	}

	dstId, _ := dst.Id()

	outs := sourceRef.Outgoings()
	if !IsGateway(source) && len(*outs) > 0 {
		outgoing := (*outs)[0]
		elem, _ := p.Elements.GetMut(string(outgoing))
		flow := elem.(schema.FlowNodeInterface)
		incomings := make([]schema.QName, 0)
		incomings = append(incomings, schema.QName(*dstId))
		flow.SetIncomings(incomings)
		dst.SetOutgoings(incomings)
	}

	p.AppendElem(dst)
	p.Link(src, *dstId, cond)

	return p
}

// BeforeInsertElem inserts a new element to Process before the specified Element
func (p *ProcessBuilder) BeforeInsertElem(dst string, src schema.FlowNodeInterface, cond *schema.AnExpression) *ProcessBuilder {
	target, ok := p.Elements.GetMut(dst)
	if !ok {
		return p
	}
	targetRef, ok := target.(schema.FlowNodeInterface)
	if !ok {
		return p
	}

	srcId, _ := src.Id()

	ints := targetRef.Incomings()
	if !IsGateway(target) && len(*ints) > 0 {
		incoming := (*ints)[0]
		elem, _ := p.Elements.GetMut(string(incoming))
		outgoings := make([]schema.QName, 0)
		outgoings = append(outgoings, schema.QName(*srcId))
		flow := elem.(schema.FlowNodeInterface)
		flow.SetOutgoings(outgoings)
		src.SetIncomings(outgoings)
	}

	p.AppendElem(src)
	p.Link(*srcId, dst, cond)

	return p
}

func (p *ProcessBuilder) Out() *schema.Process {
	proc := schema.Process{}
	proc.SetId(&p.Id)
	proc.SetName(&p.Name)
	proc.ExtensionElementsField = p.ExtensionElement

	objects := make([]schema.DataObject, 0)
	p.Objects.Scan(func(key string, value *schema.DataObject) bool {
		objects = append(objects, *value)
		return true
	})
	proc.SetDataObjects(objects)

	storesReferences := make([]schema.DataStoreReference, 0)
	p.StoreReferences.Scan(func(key string, value *schema.DataStoreReference) bool {
		storesReferences = append(storesReferences, *value)
		return true
	})
	proc.SetDataStoreReferences(storesReferences)

	p.Elements.Scan(func(key string, value schema.FlowElementInterface) bool {
		switch elem := value.(type) {
		case *schema.StartEvent:
			if proc.StartEventField == nil {
				proc.StartEventField = make([]schema.StartEvent, 0)
			}
			proc.StartEventField = append(proc.StartEventField, *elem)
		case *schema.EndEvent:
			if proc.EndEventField == nil {
				proc.EndEventField = make([]schema.EndEvent, 0)
			}
			proc.EndEventField = append(proc.EndEventField, *elem)
		case *schema.SubProcess:
			if proc.SubProcessField == nil {
				proc.SubProcessField = make([]schema.SubProcess, 0)
			}
			proc.SubProcessField = append(proc.SubProcessField, *elem)
		case *SubProcessBuilder:
			if proc.SubProcessField == nil {
				proc.SubProcessField = make([]schema.SubProcess, 0)
			}
			proc.SubProcessField = append(proc.SubProcessField, *elem.Out())
		case *schema.Task:
			if proc.TaskField == nil {
				proc.TaskField = make([]schema.Task, 0)
			}
			proc.TaskField = append(proc.TaskField, *elem)
		case *schema.ServiceTask:
			if proc.ServiceTaskField == nil {
				proc.ServiceTaskField = make([]schema.ServiceTask, 0)
			}
			proc.ServiceTaskField = append(proc.ServiceTaskField, *elem)
		case *schema.UserTask:
			if proc.UserTaskField == nil {
				proc.UserTaskField = make([]schema.UserTask, 0)
			}
			proc.UserTaskField = append(proc.UserTaskField, *elem)
		case *schema.SequenceFlow:
			if proc.SequenceFlowField == nil {
				proc.SequenceFlowField = make([]schema.SequenceFlow, 0)
			}
			proc.SequenceFlowField = append(proc.SequenceFlowField, *elem)
		}

		return true
	})

	return &proc
}

func (p *ProcessBuilder) Draw(startCoordX, startCoordY schema.Double) ([]schema.BPMNShape, []schema.BPMNEdge) {

	_, bpmnShapes, bpmnEdges := draw(p.start, p.Elements, startCoordX, startCoordY)

	return bpmnShapes, bpmnEdges
}

// SubProcessDefinitionsBuilder builds SubProcess Definitions structure
type SubProcessDefinitionsBuilder struct {
	d   *schema.Definitions
	ptr *SubProcessBuilder
	cur string
}

func NewSubProcessDefinitionsBuilder(name string) *SubProcessDefinitionsBuilder {
	def := schema.DefaultDefinitions()
	ptr := NewSubProcessBuilder(name)
	return &SubProcessDefinitionsBuilder{d: &def, ptr: ptr}
}

func (b *SubProcessDefinitionsBuilder) Id(id string) *SubProcessDefinitionsBuilder {
	b.ptr.id = id
	b.ptr.SetId(&id)
	return b
}

func (b *SubProcessDefinitionsBuilder) SetProperty(key, value string) *SubProcessDefinitionsBuilder {
	property := &schema.Item{
		Name:  key,
		Value: value,
	}
	if b.ptr.ExtensionElement.PropertiesField == nil {
		b.ptr.ExtensionElement.PropertiesField = &schema.Properties{Property: make([]*schema.Item, 0)}
	}
	if b.ptr.ExtensionElement.TaskHeaderField == nil {
		b.ptr.ExtensionElement.TaskHeaderField = &schema.TaskHeader{Header: make([]*schema.Item, 0)}
	}
	if key != "action" && !strings.HasPrefix(key, "__step_mapping") {
		b.ptr.ExtensionElement.TaskHeaderField.Header = append(b.ptr.ExtensionElement.TaskHeaderField.Header, &schema.Item{Name: key})
	}
	b.ptr.ExtensionElement.PropertiesField.Property = append(b.ptr.ExtensionElement.PropertiesField.Property, property)
	return b
}

func (b *SubProcessDefinitionsBuilder) PopProperty() map[string]string {
	properties := map[string]string{}
	if b.ptr == nil || b.ptr.ExtensionElement == nil ||
		b.ptr.ExtensionElement.PropertiesField == nil ||
		len(b.ptr.ExtensionElement.PropertiesField.Property) == 0 {
		return properties
	}

	for _, item := range b.ptr.ExtensionElement.PropertiesField.Property {
		properties[item.Name] = item.Value
	}
	return properties
}

func (b *SubProcessDefinitionsBuilder) Start() *SubProcessDefinitionsBuilder {
	event := schema.DefaultStartEvent()
	id := randShapeName(&event)
	event.SetId(id)
	b.ptr.AppendElem(&event)
	b.cur = *id

	return b
}

func (b *SubProcessDefinitionsBuilder) AppendElem(elem schema.FlowNodeInterface) *SubProcessDefinitionsBuilder {
	id, found := elem.Id()
	if !found {
		id = randShapeName(elem)
		elem.SetId(id)
	}
	b.ptr.AfterInsertElem(b.cur, elem, nil)
	b.cur = *id
	return b
}

func (b *SubProcessDefinitionsBuilder) End() *SubProcessDefinitionsBuilder {
	event := schema.DefaultEndEvent()
	event.SetId(randShapeName(&event))
	b.AppendElem(&event)

	return b
}

func (b *SubProcessDefinitionsBuilder) ToDefinitions() *schema.Definitions {
	proc := schema.Process{}
	pid := "Process_" + randName()
	proc.SetId(schema.NewStringP(pid))
	proc.SetIsExecutable(schema.NewBoolP(true))
	proc.SetSubProcesses([]schema.SubProcess{*b.ptr.Out()})
	shapes, edges := b.ptr.Draw(100, 150)
	plane := &schema.BPMNPlane{}
	planeId := schema.QName(pid)
	plane.SetBpmnElement(&planeId)
	plane.SetId(schema.NewStringP(pid + "_di"))
	plane.BPMNShapeFields = shapes
	plane.BPMNEdgeFields = edges
	diagram := &schema.BPMNDiagram{}
	diagram.BPMNPlaneField = plane
	b.d.DiagramField = diagram
	b.d.SetProcesses([]schema.Process{proc})

	return b.d
}

func (b *SubProcessDefinitionsBuilder) Out() *SubProcessBuilder {
	return b.ptr
}

func (b *SubProcessDefinitionsBuilder) ToSubProcess() (*schema.SubProcess, []schema.BPMNShape, []schema.BPMNEdge) {
	process := b.ptr.Out()
	shapes, edges := b.ptr.Draw(100, 150)

	return process, shapes, edges
}

type SubProcessBuilder struct {
	*schema.SubProcess

	id   string
	name string

	ExtensionElement *schema.ExtensionElements
	Elements         *btree.Map[string, schema.FlowElementInterface]
	Objects          *btree.Map[string, *schema.DataObject]
	StoreReferences  *btree.Map[string, *schema.DataStoreReference]
	start            schema.FlowElementInterface
	end              schema.FlowElementInterface
}

func NewSubProcessBuilder(name string) *SubProcessBuilder {
	extensionElements := schema.DefaultExtensionElements()
	proc := schema.SubProcess{}
	id := "SubProcess_" + randName()
	proc.SetId(&id)
	proc.SetName(&name)
	process := &SubProcessBuilder{
		SubProcess:       &proc,
		id:               id,
		name:             name,
		ExtensionElement: &extensionElements,
		Elements:         &btree.Map[string, schema.FlowElementInterface]{},
		Objects:          &btree.Map[string, *schema.DataObject]{},
		StoreReferences:  &btree.Map[string, *schema.DataStoreReference]{},
	}

	return process
}

func (p *SubProcessBuilder) SetExtension(elem *schema.ExtensionElements) {
	p.ExtensionElement = elem
}

func (p *SubProcessBuilder) Link(src, dst string, cond *schema.AnExpression) *SubProcessBuilder {
	source, ok := p.Elements.GetMut(src)
	if !ok {
		return p
	}
	sourceRef, ok := source.(schema.FlowNodeInterface)
	if !ok {
		return p
	}

	target, ok := p.Elements.GetMut(dst)
	if !ok {
		return p
	}
	targetRef, ok := target.(schema.FlowNodeInterface)
	if !ok {
		return p
	}

	sourceRefId, _ := source.Id()
	targetRefId, _ := target.Id()

	fid := "Flow_" + randName()
	flow := &schema.SequenceFlow{
		SourceRefField:           *sourceRefId,
		TargetRefField:           *targetRefId,
		ConditionExpressionField: cond,
	}
	flow.SetId(&fid)
	p.Elements.Set(fid, flow)
	setOutgoing(sourceRef, fid)
	setIncoming(targetRef, fid)

	return p
}

func (p *SubProcessBuilder) AppendElem(elem schema.FlowElementInterface) *SubProcessBuilder {
	id, _ := elem.Id()

	p.Elements.Set(*id, elem)

	if IsStartEvent(elem) {
		p.start = elem
	}
	if IsEndEvent(elem) {
		p.end = elem
	}

	return p
}

// AfterInsertElem inserts a new element to SubProcess after the specified Element
func (p *SubProcessBuilder) AfterInsertElem(src string, dst schema.FlowNodeInterface, cond *schema.AnExpression) *SubProcessBuilder {
	source, ok := p.Elements.GetMut(src)
	if !ok {
		source = p.start
	}
	sourceRef, ok := source.(schema.FlowNodeInterface)
	if !ok {
		return p
	}

	dstId, _ := dst.Id()

	outs := sourceRef.Outgoings()
	if !IsGateway(source) && len(*outs) > 0 {
		outgoing := (*outs)[0]
		elem, _ := p.Elements.GetMut(string(outgoing))
		flow := elem.(schema.FlowNodeInterface)
		incomings := make([]schema.QName, 0)
		incomings = append(incomings, schema.QName(*dstId))
		flow.SetIncomings(incomings)
		dst.SetOutgoings(incomings)
	}

	p.AppendElem(dst)
	p.Link(src, *dstId, cond)

	return p
}

// BeforeInsertElem inserts a new element to SubProcess before the specified Element
func (p *SubProcessBuilder) BeforeInsertElem(dst string, src schema.FlowNodeInterface, cond *schema.AnExpression) *SubProcessBuilder {
	target, ok := p.Elements.GetMut(dst)
	if !ok {
		return p
	}
	targetRef, ok := target.(schema.FlowNodeInterface)
	if !ok {
		return p
	}

	srcId, _ := src.Id()

	ints := targetRef.Incomings()
	if !IsGateway(target) && len(*ints) > 0 {
		incoming := (*ints)[0]
		elem, _ := p.Elements.GetMut(string(incoming))
		outgoings := make([]schema.QName, 0)
		outgoings = append(outgoings, schema.QName(*srcId))
		flow := elem.(schema.FlowNodeInterface)
		flow.SetOutgoings(outgoings)
		src.SetIncomings(outgoings)
	}

	p.AppendElem(src)
	p.Link(*srcId, dst, cond)

	return p
}

func (p *SubProcessBuilder) Out() *schema.SubProcess {
	proc := p.SubProcess
	proc.ExtensionElementsField = p.ExtensionElement

	objects := make([]schema.DataObject, 0)
	p.Objects.Scan(func(key string, value *schema.DataObject) bool {
		objects = append(objects, *value)
		return true
	})
	proc.SetDataObjects(objects)

	storesReferences := make([]schema.DataStoreReference, 0)
	p.StoreReferences.Scan(func(key string, value *schema.DataStoreReference) bool {
		storesReferences = append(storesReferences, *value)
		return true
	})
	proc.SetDataStoreReferences(storesReferences)

	p.Elements.Scan(func(key string, value schema.FlowElementInterface) bool {
		switch elem := value.(type) {
		case *schema.StartEvent:
			if proc.StartEventField == nil {
				proc.StartEventField = make([]schema.StartEvent, 0)
			}
			proc.StartEventField = append(proc.StartEventField, *elem)
		case *schema.EndEvent:
			if proc.EndEventField == nil {
				proc.EndEventField = make([]schema.EndEvent, 0)
			}
			proc.EndEventField = append(proc.EndEventField, *elem)
		case *schema.SubProcess:
			if proc.SubProcessField == nil {
				proc.SubProcessField = make([]schema.SubProcess, 0)
			}
			proc.SubProcessField = append(proc.SubProcessField, *elem)
		case *schema.Task:
			if proc.TaskField == nil {
				proc.TaskField = make([]schema.Task, 0)
			}
			proc.TaskField = append(proc.TaskField, *elem)
		case *schema.ServiceTask:
			if proc.ServiceTaskField == nil {
				proc.ServiceTaskField = make([]schema.ServiceTask, 0)
			}
			proc.ServiceTaskField = append(proc.ServiceTaskField, *elem)
		case *schema.UserTask:
			if proc.UserTaskField == nil {
				proc.UserTaskField = make([]schema.UserTask, 0)
			}
			proc.UserTaskField = append(proc.UserTaskField, *elem)
		case *schema.SequenceFlow:
			if proc.SequenceFlowField == nil {
				proc.SequenceFlowField = make([]schema.SequenceFlow, 0)
			}
			proc.SequenceFlowField = append(proc.SequenceFlowField, *elem)
		}

		return true
	})

	return proc
}

func (p *SubProcessBuilder) Draw(startCoordX, startCoordY schema.Double) ([]schema.BPMNShape, []schema.BPMNEdge) {

	coordMap, bpmnShapes, bpmnEdges := draw(p.start, p.Elements, startCoordX, startCoordY)

	rootShape := schema.BPMNShape{}
	rootId := p.id + "_di"
	rootShape.SetId(&rootId)
	rootElem := schema.QName(p.id)
	rootShape.SetBpmnElement(&rootElem)
	rootShape.SetIsExpanded(schema.NewBoolP(true))
	rootWidth, rootHeight := getFlowSize(&schema.SubProcess{})
	rootShape.SetBounds(&schema.Bounds{
		XField:      startCoordX - 30,
		YField:      startCoordY - rootHeight/2,
		WidthField:  rootWidth,
		HeightField: rootHeight,
	})

	endId, _ := p.end.Id()
	if coord, ok := coordMap[*endId]; ok {
		rootShape.BoundsField.WidthField = coord.x + 60 - startCoordX
	}
	bpmnShapes = append([]schema.BPMNShape{rootShape}, bpmnShapes...)

	return bpmnShapes, bpmnEdges
}
