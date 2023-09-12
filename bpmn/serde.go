package bpmn

import (
	"fmt"
	"html"
	"strconv"

	"github.com/beevik/etree"
	"github.com/tidwall/btree"
)

var serializers = map[string]Serializer{
	getKind(new(Definitions)):      &definitionSerde{},
	getKind(new(Process)):          &processSerde{},
	getKind(new(ExtensionElement)): &extensionElementSerde{},
	getKind(new(StartEvent)):       &startEventSerde{},
	getKind(new(EndEvent)):         &endEventSerde{},
	getKind(new(SequenceFlow)):     &sequenceFlowSerde{},
	getKind(new(ExclusiveGateway)): &exclusiveGatewaySerde{},
	getKind(new(InclusiveGateway)): &inclusiveGatewaySerde{},
	getKind(new(ParallelGateway)):  &parallelGatewaySerde{},
	getKind(new(Task)):             &taskSerde{},
	getKind(new(ServiceTask)):      &serviceTaskSerde{},
	getKind(new(UserTask)):         &userTaskSerde{},
	getKind(new(DataObject)):       &dataObjectSerde{},
	getKind(new(DataStore)):        &dataStoreSerde{},
	getKind(new(Diagram)):          &diagramSerde{},
	getKind(new(DiagramPlane)):     &diagramPlaneSerde{},
	getKind(new(DiagramEdge)):      &diagramEdgeSerde{},
	getKind(new(DiagramShape)):     &diagramShapeSerde{},
	getKind(new(DiagramLabel)):     &diagramLabelSerde{},
}

var deserializers = map[string]Deserializer{
	"bpmn:definitions":       &definitionSerde{},
	"bpmn:process":           &processSerde{},
	"bpmn:extensionElements": &extensionElementSerde{},
	"bpmn:startEvent":        &startEventSerde{},
	"bpmn:endEvent":          &endEventSerde{},
	"bpmn:sequenceFlow":      &sequenceFlowSerde{},
	"bpmn:exclusiveGateway":  &exclusiveGatewaySerde{},
	"bpmn:inclusiveGateway":  &inclusiveGatewaySerde{},
	"bpmn:parallelGateway":   &parallelGatewaySerde{},
	"bpmn:task":              &taskSerde{},
	"bpmn:serviceTask":       &serviceTaskSerde{},
	"bpmn:userTask":          &userTaskSerde{},
	"bpmn:dataObject":        &dataObjectSerde{},
	"bpmn:dataStore":         &dataStoreSerde{},
	"bpmndi:BPMNDiagram":     &diagramSerde{},
	"bpmndi:BPMNPlane":       &diagramPlaneSerde{},
	"bpmndi:BPMNEdge":        &diagramEdgeSerde{},
	"bpmndi:BPMNShape":       &diagramShapeSerde{},
	"bpmndi:BPMNLabel":       &diagramLabelSerde{},
}

type Serializer interface {
	Serialize(element any, start *etree.Element) error
}

type Deserializer interface {
	Deserialize(start *etree.Element) (any, error)
}

func Serialize(element any, start *etree.Element) error {
	kind := getKind(element)
	serializer, ok := serializers[kind]
	if !ok {
		return fmt.Errorf("%s not support to serialize", kind)
	}

	return serializer.Serialize(element, start)
}

func Deserialize(start *etree.Element) (any, error) {
	deserializer, ok := deserializers[start.FullTag()]
	if !ok {
		return nil, fmt.Errorf("%s not support to deserialize", start.FullTag())
	}

	return deserializer.Deserialize(start)
}

type definitionSerde struct{}

func (s *definitionSerde) Serialize(element any, start *etree.Element) error {
	definitions, ok := element.(*Definitions)
	if !ok {
		return fmt.Errorf("%v is not Definiations", element)
	}

	start.Space = "bpmn"
	start.Tag = "definitions"
	if definitions.Bpmn != "" {
		start.CreateAttr("xmlns:bpmn", definitions.Bpmn)
	}
	if definitions.BpmnDI != "" {
		start.CreateAttr("xmlns:bpmndi", definitions.BpmnDI)
	}
	if definitions.DC != "" {
		start.CreateAttr("xmlns:dc", definitions.DC)
	}
	if definitions.DI != "" {
		start.CreateAttr("xmlns:di", definitions.DI)
	}
	if definitions.XSI != "" {
		start.CreateAttr("xmlns:xsi", definitions.XSI)
	}
	if definitions.Olive != "" {
		start.CreateAttr("xmlns:olive", definitions.Olive)
	}
	if definitions.TargetNamespace != "" {
		start.CreateAttr("targetNamespace", definitions.TargetNamespace)
	}
	if definitions.Id != "" {
		start.CreateAttr("id", definitions.Id)
	}

	for _, elem := range definitions.elements {
		child := start.CreateElement("")
		if err := Serialize(elem, child); err != nil {
			return err
		}
		start.AddChild(child)
	}

	if definitions.Diagram != nil {
		child := start.CreateElement("")
		if err := Serialize(definitions.Diagram, child); err != nil {
			return err
		}
		start.AddChild(child)
	}

	return nil
}

