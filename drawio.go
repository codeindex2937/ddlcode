package ddlcode

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/codeindex2937/ddlcode/drawio"
	"github.com/codeindex2937/ddlcode/html"
	"github.com/codeindex2937/ddlcode/toposort"
	"github.com/codeindex2937/oracle-sql-parser/ast"
	"github.com/codeindex2937/oracle-sql-parser/ast/element"
	"github.com/iancoleman/strcase"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var rowHeight = 18
var titleHeight = 20

type DrawioConfig struct {
	ExportPath     string
	CellId         string
	EntityStyle    map[string]string
	Width          int
	Height         int
	Tables         []*Table
	HeaderStyle    map[string]string
	TableStyle     map[string]string
	LinkStyle      map[string]string
	EdgeLabelStyle map[string]string
	CellStyle      map[string]string
}

type position struct {
	x int
	y int
}

func GetDefaultDrawioConfig() DrawioConfig {
	var err error
	buf := make([]byte, 8)

	_, err = rand.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	config := DrawioConfig{
		CellId: hex.EncodeToString(buf),
		EntityStyle: map[string]string{
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
		},
		HeaderStyle: map[string]string{
			"box-sizing": "border-box",
			"width":      "100%",
			"background": "#e4e4e4",
			"padding":    "2px",
			"color":      "black",
		},
		TableStyle: map[string]string{
			"width":           "100%",
			"font-size":       "1em",
			"background":      "DimGray",
			"border-collapse": "collapse",
		},
		LinkStyle: map[string]string{
			"edgeStyle":      "entityRelationEdgeStyle",
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
		},
		EdgeLabelStyle: map[string]string{
			"edgeLabel":     "",
			"html":          "1",
			"align":         "center",
			"verticalAlign": "middle",
			"resizable":     "0",
			"points":        "[]",
		},
		CellStyle: map[string]string{
			"border": "1px solid",
		},
	}

	return config
}

