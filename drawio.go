package ddlcode

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/blastrain/vitess-sqlparser/tidbparser/ast"
	"github.com/blastrain/vitess-sqlparser/tidbparser/dependency/types"
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
	ExportPath  string
	CellId      string
	EntityStyle string
	Width       int
	Height      int
	Tables      []*Table
	Template    *template.Template
	HeaderStyle string
	TableStyle  string
	CellStyle   string
}

type drawioContext struct {
	DrawioConfig
	ModTime  string
	Entities []entity
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
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
`

var entityFuncMap = template.FuncMap{
	"IsCompositePrimaryKey": isCompositePrimaryKey,
	"IsNotNull":             isNotNull,
	"IsPrimaryKey":          isPrimaryKey,
	"IsAutoIncrement":       isAutoIncrement,
	"IsUnique":              isUnique,
	"GetDefaultValue":       getDefaultValueFromAttribute,
}

var entityTemplate, _ = template.New("drawioEntity").Funcs(entityFuncMap).Parse(
	`<div style="{{.HeaderStyle}}">{{ .Table.Name }}</div>
<table style="{{.TableStyle}}">
{{- $ctx := . -}}
{{- range .Table.Columns }}
<tr>
<td style="{{ $ctx.CellStyle }}">{{ .Name }}</td>
<td style="{{ $ctx.CellStyle }}">{{ .Type }}</td>
<td style="{{ $ctx.CellStyle }}">{{ if IsNotNull .Attribute }}NN{{else}}-{{ end }}</td>
<td style="{{ $ctx.CellStyle }}">{{ if IsPrimaryKey .Attribute }}PK{{else}}-{{ end }}</td>
<td style="{{ $ctx.CellStyle }}">{{ if IsAutoIncrement .Attribute }}AI{{else}}-{{ end }}</td>
<td style="{{ $ctx.CellStyle }}">{{ if IsUnique .Attribute }}U{{else}}-{{ end }}</td>
<td style="{{ $ctx.CellStyle }}">{{ GetDefaultValue .Attribute }}</td>
</tr>
{{- end }}
</table>
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
	y := 0
	for _, table := range config.Tables {
		height := rowHeight*len(table.Columns) + 20
		entities = append(entities, entity{
			X:           0,
			Y:           y,
			Width:       180,
			Height:      height,
			HeaderStyle: config.HeaderStyle,
			TableStyle:  config.TableStyle,
			CellStyle:   config.CellStyle,
			Table:       table,
		})
		y += height
	}
	ctx := drawioContext{}
	ctx.DrawioConfig = config
	ctx.Entities = entities
	ctx.ModTime = time.Now().Format("2006-01-02T15:04:05.999Z")

	file, err := generateFile(config.Template, config.ExportPath, ctx)
	if err != nil {
		return File{}, err
	}

	return file, nil
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
		if isPrimaryKey(col.Attribute) {
			pKeyCount += 1
		}
	}
	return pKeyCount > 1
}

func isPrimaryKey(attr map[ast.ColumnOptionType]ast.ExprNode) bool {
	if _, ok := attr[ast.ColumnOptionPrimaryKey]; ok {
		return true
	}
	return false
}

func isNotNull(attr map[ast.ColumnOptionType]ast.ExprNode) bool {
	if _, ok := attr[ast.ColumnOptionNotNull]; ok {
		return true
	}
	return false
}

func isAutoIncrement(attr map[ast.ColumnOptionType]ast.ExprNode) bool {
	if _, ok := attr[ast.ColumnOptionAutoIncrement]; ok {
		return true
	}
	return false
}

func isUnique(attr map[ast.ColumnOptionType]ast.ExprNode) bool {
	if _, ok := attr[ast.ColumnOptionUniqKey]; ok {
		return true
	}
	return false
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
		if !isPrimaryKey(c.Attribute) {
			continue
		}
		entityName := strcase.ToLowerCamel(c.Name)
		columnNames = append(columnNames, entityName)
	}
	return strings.Join(columnNames, ",")
}

func getDefaultValueFromAttribute(attr map[ast.ColumnOptionType]ast.ExprNode) string {
	if attr, ok := attr[ast.ColumnOptionDefaultValue]; ok {
		return getDefaultValue(attr)
	}
	return ""
}

func getDefaultValue(expr ast.ExprNode) (value string) {
	if expr.GetDatum().Kind() != types.KindNull {
		value = fmt.Sprintf("%v", expr.GetDatum().GetValue())
	} else if expr.GetFlag() != ast.FlagConstant {
		if expr.GetFlag() == ast.FlagHasFunc {
			if funcExpr, ok := expr.(*ast.FuncCallExpr); ok {
				value = funcExpr.FnName.O
			}
		}
	}
	return
}

func join(style map[string]string, assignChar string) string {
	sb := strings.Builder{}
	for k, v := range style {
		sb.WriteString(fmt.Sprintf("%v%v%v;", k, assignChar, v))
	}
	return sb.String()
}

// func toSqlType(colTp *types.FieldType) (name string) {
// 	switch colTp.Tp {
// 	case mysql.TypeTiny:
// 		name = "TINYINT"
// 	case mysql.TypeShort:
// 		name = "SMALLINT"
// 	case mysql.TypeInt24:
// 		name = "MEDIUMINT"
// 	case mysql.TypeLong:
// 		name = "INT"
// 	case mysql.TypeLonglong:
// 		name = "BIGINT"
// 	case mysql.TypeFloat:
// 		name = "FLOAT"
// 	case mysql.TypeDouble:
// 		name = "DOUBLE"
// 	case mysql.TypeString:
// 		name = "CHAR"
// 	case mysql.TypeVarchar:
// 		name = "VARCHAR"
// 	case mysql.TypeVarString:
// 		name = "TEXT"
// 	case mysql.TypeBlob:
// 		name = "BLOB"
// 	case mysql.TypeTinyBlob:
// 		name = "TINYBLOB"
// 	case mysql.TypeMediumBlob:
// 		name = "MEDIUMBLOB"
// 	case mysql.TypeLongBlob:
// 		name = "LONGBLOB"
// 	case mysql.TypeTimestamp:
// 		name = "TIMESTAMP"
// 	case mysql.TypeDatetime:
// 		name = "DATETIME"
// 	case mysql.TypeDate:
// 		name = "DATE"
// 	case mysql.TypeDecimal:
// 		name = "DECIMAL"
// 	case mysql.TypeNewDecimal:
// 		name = "NUMERIC"
// 	case mysql.TypeJSON:
// 		name = "JSON"
// 	case mysql.TypeNull:
// 		name = "NULL"
// 	case mysql.TypeYear:
// 		name = "YEAR"
// 	case mysql.TypeBit:
// 		name = "BIT"
// 	case mysql.TypeSet:
// 		name = "SET"
// 	case mysql.TypeGeometry:
// 		name = "GEOMETRY"
// 	default:
// 		return "UnSupport"
// 	}
// 	return
// }