func (s *definitionSerde) Deserialize(start *etree.Element) (any, error) {
	d := &Definitions{}
	for _, attr := range start.Attr {
		switch attr.FullKey() {
		case "xmlns:bpmn":
			d.Bpmn = attr.Value
		case "xmlns:bpmndi":
			d.BpmnDI = attr.Value
		case "xmlns:dc":
			d.DC = attr.Value
		case "xmlns:di":
			d.DI = attr.Value
		case "xmlns:xsi":
			d.XSI = attr.Value
		case "xmlns:olive":
			d.Olive = attr.Value
		case "targetNamespace":
			d.TargetNamespace = attr.Value
		case "id":
			d.Id = attr.Value
		}
	}

	d.elements = make([]Element, 0)
	for _, child := range start.ChildElements() {
		elem, err := Deserialize(child)
		if err != nil {
			return nil, err
		}

		if v, ok := elem.(Element); ok {
			d.elements = append(d.elements, v)
		}

		if v, ok := elem.(*Diagram); ok {
			d.Diagram = v
		}
	}

	return d, nil
}

type elementSerde struct{}

func (s *elementSerde) serialize(element Element, start *etree.Element) error {
	if element.GetID() != "" {
		start.CreateAttr("id", element.GetID())
	}
	if element.GetName() != "" {
		start.CreateAttr("name", element.GetName())
	}

	if element.GetExtension() != nil {
		child := start.CreateElement("")
		if err := Serialize(element.GetExtension(), child); err != nil {
			return err
		}
		start.AddChild(child)
	}
	for _, elem := range element.GetIncoming() {
		child := start.CreateElement("bpmn:incoming")
		child.SetText(elem)
		start.AddChild(child)
	}

	for _, elem := range element.GetOutgoing() {
		child := start.CreateElement("bpmn:outgoing")
		child.SetText(elem)
		start.AddChild(child)
	}

	return nil
}

func (s *elementSerde) deserialize(start *etree.Element, element Element, ranger func(element *etree.Element) error) error {
	id, _ := getAttr(start.Attr, "id")
	if id != "" {
		element.SetID(id)
	}
	name, _ := getAttr(start.Attr, "name")
	if name != "" {
		element.SetName(name)
	}

	incoming := make([]string, 0)
	outgoing := make([]string, 0)
	for _, child := range start.ChildElements() {
		switch child.FullTag() {
		case "bpmn:extensionElements":
			v, err := (&extensionElementSerde{}).Deserialize(child)
			if err != nil {
				return err
			}
			element.SetExtension(v.(*ExtensionElement))
		case "bpmn:incoming":
			incoming = append(incoming, child.Text())
		case "bpmn:outgoing":
			outgoing = append(outgoing, child.Text())
		default:
			if ranger != nil {
				if err := ranger(child); err != nil {
					return err
				}
			}
		}
	}
	if len(incoming) != 0 {
		element.SetIncoming(incoming)
	}
	if len(outgoing) != 0 {
		element.SetOutgoing(outgoing)
	}

	return nil
}

type processSerde struct{ inner elementSerde }

func (s *processSerde) Serialize(element any, start *etree.Element) error {
	process, ok := element.(*Process)
	if !ok {
		return fmt.Errorf("%v is not Definiations", element)
	}

	start.Space = "bpmn"
	start.Tag = "process"

	if err := s.inner.serialize(process, start); err != nil {
		return err
	}

	if process.Executable {
		start.CreateAttr("isExecutable", "true")
	}

	var err error
	process.Elements.Scan(func(key string, elem Element) bool {
		child := start.CreateElement("")
		if err = Serialize(elem, child); err != nil {
			return false
		}
		start.AddChild(child)
		return true
	})
	process.ObjectReferences.Scan(func(key string, reference *DataReference) bool {
		child := start.CreateElement("bpmn:dataObjectReference")
		child.CreateAttr("id", reference.Id)
		child.CreateAttr("dataObjectRef", reference.Ref)
		start.AddChild(child)
		return true
	})
	process.StoreReferences.Scan(func(key string, reference *DataReference) bool {
		child := start.CreateElement("bpmn:dataStoreReference")
		child.CreateAttr("id", reference.Id)
		child.CreateAttr("dataStoreRef", reference.Ref)
		start.AddChild(child)
		return true
	})

	return nil
}

