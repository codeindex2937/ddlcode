package drawio

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
				MxGraphModelBase: MxGraphModelBase{
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
				},
				Cells: []MxCell{
					&Shape{MxCellBase: MxCellBase{Id: "0"}},
					&Shape{MxCellBase: MxCellBase{Id: "1", Parent: "0"}},
				},
			},
		},
	}
}

func NewShape(id string, x, y, width, height float64, style map[string]string) *Shape {
	return &Shape{
		MxCellBase: MxCellBase{
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
		MxCellBase: MxCellBase{
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
			MxCellBase: MxCellBase{
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
			Connectable: "0",
		}, &Label{
			MxCellBase: MxCellBase{
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
