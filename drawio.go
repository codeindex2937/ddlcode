package ddlcode

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/codeindex2937/oracle-sql-parser/ast"
	"github.com/codeindex2937/oracle-sql-parser/ast/element"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
)

type entity struct {
	*Table
	HeaderStyle string
	TableStyle  string
	CellStyle   string
	X           int
	Y           int
	Width       int
	Height      int
}

type DrawioConfig struct {
	ExportPath     string
	CellId         string
	EntityStyle    string
	Width          int
	Height         int
	Tables         []*Table
	Template       *template.Template
	HeaderStyle    string
	TableStyle     string
	LinkStyle      string
	EdgeLabelStyle string
	CellStyle      string
}

type drawioContext struct {
	DrawioConfig
	ModTime  string
	Entities []entity
	Links    []drawioLink
}

type drawioTable struct {
	Id string
}

type drawioLink struct {
	Id            string
	LineStyle     string
	Source        string
	Target        string
	LabelId       string
	LabelStyle    string
	RefColumn     string
	ForeignColumn string
}

var DrawioFuncMap = template.FuncMap{
	"GenUUID":   func() string { return uuid.NewString() },
	"Half":      func(v int) int { return v / 2 },
	"GetEntity": getEntity,
}

var DrawioTEmplate = `<mxfile host="Electron" modified="{{ .ModTime }}" agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) draw.io/21.7.5 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36" etag="va3-sl2bMTW4jECdg_Ks" version="21.7.5" type="device">
  <diagram name="Page-1" id="{{ GenUUID }}">
    <mxGraphModel dx="{{Half .Width}}" dy="{{Half .Height}}" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="{{.Width}}" pageHeight="{{.Height}}" background="none" math="0" shadow="0">
      <root>
        <mxCell id="0" />
        <mxCell id="1" parent="0" />
		{{- $config := . -}}
		{{- range $i, $entity := .Entities }}
        <mxCell id="{{$config.CellId}}-{{$i}}" value="{{ GetEntity $entity }}" style="{{ $config.EntityStyle }}" parent="1" vertex="1">
          <mxGeometry x="{{ $entity.X }}" y="{{ $entity.Y }}" width="{{ $entity.Width }}" height="{{ $entity.Height }}" as="geometry" />
        </mxCell>
		{{- end }}
		{{- range .Links }}
			<mxCell id="{{ .Id }}" style="{{ .LineStyle }}" edge="1" parent="1" source="{{ .Source }}" target="{{ .Target }}">
				<mxGeometry relative="1" as="geometry" />
			</mxCell>
			<mxCell id="{{ .LabelId }}-1" value="{{ .RefColumn }}" style="{{ .LabelStyle }}" parent="{{ .Id }}" vertex="1" connectable="0">
				<mxGeometry x="-0.9" y="0" relative="1" as="geometry">
					<mxPoint as="offset" />
				</mxGeometry>
			</mxCell>
			<mxCell id="{{ .LabelId }}-2" value="{{ .ForeignColumn }}" style="{{ .LabelStyle }}" parent="{{ .Id }}" vertex="1" connectable="0">
				<mxGeometry x="0.9" y="0" relative="1" as="geometry">
					<mxPoint as="offset" />
				</mxGeometry>
			</mxCell>
		{{- end }}
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
`

var entityFuncMap = template.FuncMap{
	"IsCompositePrimaryKey": isCompositePrimaryKey,
	"GetDefaultValue":       getDefaultValueFromAttribute,
	"ToSqlType":             toSqlType,
}

var entityTemplate, _ = template.New("drawioEntity").Funcs(entityFuncMap).Parse(
	`<div style="display:flex;flex-direction:column;height:100%;"><div style="{{.HeaderStyle}}flex:0">{{ .Table.Name }}</div>
<table style="{{.TableStyle}}flex:1;">
{{- $ctx := . -}}
{{- range .Table.Columns }}
<tr>
<td style="{{ $ctx.CellStyle }}">{{ .Name }}</td>
<td style="{{ $ctx.CellStyle }}">{{ ToSqlType .Type }}</td>
<td style="{{ $ctx.CellStyle }}">{{ if .Attribute.IsNotNull }}NN{{else}}-{{ end }}</td>
<td style="{{ $ctx.CellStyle }}">{{ if .Attribute.IsPrimaryKey }}PK{{else}}-{{ end }}</td>
<td style="{{ $ctx.CellStyle }}">{{ if .Attribute.IsAutoIncrement }}AI{{else}}-{{ end }}</td>
<td style="{{ $ctx.CellStyle }}">{{ if .Attribute.IsUnique }}U{{else}}-{{ end }}</td>
<td style="{{ $ctx.CellStyle }}">{{ GetDefaultValue .Attribute }}</td>
</tr>
{{- end }}
</table></div>
`)