func (s *processSerde) Deserialize(start *etree.Element) (any, error) {
	p := &Process{}

	if v, _ := getAttr(start.Attr, "isExecutable"); v == "true" {
		p.Executable = true
	}

	p.Elements = &btree.Map[string, Element]{}
	p.ObjectReferences = &btree.Map[string, *DataReference]{}
	p.StoreReferences = &btree.Map[string, *DataReference]{}
	ranger := func(root *etree.Element) error {
		switch root.FullTag() {
		case "bpmn:dataObjectReference":
			reference := &DataReference{}
			reference.Id, _ = getAttr(root.Attr, "id")
			reference.Ref, _ = getAttr(root.Attr, "dataObjectRef")
			p.ObjectReferences.Set(reference.Id, reference)
			return nil
		case "bpmn:dataStoreReference":
			reference := &DataReference{}
			reference.Id, _ = getAttr(root.Attr, "id")
			reference.Ref, _ = getAttr(root.Attr, "dataStoreRef")
			p.StoreReferences.Set(reference.Id, reference)
			return nil
		}
		elem, err := Deserialize(root)
		if err != nil {
			return err
		}

		if v, ok := elem.(Element); ok {
			p.Elements.Set(v.GetID(), v)
			if vv, ok := v.(*StartEvent); ok {
				p.start = vv
			}
			if vv, ok := v.(*EndEvent); ok {
				p.end = vv
			}
		}

		return nil
	}

	if err := s.inner.deserialize(start, p, ranger); err != nil {
		return nil, err
	}

	return p, nil
}

type startEventSerde struct{ inner elementSerde }

func (s *startEventSerde) Serialize(element any, start *etree.Element) error {
	event, ok := element.(*StartEvent)
	if !ok {
		return fmt.Errorf("%v is not StartEvent", element)
	}
	start.Space = "bpmn"
	start.Tag = "startEvent"
	if err := s.inner.serialize(event, start); err != nil {
		return err
	}

	return nil
}

func (s *startEventSerde) Deserialize(start *etree.Element) (any, error) {
	event := &StartEvent{}
	if err := s.inner.deserialize(start, event, nil); err != nil {
		return nil, err
	}

	return event, nil
}

type endEventSerde struct{ inner elementSerde }

func (s *endEventSerde) Serialize(element any, start *etree.Element) error {
	event, ok := element.(*EndEvent)
	if !ok {
		return fmt.Errorf("%v is not StartEvent", element)
	}
	start.Space = "bpmn"
	start.Tag = "endEvent"
	if err := s.inner.serialize(event, start); err != nil {
		return err
	}

	return nil
}

func (s *endEventSerde) Deserialize(start *etree.Element) (any, error) {
	event := &EndEvent{}
	if err := s.inner.deserialize(start, event, nil); err != nil {
		return nil, err
	}

	return event, nil
}

type sequenceFlowSerde struct{}

func (s *sequenceFlowSerde) Serialize(element any, start *etree.Element) error {
	flow, ok := element.(*SequenceFlow)
	if !ok {
		return fmt.Errorf("%v is not SequenceFlow", element)
	}

	start.Space = "bpmn"
	start.Tag = "sequenceFlow"
	if flow.Id != "" {
		start.CreateAttr("id", flow.Id)
	}
	if flow.Name != "" {
		start.CreateAttr("name", flow.Name)
	}
	if flow.SourceRef != "" {
		start.CreateAttr("sourceRef", flow.SourceRef)
	}
	if flow.TargetRef != "" {
		start.CreateAttr("targetRef", flow.TargetRef)
	}
	if flow.ExtensionElement != nil {
		child := start.CreateElement("")
		if err := new(extensionElementSerde).Serialize(flow.ExtensionElement, child); err != nil {
			return err
		}
		start.AddChild(child)
	}
	if condition := flow.Condition; condition != nil {
		child := start.CreateElement("bpmn:conditionExpression")
		child.CreateAttr("xsi:type", condition.Type)
		child.SetText(condition.Value)
		start.AddChild(child)
	}

	return nil
}

