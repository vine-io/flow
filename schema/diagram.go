package schema

type Diagram struct {
	MetaName
	Plane *Plane `xml:"bpmndi:BPMNPlane,omitempty"`
}

type Plane struct {
	MetaName
	Element string   `xml:"bpmnElement,attr,omitempty"`
	Shapes  []*Shape `xml:"bpmndi:BPMNShape,omitempty"`
	Edges   []*Edge  `xml:"bpmndi:BPMNEdge,omitempty"`
}

type Shape struct {
	MetaName
	Element         string `xml:"bpmnElement,attr,omitempty"`
	Bound           *Bound `xml:"dc:Bounds,omitempty"`
	IsHorizontal    bool   `xml:"isHorizontal,attr,omitempty"`
	IsMarkerVisible bool   `xml:"isMarkerVisible,attr,omitempty"`
	Label           *Label `xml:"bpmndi:BPMNLabel,omitempty"`
}

type Edge struct {
	MetaName
	Element   string      `xml:"bpmnElement,attr,omitempty"`
	Waypoints []*Waypoint `xml:"di:waypoint,omitempty"`
	Label     *Label      `xml:"bpmndi:BPMNLabel,omitempty"`
}

type Label struct {
	Bound *Bound `xml:"dc:Bounds,omitempty"`
}

type Bound struct {
	X      int64 `xml:"x,attr,omitempty"`
	Y      int64 `xml:"y,attr,omitempty"`
	Height int64 `xml:"height,attr,omitempty"`
	Weight int64 `xml:"weight,attr,omitempty"`
}

type Waypoint struct {
	X int64 `xml:"x,attr,omitempty"`
	Y int64 `xml:"y,attr,omitempty"`
}
