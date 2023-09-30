package drawio

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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

type MxGraphModel struct {
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
	Shapes     []MxCell `xml:"root>mxCell"`
}

type MxCell interface {
	GetId() string
	SetParent(id string)
}

type mxCell struct {
	XMLName  xml.Name  `xml:"mxCell"`
	Id       string    `xml:"id,attr"`
	Parent   string    `xml:"parent,attr,omitempty"`
	Style    string    `xml:"style,attr,omitempty"`
	Vertex   string    `xml:"vertex,attr,omitempty"`
	Value    string    `xml:"value,attr,omitempty"`
	Geometry *Geometry `xml:"mxGeometry"`
}

func (c mxCell) GetId() string {
	return c.Id
}

func (c *mxCell) SetParent(id string) {
	c.Parent = id
}

type Shape struct {
	mxCell
}

type Line struct {
	mxCell
	Source string `xml:"source,attr"`
	Target string `xml:"target,attr"`
	Edge   string `xml:"edge,attr"`
}

type Label struct {
	mxCell
	Connectable int `xml:"connectable,attr"`
	Edge        int `xml:"edge,attr"`
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

func NewFile(width, height int) *MxFile {
	return &MxFile{
		Host:     "Electron",
		Modified: time.Now(),
		Agent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) draw.io/21.7.5 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",
		Etag:     "va3-sl2bMTW4jECdg_Ks",
		Version:  "21.7.5",
		Type:     "device",
		Diagram: Diagram{
			Name: "Page-1",
			Id:   uuid.NewString(),
			MxGraphModel: MxGraphModel{
				Dx:         width / 2,
				Dy:         height / 2,
				Grid:       1,
				GridSize:   10,
				Guides:     1,
				Tooltips:   1,
				Connect:    1,
				Arrows:     1,
				Fold:       1,
				Page:       1,
				PageScale:  1,
				PageWidth:  width,
				PageHeight: height,
				Background: "none",
				Math:       0,
				Shadow:     0,
				Shapes: []MxCell{
					&Shape{mxCell: mxCell{Id: "0"}},
					&Shape{mxCell: mxCell{Id: "1", Parent: "0"}},
				},
			},
		},
	}
}

func (m *MxGraphModel) AddCells(cells ...MxCell) {
	for _, cell := range cells {
		cell.SetParent(m.Shapes[1].GetId())
	}
	m.Shapes = append(m.Shapes, cells...)
}

func NewShape(id string, x, y, width, height float64, style map[string]string) *Shape {
	return &Shape{
		mxCell: mxCell{
			Id:     id,
			Vertex: "1",
			Style:  join(style, "="),
			Geometry: &Geometry{
				X:      strconv.FormatFloat(x, 'f', -1, 64),
				Y:      strconv.FormatFloat(y, 'f', -1, 64),
				Width:  strconv.FormatFloat(width, 'f', -1, 64),
				Height: strconv.FormatFloat(height, 'f', -1, 64),
				As:     "geometry",
			},
		},
	}
}

func NewLine(parentId, sourceId, targetId string, style map[string]string) *Line {
	return &Line{
		mxCell: mxCell{
			Id:     RandId() + "-1",
			Parent: parentId,
			Style:  join(style, "="),
			Geometry: &Geometry{
				Relative: "1",
				As:       "geometry",
			},
		},
		Edge:   "1",
		Source: sourceId,
		Target: targetId,
	}
}

func (s Line) NewEdgeLabel(style map[string]string) (*Label, *Label) {
	id := RandId()
	styleStr := join(style, "=")
	return &Label{
			mxCell: mxCell{
				Id:     id + "-1",
				Parent: s.Id,
				Vertex: "1",
				Style:  styleStr,
				Geometry: &Geometry{
					X:        "-0.8",
					Y:        "0",
					Relative: "1",
					As:       "geometry",
					Point:    &Point{As: "offset"},
				},
			},
			Connectable: 0,
		}, &Label{
			mxCell: mxCell{
				Id:     id + "-2",
				Parent: s.Id,
				Vertex: "1",
				Style:  styleStr,
				Geometry: &Geometry{
					X:        "0.8",
					Y:        "0",
					Relative: "1",
					As:       "geometry",
					Point:    &Point{As: "offset"},
				},
			},
		}
}

func RandId() string {
	r := make([]byte, 15)
	if _, err := rand.Read(r); err != nil {
		log.Fatal(err)
	}

	return base64.StdEncoding.EncodeToString(r)
}

func join(style map[string]string, assignChar string) string {
	sb := strings.Builder{}
	for k, v := range style {
		if len(v) > 0 {
			sb.WriteString(fmt.Sprintf("%v%v%v;", k, assignChar, v))
		} else {
			sb.WriteString(fmt.Sprintf("%v;", k))
		}
	}
	return sb.String()
}