func (s *sequenceFlowSerde) Deserialize(start *etree.Element) (any, error) {
	flow := &SequenceFlow{}

	for _, attr := range start.Attr {
		switch attr.FullKey() {
		case "id":
			flow.Id = attr.Value
		case "name":
			flow.Name = attr.Value
		case "sourceRef":
			flow.SourceRef = attr.Value
		case "targetRef":
			flow.TargetRef = attr.Value
		}
	}

	for _, child := range start.ChildElements() {
		switch child.FullTag() {
		case "bpmn:conditionExpression":
			typ, _ := getAttr(child.Attr, "xsi:type")
			flow.Condition = &ConditionExpression{
				Type:  typ,
				Value: child.Text(),
			}
		case "bpmn:extensiveElement":
			out, err := new(extensionElementSerde).Deserialize(child)
			if err != nil {
				return nil, err
			}
			flow.ExtensionElement = out.(*ExtensionElement)
		}
	}

	return flow, nil
}

type exclusiveGatewaySerde struct{ inner elementSerde }

func (s *exclusiveGatewaySerde) Serialize(element any, start *etree.Element) error {
	gw, ok := element.(*ExclusiveGateway)
	if !ok {
		return fmt.Errorf("%v is not ExclusiveGateway", element)
	}
	start.Space = "bpmn"
	start.Tag = "exclusiveGateway"
	if err := s.inner.serialize(gw, start); err != nil {
		return err
	}

	return nil
}

func (s *exclusiveGatewaySerde) Deserialize(start *etree.Element) (any, error) {
	gw := &ExclusiveGateway{}
	if err := s.inner.deserialize(start, gw, nil); err != nil {
		return nil, err
	}

	return gw, nil
}

type inclusiveGatewaySerde struct{ inner elementSerde }

func (s *inclusiveGatewaySerde) Serialize(element any, start *etree.Element) error {
	gw, ok := element.(*InclusiveGateway)
	if !ok {
		return fmt.Errorf("%v is not InclusiveGateway", element)
	}
	start.Space = "bpmn"
	start.Tag = "inclusiveGateway"
	if err := s.inner.serialize(gw, start); err != nil {
		return err
	}

	return nil
}

func (s *inclusiveGatewaySerde) Deserialize(start *etree.Element) (any, error) {
	gw := &InclusiveGateway{}
	if err := s.inner.deserialize(start, gw, nil); err != nil {
		return nil, err
	}

	return gw, nil
}

type parallelGatewaySerde struct{ inner elementSerde }

func (s *parallelGatewaySerde) Serialize(element any, start *etree.Element) error {
	gw, ok := element.(*ParallelGateway)
	if !ok {
		return fmt.Errorf("%v is not ParallelGateway", element)
	}
	start.Space = "bpmn"
	start.Tag = "parallelGateway"
	if err := s.inner.serialize(gw, start); err != nil {
		return err
	}

	return nil
}

func (s *parallelGatewaySerde) Deserialize(start *etree.Element) (any, error) {
	gw := &ParallelGateway{}
	if err := s.inner.deserialize(start, gw, nil); err != nil {
		return nil, err
	}

	return gw, nil
}

type taskSerde struct{ inner elementSerde }

func (s *taskSerde) Serialize(element any, start *etree.Element) error {
	task, ok := element.(*Task)
	if !ok {
		return fmt.Errorf("%v is not Task", element)
	}
	start.Space = "bpmn"
	start.Tag = "task"
	if err := s.inner.serialize(task, start); err != nil {
		return err
	}
	for _, property := range task.Properties {
		child := start.CreateElement("bpmn:property")
		child.CreateAttr("id", property.Id)
		child.CreateAttr("name", property.Name)
		start.AddChild(child)
	}
	for _, ass := range task.DataInputAssociation {
		child := start.CreateElement("bpmn:dataInputAssociation")
		child.CreateAttr("id", ass.Id)
		source := child.CreateElement("bpmn:sourceRef")
		source.SetText(ass.Source)
		child.AddChild(source)
		target := child.CreateElement("bpmn:targetRef")
		target.SetText(ass.Target)
		child.AddChild(target)
		start.AddChild(child)
	}
	for _, ass := range task.DataOutputAssociation {
		child := start.CreateElement("bpmn:dataOutputAssociation")
		child.CreateAttr("id", ass.Id)
		target := child.CreateElement("bpmn:targetRef")
		target.SetText(ass.Target)
		child.AddChild(target)
		start.AddChild(child)
	}

	return nil
}

