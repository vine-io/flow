package schema

import "encoding/xml"

type Diagram struct {
	xml.Name `xml:"bpmndi:BPMNDiagram"`
	ID       string `xml:"id,attr,omitempty"`
	Plane    *Plane `xml:"bpmndi:BPMNPlane,omitempty"`
}

type Plane struct {
	xml.Name `xml:"bpmndi:BPMNPlane"`
	ID       string   `xml:"id,attr,omitempty"`
	Element  string   `xml:"bpmnElement,attr,omitempty"`
	Shapes   []*Shape `xml:"bpmndi:BPMNShape,omitempty"`
	Edges    []*Edge  `xml:"bpmndi:BPMNEdge,omitempty"`
}

type Shape struct {
	xml.Name        `xml:"bpmndi:BPMNEdge"`
	ID              string `xml:"id,attr,omitempty"`
	Element         string `xml:"bpmnElement,attr,omitempty"`
	Bound           *Bound `xml:"dc:Bounds,omitempty"`
	IsHorizontal    bool   `xml:"isHorizontal,attr,omitempty"`
	IsMarkerVisible bool   `xml:"isMarkerVisible,attr,omitempty"`
	Label           *Label `xml:"bpmndi:BPMNLabel,omitempty"`
}

type Edge struct {
	xml.Name  `xml:"bpmndi:BPMNEdge"`
	ID        string      `xml:"id,attr,omitempty"`
	Element   string      `xml:"bpmnElement,attr,omitempty"`
	Waypoints []*Waypoint `xml:"di:waypoint,omitempty"`
	Label     *Label      `xml:"bpmndi:BPMNLabel,omitempty"`
}

type Label struct {
	xml.Name `xml:"bpmndi:BPMNLabel"`
	Bound    *Bound `xml:"dc:Bounds,omitempty"`
}

type Bound struct {
	xml.Name `xml:"dc:Bounds,omitempty"`
	X        int64 `xml:"x,attr,omitempty"`
	Y        int64 `xml:"y,attr,omitempty"`
	Height   int64 `xml:"height,attr,omitempty"`
	Weight   int64 `xml:"weight,attr,omitempty"`
}

type Waypoint struct {
	xml.Name `xml:"di:waypoint"`
	X        int64 `xml:"x,attr,omitempty"`
	Y        int64 `xml:"y,attr,omitempty"`
}