func GenerateDrawio(config DrawioConfig) (File, error) {
	var err error
	tableIdMap := map[string]string{}

	f := drawio.NewFile(config.Width, config.Height)
	parent := f.Diagram.MxGraphModel.Cells[1].(*drawio.Shape)
	tableWidth := 350

	positionMap := getPositions(config.Tables, func(rowNum int) int { return rowHeight*rowNum + titleHeight }, tableWidth+200)

	for i, table := range config.Tables {
		tableId := fmt.Sprintf("%v-%v", config.CellId, i)
		height := rowHeight*len(table.Columns) + titleHeight
		position := positionMap[table.Name]
		entity := drawio.NewShape(tableId, float64(position.x), float64(position.y), float64(tableWidth), float64(height), config.EntityStyle)
		entity.Value = getEntityBody(config, table)

		f.Diagram.MxGraphModel.AddCells(entity)

		tableIdMap[table.Name] = tableId
	}

	linkStyle := map[string]string{}
	maps.Copy(linkStyle, config.LinkStyle)

	for _, table := range config.Tables {
		for _, col := range table.Columns {
			if col.ForeignTable == nil {
				continue
			}

			linkStyle["entryX"] = "0"
			linkStyle["entryY"] = fmt.Sprintf("%v", getRelativeVerticalColumnPosition(col.ForeignTable, col.ForeignColumn.Name))
			linkStyle["exitX"] = "1"
			linkStyle["exitY"] = fmt.Sprintf("%v", getRelativeVerticalColumnPosition(table, col.Name))

			link := drawio.NewLine(
				parent.Id,
				tableIdMap[table.Name],
				tableIdMap[col.ForeignTable.Name],
				linkStyle)
			target, source := link.NewEdgeLabel(config.EdgeLabelStyle)
			target.Value = col.Name
			source.Value = col.ForeignColumn.Name
			f.Diagram.MxGraphModel.AddCells(link, target, source)
		}
	}

	if _, err := os.Stat(config.ExportPath); err == nil {
		mergePosition(config.ExportPath, f)
	}

	file := File{
		Path: config.ExportPath,
	}

	file.Content, err = xml.MarshalIndent(f, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return file, nil
}

func getPositions(tables []*Table, h func(int) int, w int) map[string]position {
	tableMap := map[string]*Table{}
	for _, table := range tables {
		tableMap[table.Name] = table
	}

	layers := sortIntoLayers(tableMap)
	isolatedNoes := []string{}
	isolatedTotalHeight := 0
	for key := range tableMap {
		if _, ok := layers[key]; !ok {
			isolatedNoes = append(isolatedNoes, key)
			isolatedTotalHeight += h(len(tableMap[key].Columns))
		}
	}
	slices.Sort(isolatedNoes)

	positionMap := map[string]position{}
	maxLayer := slices.Max(values(layers))
	avgIsolatedHeight := isolatedTotalHeight / (maxLayer + 1)
	currentHeight := 0
	maxIsolatedHeight := 0
	column := 0
	for _, key := range isolatedNoes {
		positionMap[key] = position{
			x: column * w,
			y: currentHeight,
		}
		currentHeight += h(len(tableMap[key].Columns))
		if maxIsolatedHeight < currentHeight {
			maxIsolatedHeight = currentHeight
		}

		if currentHeight >= avgIsolatedHeight {
			currentHeight = 0
			column += 1
			continue
		}
	}

	layerCurrentHeight := map[int]int{}
	for key := range tableMap {
		if slices.Contains(isolatedNoes, key) {
			continue
		}
		layer := (maxLayer - layers[key])
		positionMap[key] = position{
			x: layer * w,
			y: maxIsolatedHeight + layerCurrentHeight[layer],
		}
		layerCurrentHeight[layer] += h(len(tableMap[key].Columns))
	}
	return positionMap
}

func values[K comparable, V any](m map[K]V) []V {
	values := []V{}
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func sortIntoLayers(tables map[string]*Table) map[string]int {
	g := toposort.NewGraph[string]()
	for _, t := range tables {
		for _, c := range t.Columns {
			if c.ForeignTable == nil {
				continue
			}
			g.AddEdge(t.Name, c.ForeignTable.Name)
		}
	}

	linearOrders, _ := g.Sort()
	layerOrders := map[string]int{}
	for _, key := range reverseSlice(linearOrders) {
		neighborOrders := []int{}
		for _, k := range g.Neighbors(key) {
			neighborOrders = append(neighborOrders, layerOrders[k])
		}
		if len(neighborOrders) == 0 {
			layerOrders[key] = 0
			continue
		} else {
			layerOrders[key] = slices.Max(neighborOrders) + 1
		}
	}

	return layerOrders
}

func reverseSlice[V any](src []V) []V {
	s := len(src)
	t := make([]V, s)
	for i, v := range src {
		t[s-i-1] = v
	}
	return t
}

func getEntityBody(config DrawioConfig, table *Table) string {
	entity := html.Entity{}
	entity.Style = join(map[string]string{
		"display":        "flex",
		"flex-direction": "column",
		"height":         "100%",
	}, ":")
	entity.Title.Body = table.Name
	entity.Title.Style = join(config.HeaderStyle, ":") + "flex:0;"
	entity.Table.Style = join(config.TableStyle, ":") + "flex:1;"
	dataStyle := join(config.CellStyle, ":")

	for _, col := range table.Columns {
		notNull := "-"
		pk := "-"
		autoIncrement := "-"
		unique := "-"

		if col.Attribute.IsNotNull() {
			notNull = "NN"
		}
		if col.Attribute.IsPrimaryKey() {
			pk = "PK"
		}
		if col.Attribute.IsAutoIncrement() {
			autoIncrement = "AI"
		}
		if col.Attribute.IsUnique() {
			unique = "U"
		}

		row := html.TableRow{
			Data: []html.TableData{
				{Data: col.Name},
				{Data: toSqlType(col.Type)},
				{Data: notNull},
				{Data: pk},
				{Data: autoIncrement},
				{Data: unique},
				{Data: getDefaultValueFromAttribute(col.Attribute)},
			},
		}
		for i := range row.Data {
			row.Data[i].Style = dataStyle
		}
		entity.Table.Row = append(entity.Table.Row, row)
	}

	serialized, err := xml.Marshal(entity)
	if err != nil {
		log.Fatal(err)
	}

	return string(serialized)
}

func mergePosition(path string, f *drawio.MxFile) {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	src := drawio.MxFile{}
	if err := xml.Unmarshal(content, &src); err != nil {
		log.Fatal(err)
	}

	fileEntities := GetEntities(src.Diagram.MxGraphModel)
	entities := GetEntities(f.Diagram.MxGraphModel)
	for k, entity := range entities {
		fileEntity, ok := fileEntities[k]
		if !ok {
			continue
		}

		entity.Geometry = fileEntity.Geometry
	}
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

func GetEntities(m drawio.MxGraphModel) map[string]*drawio.Shape {
	shapes := make(map[string]*drawio.Shape)
	entityIdMap := make(map[string]html.Entity)

	for _, cell := range m.Cells {
		shape, ok := cell.(*drawio.Shape)
		if !ok {
			continue
		}
		if len(shape.Value) < 1 {
			continue
		}

		entity := html.Entity{}
		if err := xml.Unmarshal([]byte(shape.Value), &entity); err != nil {
			log.Fatal(err)
		}

		shapes[entity.Title.Body] = shape
		entityIdMap[shape.Id] = entity
	}

	return shapes
}

func getRelativeVerticalColumnPosition(table *Table, columnName string) float64 {
	index := slices.IndexFunc(table.Columns, func(column *Column) bool { return column.Name == columnName })
	entityHeight := rowHeight*len(table.Columns) + titleHeight
	return ((float64(index)+0.5)*float64(rowHeight) + float64(titleHeight)) / float64(entityHeight)
}

func getColumnNameFromRelativePosition(rows []html.TableRow, position float64) string {
	index := int((position*float64(len(rows)) + (position-1)*float64(titleHeight)/float64(rowHeight)))
	return rows[index].Data[0].Data
}

func getValueFromStyleString(styleString string, name string) string {
	for _, stmt := range strings.Split(styleString, ";") {
		index := strings.Index(stmt, "=")
		if stmt[:index] == name {
			return stmt[index+1:]
		}
	}
	return ""
}