func (s *taskSerde) Deserialize(start *etree.Element) (any, error) {
	task := &Task{}

	properties := make([]*TaskProperty, 0)
	inputAssociation := make([]*DataAssociation, 0)
	outputAssociation := make([]*DataAssociation, 0)
	ranger := func(root *etree.Element) error {
		switch root.FullTag() {
		case "bpmn:property":
			property := &TaskProperty{}
			property.Id, _ = getAttr(root.Attr, "id")
			property.Name, _ = getAttr(root.Attr, "name")
			properties = append(properties, property)
		case "bpmn:dataInputAssociation":
			ass := &DataAssociation{}
			ass.Id, _ = getAttr(root.Attr, "id")
			for _, child := range root.ChildElements() {
				if child.FullTag() == "bpmn:sourceRef" {
					ass.Source = child.Text()
				}
				if child.FullTag() == "bpmn:targetRef" {
					ass.Target = child.Text()
				}
			}
			inputAssociation = append(inputAssociation, ass)
		case "bpmn:dataOutputAssociation":
			ass := &DataAssociation{}
			ass.Id, _ = getAttr(root.Attr, "id")
			for _, child := range root.ChildElements() {
				if child.FullTag() == "bpmn:sourceRef" {
					ass.Source = child.Text()
				}
				if child.FullTag() == "bpmn:targetRef" {
					ass.Target = child.Text()
				}
			}
			outputAssociation = append(outputAssociation, ass)
		}

		return nil
	}

	if err := s.inner.deserialize(start, task, ranger); err != nil {
		return nil, err
	}

	if len(properties) > 0 {
		task.Properties = properties
	}
	if len(inputAssociation) > 0 {
		task.DataInputAssociation = inputAssociation
	}
	if len(outputAssociation) > 0 {
		task.DataOutputAssociation = outputAssociation
	}

	return task, nil
}

type serviceTaskSerde struct{}

func (s *serviceTaskSerde) Serialize(element any, start *etree.Element) error {
	task, ok := element.(*ServiceTask)
	if !ok {
		return fmt.Errorf("%v is not ServiceTask", element)
	}

	target := task.Task
	if err := new(taskSerde).Serialize(&target, start); err != nil {
		return err
	}
	start.Space = "bpmn"
	start.Tag = "serviceTask"

	return nil
}

func (s *serviceTaskSerde) Deserialize(start *etree.Element) (any, error) {
	task := &ServiceTask{}
	v, err := new(taskSerde).Deserialize(start)
	if err != nil {
		return nil, err
	}
	task.Task = *v.(*Task)

	return task, nil
}

type userTaskSerde struct{}

func (s *userTaskSerde) Serialize(element any, start *etree.Element) error {
	task, ok := element.(*UserTask)
	if !ok {
		return fmt.Errorf("%v is not UserTask", element)
	}
	target := task.Task
	if err := new(taskSerde).Serialize(&target, start); err != nil {
		return err
	}
	start.Space = "bpmn"
	start.Tag = "userTask"

	return nil
}

func (s *userTaskSerde) Deserialize(start *etree.Element) (any, error) {
	task := &UserTask{}
	v, err := new(taskSerde).Deserialize(start)
	if err != nil {
		return nil, err
	}
	task.Task = *v.(*Task)

	return task, nil
}

type dataObjectSerde struct{ inner elementSerde }

func (s *dataObjectSerde) Serialize(element any, start *etree.Element) error {
	do, ok := element.(*DataObject)
	if !ok {
		return fmt.Errorf("%v is not DataObject", element)
	}
	start.Space = "bpmn"
	start.Tag = "dataObject"
	if err := s.inner.serialize(do, start); err != nil {
		return err
	}

	return nil
}

func (s *dataObjectSerde) Deserialize(start *etree.Element) (any, error) {
	do := &DataObject{}
	if err := s.inner.deserialize(start, do, nil); err != nil {
		return nil, err
	}

	return do, nil
}

type dataStoreSerde struct{ inner elementSerde }

func (s *dataStoreSerde) Serialize(element any, start *etree.Element) error {
	ds, ok := element.(*DataStore)
	if !ok {
		return fmt.Errorf("%v is not DataStore", element)
	}
	start.Space = "bpmn"
	start.Tag = "dataStore"
	if err := s.inner.serialize(ds, start); err != nil {
		return err
	}

	return nil
}