func GetDefaultDrawioConfig() DrawioConfig {
	var err error
	buf := make([]byte, 8)

	_, err = rand.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	config := DrawioConfig{
		CellId: hex.EncodeToString(buf),
		EntityStyle: join(map[string]string{
			"verticalAlign":        "top",
			"align":                "left",
			"overflow":             "fill",
			"html":                 "1",
			"rounded":              "0",
			"shadow":               "0",
			"comic":                "0",
			"labelBackgroundColor": "none",
			"strokeWidth":          "1",
			"fontFamily":           "Verdana",
			"fontSize":             "12",
		}, "="),
		HeaderStyle: join(map[string]string{
			"box-sizing": "border-box",
			"width":      "100%",
			"background": "#e4e4e4",
			"padding":    "2px",
			"color":      "black",
		}, ":"),
		TableStyle: join(map[string]string{
			"width":           "100%",
			"font-size":       "1em",
			"background":      "DimGray",
			"border-collapse": "collapse",
		}, ":"),
		LinkStyle: join(map[string]string{
			"edgeStyle":      "orthogonalEdgeStyle",
			"rounded":        "0",
			"orthogonalLoop": "1",
			"jettySize":      "auto",
			"html":           "1",
			"exitX":          "1",
			"exitY":          "0.5",
			"exitDx":         "0",
			"exitDy":         "0",
			"entryX":         "0",
			"entryY":         "0.5",
			"entryDx":        "0",
			"entryDy":        "0",
		}, "="),
		EdgeLabelStyle: join(map[string]string{
			"edgeLabel":     "",
			"html":          "1",
			"align":         "center",
			"verticalAlign": "middle",
			"resizable":     "0",
			"points":        "[]",
		}, "="),
		CellStyle: join(map[string]string{
			"border": "1px solid",
		}, ":"),
	}

	config.Template, err = template.New("drwaio").Funcs(DrawioFuncMap).Parse(DrawioTEmplate)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func GenerateDrawio(config DrawioConfig) (File, error) {
	rowHeight := 16
	entities := []entity{}
	links := []drawioLink{}
	tableIdMap := map[string]string{}
	lineInTotal := map[string]float64{}
	lineOutTotal := map[string]float64{}
	lineInCount := map[string]float64{}
	lineOutCount := map[string]float64{}
	y := 0

	for i, table := range config.Tables {
		height := rowHeight*len(table.Columns) + 20
		entities = append(entities, entity{
			X:           0,
			Y:           y,
			Width:       350,
			Height:      height,
			HeaderStyle: config.HeaderStyle,
			TableStyle:  config.TableStyle,
			CellStyle:   config.CellStyle,
			Table:       table,
		})
		y += height

		tableId := fmt.Sprintf("%v-%v", config.CellId, i)
		tableIdMap[table.Name] = tableId
		for _, col := range table.Columns {
			if col.ForeignTable == nil {
				continue
			}
			lineOutTotal[table.Name] += 1
			lineInTotal[col.ForeignTable.Name] += 1
		}
	}

	for k, v := range lineOutTotal {
		lineOutTotal[k] = v + 1
	}

	for k, v := range lineInTotal {
		lineInTotal[k] = v + 1
	}

	for _, table := range config.Tables {
		for _, col := range table.Columns {
			if col.ForeignTable == nil {
				continue
			}

			lineOutCount[table.Name] += 1
			lineInCount[col.ForeignTable.Name] += 1
			entryPosition := fmt.Sprintf("entryX=%v;entryY=%v;", lineInCount[col.ForeignTable.Name]/lineInTotal[col.ForeignTable.Name], 1)
			exitPosition := fmt.Sprintf("exitX=%v;exitY=%v;", lineOutCount[table.Name]/lineOutTotal[table.Name], 0)

			link := drawioLink{
				Id:            fmt.Sprintf("%v-1", randId()),
				LineStyle:     config.LinkStyle + entryPosition + exitPosition,
				Source:        tableIdMap[table.Name],
				Target:        tableIdMap[col.ForeignTable.Name],
				LabelStyle:    config.EdgeLabelStyle,
				LabelId:       randId(),
				ForeignColumn: col.ForeignColumn.Name,
				RefColumn:     col.Name,
			}
			links = append(links, link)
		}
	}

	ctx := drawioContext{}
	ctx.DrawioConfig = config
	ctx.Entities = entities
	ctx.Links = links
	ctx.ModTime = time.Now().Format("2006-01-02T15:04:05.999Z")

	file, err := generateFile(config.Template, config.ExportPath, ctx)
	if err != nil {
		return File{}, err
	}

	return file, nil
}

func randId() string {
	r := make([]byte, 15)
	if _, err := rand.Read(r); err != nil {
		log.Fatal(err)
	}

	return base64.StdEncoding.EncodeToString(r)
}

func getEntity(e entity) string {
	buf := bytes.NewBuffer([]byte{})
	if err := entityTemplate.Execute(buf, e); err != nil {
		log.Fatal(err)
	}

	replacements := map[string]string{
		"&":  "&amp;",
		"<":  "&lt;",
		">":  "&gt;",
		`"`:  "&quot;",
		"'":  "&apos;",
		"\n": "",
	}
	return regexp.MustCompile(`&|<|>|"|'|\n`).ReplaceAllStringFunc(buf.String(), func(noe string) string {
		return replacements[noe]
	})
}

func isCompositePrimaryKey(table *Table) bool {
	var pKeyCount int
	for _, col := range table.Columns {
		if col.Attribute.IsPrimaryKey() {
			pKeyCount += 1
		}
	}
	return pKeyCount > 1
}

func getAllFields(table *Table) string {
	columnNames := []string{}
	for _, c := range table.Columns {
		entityName := strcase.ToLowerCamel(c.Name)
		columnNames = append(columnNames, entityName)
	}
	return strings.Join(columnNames, ",")
}

func getPkFields(table *Table) string {
	columnNames := []string{}
	for _, c := range table.Columns {
		if !c.Attribute.IsPrimaryKey() {
			continue
		}
		entityName := strcase.ToLowerCamel(c.Name)
		columnNames = append(columnNames, entityName)
	}
	return strings.Join(columnNames, ",")
}

func getDefaultValueFromAttribute(attr AttributeMap) string {
	if attr, ok := attr[ast.ConstraintTypeDefault]; ok {
		return getDefaultValue(attr)
	}
	return ""
}

func getDefaultValue(expr *ast.ColumnDefault) (value string) {
	return fmt.Sprintf("%v", expr)
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

func toSqlType(datatype element.Datatype) (name string) {
	switch datatype.DataDef() {
	case element.DataDefChar:
		realType := datatype.(*element.Char)
		if realType.Size == nil {
			name = "CHAR"
		} else {
			if realType.IsByteSize {
				name = fmt.Sprintf("CHAR(%v BYTE)", *realType.Size)
			} else if realType.IsCharSize {
				name = fmt.Sprintf("CHAR(%v CHAR)", *realType.Size)
			} else {
				name = fmt.Sprintf("CHAR(%v)", *realType.Size)
			}
		}
	case element.DataDefVarchar2:
		realType := datatype.(*element.Varchar2)
		if realType.Size == nil {
			name = "VARCHAR2"
		} else {
			if realType.IsByteSize {
				name = fmt.Sprintf("VARCHAR2(%v BYTE)", *realType.Size)
			} else if realType.IsCharSize {
				name = fmt.Sprintf("VARCHAR2(%v CHAR)", *realType.Size)
			} else {
				name = fmt.Sprintf("VARCHAR2(%v)", *realType.Size)
			}
		}
	case element.DataDefCharacterVarying:
		realType := datatype.(*element.Varchar2)
		if realType.Size == nil {
			name = "CHARRACTER VARYING"
		} else {
			name = fmt.Sprintf("CHARRACTER VARYING(%v)", *realType.Size)
		}
	case element.DataDefCharVarying:
		realType := datatype.(*element.Varchar2)
		if realType.Size == nil {
			name = "CHAR VARYING"
		} else {
			name = fmt.Sprintf("CHAR VARYING(%v)", *realType.Size)
		}
	case element.DataDefVarchar:
		realType := datatype.(*element.Varchar2)
		if realType.Size == nil {
			name = "VARCHAR"
		} else {
			name = fmt.Sprintf("VARCHAR(%v)", *realType.Size)
		}
	case element.DataDefNChar, element.DataDefNationalCharacter, element.DataDefNationalChar:
		realType := datatype.(*element.NChar)
		if realType.Size == nil {
			name = "NCHAR"
		} else {
			name = fmt.Sprintf("NCHAR(%v)", *realType.Size)
		}
	case element.DataDefNVarChar2, element.DataDefNCharVarying, element.DataDefNationalCharacterVarying, element.DataDefNationalCharVarying:
		realType := datatype.(*element.NVarchar2)
		if realType.Size == nil {
			name = "NVARCHAR2"
		} else {
			name = fmt.Sprintf("NVARCHAR2(%v)", *realType.Size)
		}
	case element.DataDefCharacter:
		realType := datatype.(*element.Char)
		if realType.Size == nil {
			name = "CHAR"
		} else {
			if realType.IsByteSize {
				name = fmt.Sprintf("CHAR(%v BYTE)", *realType.Size)
			} else if realType.IsCharSize {
				name = fmt.Sprintf("CHAR(%v CHAR)", *realType.Size)
			} else {
				name = fmt.Sprintf("CHAR(%v)", *realType.Size)
			}
		}
	case element.DataDefInteger:
		name = "INTEGER"
	case element.DataDefInt:
		name = "INT"
	case element.DataDefSmallInt:
		name = "SMALLINT"
	case element.DataDefLong:
		name = "LONG"
	case element.DataDefLongRaw:
		name = "LONG RAW"
	case element.DataDefFloat:
		name = formatFloatType("FLOAT", datatype.(*element.Float))
	case element.DataDefReal:
		name = "REAL"
	case element.DataDefBinaryFloat:
		name = "BINARYFLOAT"
	case element.DataDefBinaryDouble:
		name = "BINARYDOUBLE"
	case element.DataDefDoublePrecision:
		name = "Double"
	case element.DataDefDecimal:
		name = formatNumberType("DECIMAL", datatype.(*element.Number))
	case element.DataDefDec:
		name = formatNumberType("DEC", datatype.(*element.Number))
	case element.DataDefNumeric:
		name = formatNumberType("NUMERIC", datatype.(*element.Number))
	case element.DataDefNumber:
		name = formatNumberType("NUMBER", datatype.(*element.Number))
	case element.DataDefDate:
		name = "DATE"
	case element.DataDefTimestamp:
		realType := datatype.(*element.Timestamp)
		if realType.WithTimeZone {
			if realType.FractionalSecondsPrecision == nil {
				name = "TIMESTAMP WITH TIME ZONE"
			} else {
				name = fmt.Sprintf("TIMESTAMP(%v) WITH TIME ZONE", *realType.FractionalSecondsPrecision)
			}
		} else if realType.WithLocalTimeZone {
			if realType.FractionalSecondsPrecision == nil {
				name = "TIMESTAMP WITH LOCAL TIME ZONE"
			} else {
				name = fmt.Sprintf("TIMESTAMP(%v) WITH LOCAL TIME ZONE", *realType.FractionalSecondsPrecision)
			}
		} else {
			if realType.FractionalSecondsPrecision == nil {
				name = "TIMESTAMP"
			} else {
				name = fmt.Sprintf("TIMESTAMP(%v)", *realType.FractionalSecondsPrecision)
			}
		}
	case element.DataDefRowId:
		name = "ROWID"
	case element.DataDefURowId:
		realType := datatype.(*element.URowId)
		if realType.Size == nil {
			name = "UROWID"
		} else {
			name = fmt.Sprintf("UROWID(%v)", *realType.Size)
		}
	case element.DataDefBlob:
		name = "BLOB"
	case element.DataDefRaw:
		realType := datatype.(*element.Raw)
		if realType.Size == nil {
			name = "RAW"
		} else {
			name = fmt.Sprintf("RAW(%v)", *realType.Size)
		}
	case element.DataDefBFile:
		name = "BFILE"
	case element.DataDefClob:
		name = "CLOB"
	case element.DataDefNClob:
		name = "NCLOB"
	case element.DataDefIntervalYear:
		realType := datatype.(*element.IntervalYear)
		if realType.Precision == nil {
			name = "INTERVAL YEAR TO MONTH"
		} else {
			name = fmt.Sprintf("INTERVAL YEAR(%v) TO MONTH", *realType.Precision)
		}
	case element.DataDefIntervalDay:
		realType := datatype.(*element.IntervalDay)
		if realType.Precision == nil {
			name = "INTERVAL DAY TO SECOND"
		} else {
			name = fmt.Sprintf("INTERVAL DAY(%v) TO SECOND", *realType.Precision)
		}
	case element.DataDefXMLType:
		name = "XMLTYPE"
	default:
		name = "UnSupport"
	}
	return
}

func formatNumberType(typeName string, realType *element.Number) (name string) {
	if realType.Precision == nil {
		name = typeName
	} else if realType.Scale == nil {
		if realType.Precision.IsAsterisk {
			name = fmt.Sprintf("%v(*)", typeName)
		} else {
			name = fmt.Sprintf("%v(%v)", typeName, realType.Precision.Number)
		}
	} else {
		if realType.Precision.IsAsterisk {
			name = fmt.Sprintf("%v(*,%v)", typeName, *realType.Scale)
		} else {
			name = fmt.Sprintf("%v(%v, %v)", typeName, realType.Precision.Number, *realType.Scale)
		}
	}
	return
}

func formatFloatType(typeName string, realType *element.Float) (name string) {
	if realType.Precision == nil {
		name = typeName
	} else {
		if realType.Precision.IsAsterisk {
			name = fmt.Sprintf("%v(*)", typeName)
		} else {
			name = fmt.Sprintf("%v(%v)", typeName, realType.Precision.Number)
		}
	}
	return
}
