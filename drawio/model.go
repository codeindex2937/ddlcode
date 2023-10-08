package drawio

import (
	"encoding/xml"
	"log"
	"strings"
	"time"
)

type MxFile struct {
	XMLName  xml.Name  `xml:"mxfile"`
	Host     string    `xml:"host,attr"`
	Modified time.Time `xml:"modified,attr"`
	Agent    string    `xml:"agent,attr"`
	Etag     string    `xml:"etag,attr"`
	Version  string    `xml:"version,attr"`
	Type     string    `xml:"type,attr"`
	Diagram  Diagram   `xml:"diagram"`
}

type Diagram struct {
	XMLName      xml.Name     `xml:"diagram"`
	Name         string       `xml:"name,attr"`
	Id           string       `xml:"id,attr"`
	MxGraphModel MxGraphModel `xml:"mxGraphModel"`
}

type MxGraphModelBase struct {
	XMLName    xml.Name `xml:"mxGraphModel"`
	Dx         int      `xml:"dx,attr"`
	Dy         int      `xml:"dy,attr"`
	Grid       int      `xml:"grid,attr"`
	GridSize   int      `xml:"gridSize,attr"`
	Guides     int      `xml:"guides,attr"`
	Tooltips   int      `xml:"tooltips,attr"`
	Connect    int      `xml:"connect,attr"`
	Arrows     int      `xml:"arrows,attr"`
	Fold       int      `xml:"fold,attr"`
	Page       int      `xml:"page,attr"`
	PageScale  int      `xml:"pageScale,attr"`
	PageWidth  int      `xml:"pageWidth,attr"`
	PageHeight int      `xml:"pageHeight,attr"`
	Background string   `xml:"background,attr"`
	Math       int      `xml:"math,attr"`
	Shadow     int      `xml:"shadow,attr"`
}

type MxGraphModel struct {
	MxGraphModelBase
	Cells []MxCell `xml:"root>mxCell"`
}

func (m *MxGraphModel) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	proto := struct {
		MxGraphModelBase
		Cells []mxCell `xml:"root>mxCell"`
	}{}

	if err := d.DecodeElement(&proto, &start); err != nil {
		log.Printf("%v\n", err)
		return err
	}
	m.MxGraphModelBase = proto.MxGraphModelBase

	m.Cells = make([]MxCell, 0)
	for _, cell := range proto.Cells {
		if strings.Contains(cell.Style, "edgeLabel;") {
			m.Cells = append(m.Cells, cell.toLabel())
		} else if strings.Contains(cell.Style, "edgeStyle=") {
			m.Cells = append(m.Cells, cell.toLine())
		} else {
			m.Cells = append(m.Cells, cell.toShape())
		}
	}

	return nil
}

func (m *MxGraphModel) AddCells(cells ...MxCell) {
	for _, cell := range cells {
		if len(cell.GetParent()) > 0 {
			continue
		}
		cell.SetParent(m.Cells[1].GetId())
	}

	m.Cells = append(m.Cells, cells...)
}

type MxCell interface {
	GetId() string
	GetParent() string
	SetParent(id string)
}

type MxCellBase struct {
	XMLName  xml.Name  `xml:"mxCell"`
	Id       string    `xml:"id,attr"`
	Parent   string    `xml:"parent,attr,omitempty"`
	Style    string    `xml:"style,attr,omitempty"`
	Vertex   string    `xml:"vertex,attr,omitempty"`
	Value    string    `xml:"value,attr,omitempty"`
	Geometry *Geometry `xml:"mxGeometry"`
}

type mxCell struct {
	MxCellBase
	Source      string `xml:"source,attr"`
	Target      string `xml:"target,attr"`
	Edge        string `xml:"edge,attr"`
	Connectable string `xml:"connectable,attr"`
}

func (c MxCellBase) GetId() string {
	return c.Id
}

func (c MxCellBase) GetParent() string {
	return c.Parent
}

func (c *MxCellBase) SetParent(id string) {
	c.Parent = id
}

func (c mxCell) toShape() *Shape {
	return &Shape{
		MxCellBase: c.MxCellBase,
	}
}

func (c mxCell) toLine() *Line {
	return &Line{
		MxCellBase: c.MxCellBase,
		Source:     c.Source,
		Target:     c.Target,
		Edge:       c.Edge,
	}
}

func (c mxCell) toLabel() *Label {
	return &Label{
		MxCellBase:  c.MxCellBase,
		Connectable: c.Connectable,
		Edge:        c.Edge,
	}
}

type Shape struct {
	MxCellBase
}

type Line struct {
	MxCellBase
	Source string `xml:"source,attr"`
	Target string `xml:"target,attr"`
	Edge   string `xml:"edge,attr"`
}

type Label struct {
	MxCellBase
	Connectable string `xml:"connectable,attr"`
	Edge        string `xml:"edge,attr"`
}

type Geometry struct {
	XMLName  xml.Name `xml:"mxGeometry"`
	X        string   `xml:"x,attr,omitempty"`
	Y        string   `xml:"y,attr,omitempty"`
	Width    string   `xml:"width,attr,omitempty"`
	Height   string   `xml:"height,attr,omitempty"`
	Relative string   `xml:"relative,attr,omitempty"`
	As       string   `xml:"as,attr"`
	Point    *Point   `xml:"mxPoint"`
}

type Point struct {
	XMLName xml.Name `xml:"mxPoint"`
	As      string   `xml:"as,attr"`
}