func (s *dataStoreSerde) Deserialize(start *etree.Element) (any, error) {
	ds := &DataStore{}
	if err := s.inner.deserialize(start, ds, nil); err != nil {
		return nil, err
	}

	return ds, nil
}

type extensionElementSerde struct{}

func (s *extensionElementSerde) Serialize(element any, start *etree.Element) error {
	e, ok := element.(*ExtensionElement)
	if !ok {
		return fmt.Errorf("%v is not ExtensionElement", element)
	}

	start.Space = "bpmn"
	start.Tag = "extensionElements"
	if d := e.TaskDefinition; d != nil {
		child := start.CreateElement("olive:taskDefinition")
		if d.Type != "" {
			child.CreateAttr("type", d.Type)
		}
		if d.Retries != "" {
			child.CreateAttr("retries", d.Retries)
		}
		start.AddChild(child)
	}
	if mapping := e.IOMapping; mapping != nil {
		child := start.CreateElement("olive:ioMapping")
		for _, input := range mapping.Input {
			elem := child.CreateElement("olive:input")
			elem.CreateAttr("source", html.EscapeString(input.Source))
			elem.CreateAttr("target", html.EscapeString(input.Target))
			child.AddChild(elem)
		}
		for _, output := range mapping.Output {
			elem := child.CreateElement("olive:output")
			elem.CreateAttr("source", html.EscapeString(output.Source))
			elem.CreateAttr("target", html.EscapeString(output.Target))
			child.AddChild(elem)
		}
		start.AddChild(child)
	}
	if p := e.Properties; p != nil {
		child := start.CreateElement("olive:properties")
		for _, item := range p.Items {
			elem := child.CreateElement("olive:property")
			elem.CreateAttr("name", item.Name)
			elem.CreateAttr("value", item.Value)
			elem.CreateAttr("type", "string")
			child.AddChild(elem)
		}
		start.AddChild(child)
	}
	if h := e.Headers; h != nil {
		child := start.CreateElement("olive:taskHeaders")
		for _, item := range h.Items {
			elem := child.CreateElement("olive:header")
			elem.CreateAttr("name", item.Name)
			elem.CreateAttr("value", item.Value)
			elem.CreateAttr("type", "string")
			child.AddChild(elem)
		}
		start.AddChild(child)
	}

	return nil
}

func (s *extensionElementSerde) Deserialize(start *etree.Element) (any, error) {
	elem := &ExtensionElement{}

	for _, child := range start.ChildElements() {
		switch child.FullTag() {
		case "olive:properties":
			elem.Properties = &Properties{Items: make([]*Property, 0)}
			for _, item := range child.ChildElements() {
				if item.FullTag() != "olive:property" {
					continue
				}
				property := &Property{}
				property.Name, _ = getAttr(item.Attr, "name")
				property.Value, _ = getAttr(item.Attr, "value")
				elem.Properties.Items = append(elem.Properties.Items, property)
			}
		case "olive:taskHeaders":
			elem.Headers = &TaskHeaders{Items: make([]*HeaderItem, 0)}
			for _, item := range child.ChildElements() {
				if item.FullTag() != "olive:header" {
					continue
				}
				header := &HeaderItem{}
				header.Name, _ = getAttr(item.Attr, "name")
				header.Value, _ = getAttr(item.Attr, "value")
				elem.Headers.Items = append(elem.Headers.Items, header)
			}
		case "olive:taskDefinition":
			elem.TaskDefinition = &TaskDefinition{}
			elem.TaskDefinition.Type, _ = getAttr(child.Attr, "type")
			elem.TaskDefinition.Retries, _ = getAttr(child.Attr, "retries")
		case "olive:ioMapping":
			elem.IOMapping = &IOMapping{
				Input:  []*Mapping{},
				Output: []*Mapping{},
			}

			for _, item := range child.ChildElements() {
				switch item.FullTag() {
				case "olive:input":
					input := &Mapping{}
					input.Source, _ = getAttr(item.Attr, "source")
					input.Target, _ = getAttr(item.Attr, "target")
					elem.IOMapping.Input = append(elem.IOMapping.Input, input)
				case "olive:output":
					output := &Mapping{}
					output.Source, _ = getAttr(item.Attr, "source")
					output.Target, _ = getAttr(item.Attr, "target")
					elem.IOMapping.Output = append(elem.IOMapping.Output, output)
				}
			}
		}
	}

	return elem, nil
}

type diagramSerde struct{}

func (s *diagramSerde) Serialize(element any, start *etree.Element) error {
	d, ok := element.(*Diagram)
	if !ok {
		return fmt.Errorf("%v is not Diagram", element)
	}

	start.Space = "bpmndi"
	start.Tag = "BPMNDiagram"
	if d.Id != "" {
		start.CreateAttr("id", d.Id)
	}

	for _, plane := range d.Planes {
		child := start.CreateElement("")
		if err := Serialize(plane, child); err != nil {
			return err
		}
		start.AddChild(child)
	}

	return nil
}

func (s *diagramSerde) Deserialize(start *etree.Element) (any, error) {
	d := &Diagram{}

	d.Id, _ = getAttr(start.Attr, "id")

	d.Planes = make([]*DiagramPlane, 0)
	for _, child := range start.ChildElements() {
		v, err := Deserialize(child)
		if err != nil {
			return nil, err
		}
		d.Planes = append(d.Planes, v.(*DiagramPlane))
	}

	return d, nil
}

type diagramPlaneSerde struct{}

func (s *diagramPlaneSerde) Serialize(element any, start *etree.Element) error {
	plane, ok := element.(*DiagramPlane)
	if !ok {
		return fmt.Errorf("%v is not DiagramPlane", element)
	}

	start.Space = "bpmndi"
	start.Tag = "BPMNPlane"
	if plane.Id != "" {
		start.CreateAttr("id", plane.Id)
	}
	if plane.Element != "" {
		start.CreateAttr("bpmnElement", plane.Element)
	}

	for _, shape := range plane.Shapes {
		child := start.CreateElement("")
		err := new(diagramShapeSerde).Serialize(shape, child)
		if err != nil {
			return err
		}
		start.AddChild(child)
	}

	for _, edge := range plane.Edges {
		child := start.CreateElement("")
		err := new(diagramEdgeSerde).Serialize(edge, child)
		if err != nil {
			return err
		}
		start.AddChild(child)
	}

	return nil
}

func (s *diagramPlaneSerde) Deserialize(start *etree.Element) (any, error) {
	plane := &DiagramPlane{}

	plane.Id, _ = getAttr(start.Attr, "id")
	plane.Element, _ = getAttr(start.Attr, "bpmnElement")

	plane.Edges = []*DiagramEdge{}
	plane.Shapes = []*DiagramShape{}
	for _, child := range start.ChildElements() {
		elem, err := Deserialize(child)
		if err != nil {
			return nil, err
		}
		if v, ok := elem.(*DiagramEdge); ok {
			plane.Edges = append(plane.Edges, v)
		}
		if v, ok := elem.(*DiagramShape); ok {
			plane.Shapes = append(plane.Shapes, v)
		}
	}

	return plane, nil
}

type diagramShapeSerde struct{}

func (s *diagramShapeSerde) Serialize(element any, start *etree.Element) error {
	shape, ok := element.(*DiagramShape)
	if !ok {
		return fmt.Errorf("%v is not DiagramShape", element)
	}

	start.Space = "bpmndi"
	start.Tag = "BPMNShape"
	if shape.Id != "" {
		start.CreateAttr("id", shape.Id)
	}
	if shape.Element != "" {
		start.CreateAttr("bpmnElement", shape.Element)
	}

	if bounds := shape.Bounds; bounds != nil {
		child := start.CreateElement("dc:Bounds")
		child.CreateAttr("x", strconv.FormatInt(bounds.X, 10))
		child.CreateAttr("y", strconv.FormatInt(bounds.Y, 10))
		child.CreateAttr("width", strconv.FormatInt(bounds.Width, 10))
		child.CreateAttr("height", strconv.FormatInt(bounds.Height, 10))
		start.AddChild(child)
	}
	if label := shape.Label; label != nil {
		child := start.CreateElement("")
		err := new(diagramLabelSerde).Serialize(label, child)
		if err != nil {
			return err
		}
		start.AddChild(child)
	}

	return nil
}

func (s *diagramShapeSerde) Deserialize(start *etree.Element) (any, error) {
	shape := &DiagramShape{}

	shape.Id, _ = getAttr(start.Attr, "id")
	shape.Element, _ = getAttr(start.Attr, "bpmnElement")

	for _, child := range start.ChildElements() {
		if child.FullTag() == "bpmndi:BPMNLabel" {
			v, err := new(diagramLabelSerde).Deserialize(child)
			if err != nil {
				return nil, err
			}
			shape.Label = v.(*DiagramLabel)
		}
		if child.FullTag() == "dc:Bounds" {
			bounds := &DiagramBounds{}
			for _, attr := range child.Attr {
				switch attr.FullKey() {
				case "x":
					bounds.X, _ = strconv.ParseInt(attr.Value, 10, 64)
				case "y":
					bounds.Y, _ = strconv.ParseInt(attr.Value, 10, 64)
				case "width":
					bounds.Width, _ = strconv.ParseInt(attr.Value, 10, 64)
				case "height":
					bounds.Height, _ = strconv.ParseInt(attr.Value, 10, 64)
				}
			}
			shape.Bounds = bounds
		}
	}

	return shape, nil
}

type diagramEdgeSerde struct{}

func (s *diagramEdgeSerde) Serialize(element any, start *etree.Element) error {
	edge, ok := element.(*DiagramEdge)
	if !ok {
		return fmt.Errorf("%v is not DiagramEdge", element)
	}

	start.Space = "bpmndi"
	start.Tag = "BPMNEdge"
	if edge.Id != "" {
		start.CreateAttr("id", edge.Id)
	}
	if edge.Element != "" {
		start.CreateAttr("bpmnElement", edge.Element)
	}

	for _, waypoint := range edge.Waypoints {
		child := start.CreateElement("di:waypoint")
		child.CreateAttr("x", strconv.FormatInt(waypoint.X, 10))
		child.CreateAttr("y", strconv.FormatInt(waypoint.Y, 10))
		start.AddChild(child)
	}
	if label := edge.Label; label != nil {
		child := start.CreateElement("")
		err := new(diagramLabelSerde).Serialize(label, child)
		if err != nil {
			return err
		}
		start.AddChild(child)
	}

	return nil
}

func (s *diagramEdgeSerde) Deserialize(start *etree.Element) (any, error) {
	edge := &DiagramEdge{}

	edge.Id, _ = getAttr(start.Attr, "id")
	edge.Element, _ = getAttr(start.Attr, "bpmnElement")

	edge.Waypoints = []*DiagramWaypoint{}
	for _, child := range start.ChildElements() {
		if child.FullTag() == "bpmndi:BPMNLabel" {
			v, err := new(diagramLabelSerde).Deserialize(child)
			if err != nil {
				return nil, err
			}
			edge.Label = v.(*DiagramLabel)
		}
		if child.FullTag() == "di:waypoint" {
			waypoint := &DiagramWaypoint{}
			x, _ := getAttr(child.Attr, "x")
			waypoint.X, _ = strconv.ParseInt(x, 10, 64)
			y, _ := getAttr(child.Attr, "y")
			waypoint.Y, _ = strconv.ParseInt(y, 10, 64)
			edge.Waypoints = append(edge.Waypoints, waypoint)
		}
	}

	return edge, nil
}

type diagramLabelSerde struct{}

func (s *diagramLabelSerde) Serialize(element any, start *etree.Element) error {
	label, ok := element.(*DiagramLabel)
	if !ok {
		return fmt.Errorf("%v is not DiagramLabel", element)
	}

	start.Space = "bpmndi"
	start.Tag = "BPMNLabel"
	if bounds := label.Bounds; bounds != nil {
		child := start.CreateElement("dc:Bounds")
		child.CreateAttr("x", strconv.FormatInt(bounds.X, 10))
		child.CreateAttr("y", strconv.FormatInt(bounds.Y, 10))
		child.CreateAttr("width", strconv.FormatInt(bounds.Width, 10))
		child.CreateAttr("height", strconv.FormatInt(bounds.Height, 10))
		start.AddChild(child)
	}

	return nil
}

func (s *diagramLabelSerde) Deserialize(start *etree.Element) (any, error) {
	label := &DiagramLabel{}

	for _, child := range start.ChildElements() {
		if child.FullTag() == "dc:Bounds" {
			bounds := &DiagramBounds{}
			for _, attr := range child.Attr {
				switch attr.FullKey() {
				case "x":
					bounds.X, _ = strconv.ParseInt(attr.Value, 10, 64)
				case "y":
					bounds.Y, _ = strconv.ParseInt(attr.Value, 10, 64)
				case "width":
					bounds.Width, _ = strconv.ParseInt(attr.Value, 10, 64)
				case "height":
					bounds.Height, _ = strconv.ParseInt(attr.Value, 10, 64)
				}
			}
			label.Bounds = bounds
		}
	}

	return label, nil
}
