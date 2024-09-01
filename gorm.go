package ddlcode

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/codeindex2937/oracle-sql-parser/ast"
	"github.com/codeindex2937/oracle-sql-parser/ast/element"
	"github.com/iancoleman/strcase"
)

const (
	NullDisable NullStyle = iota
	NullInSql
	NullInPointer
)

type GormConfig struct {
	ExportDir string
	Package   string
	Table     *Table
	Template  *template.Template
}

var GormFuncMap = template.FuncMap{
	"ToCamel":      strcase.ToCamel,
	"ToLowerCamel": strcase.ToLowerCamel,
	"ToTypeName":   toGoType,
	"ToTags":       toTags,
}

var modelStructTmpl, _ = template.New("goFile").Funcs(GormFuncMap).Parse(`package {{.Package}}

import (
	"time"
	"gorm.io/gorm"
	"database/sql"
)

type {{ToCamel .Table.Name}} struct {
{{- range .Table.Columns}}
	{{ToLowerCamel .Name}} {{ToTypeName .Type .Attribute}} ` + "`{{ToTags .}}`" + `
{{- end}}
}`)

func GetDefaultGormConfig() GormConfig {
	config := GormConfig{
		ExportDir: ".",
		Template:  modelStructTmpl,
	}

	return config
}

func GenerateGorm(config GormConfig) ([]File, error) {
	files := []File{}
	entityName := strcase.ToLowerCamel(config.Table.Table)
	entityFile, err := generateFile(config.Template, filepath.Join(config.ExportDir, entityName+".go"), config)
	if err != nil {
		return nil, err
	}
	files = append(files, entityFile)

	return files, nil
}

func toTags(col Column) string {
	gormTag := strings.Builder{}
	gormTag.WriteString("column:")
	gormTag.WriteString(strcase.ToLowerCamel(col.Name))

	gormTag.WriteString(";type:")
	gormTag.WriteString(toGoType(col.DataType, col.Attribute))

	if col.Attribute.IsPrimaryKey() {
		gormTag.WriteString(";primary_key")
	}
	isNotNull := false
	for o, expr := range col.Attribute {
		switch o {
		case ast.ConstraintTypePK:
			if !col.Attribute.IsPrimaryKey() {
				gormTag.WriteString(";primary_key")
			}
		case ast.ConstraintTypeNotNull:
			isNotNull = true
		// case ast.ColumnOptionAutoIncrement:
		// 	gormTag.WriteString(";AUTO_INCREMENT")
		case ast.ConstraintTypeDefault:
			if value := getDefaultValue(expr); value != "" {
				gormTag.WriteString(";default:")
				gormTag.WriteString(value)
			}
		case ast.ConstraintTypeUnique:
			gormTag.WriteString(";unique")
		case ast.ConstraintTypeNull:
			// gormTag.WriteString(";NULL")
			// canNull = true
		// case ast.ColumnOptionOnUpdate: // For Timestamp and Datetime only.
		// case ast.ColumnOptionFulltext:
		// case ast.ColumnOptionComment:
		// field.Comment = expr.GetDatum().GetString()
		default:
			//return "", nil, errors.Errorf(" unsupport option %d\n", o.Tp)
		}
	}
	if !col.Attribute.IsPrimaryKey() && isNotNull {
		gormTag.WriteString(";NOT NULL")
	}

	return fmt.Sprintf(`gorm:"%v"`, gormTag.String())
}

func toGoType(datatype element.Datatype, attrs AttributeMap) (name string) {
	if attrs.IsAllowNull() {
		switch datatype.DataDef() {
		case element.DataDefInteger, element.DataDefInt, element.DataDefSmallInt:
			name = "sql.NullInt32"
		case element.DataDefLong, element.DataDefLongRaw:
			name = "sql.NullInt64"
		case element.DataDefFloat, element.DataDefReal, element.DataDefBinaryFloat, element.DataDefNumber, element.DataDefBinaryDouble, element.DataDefDoublePrecision:
			name = "sql.NullFloat64"
		case element.DataDefChar, element.DataDefVarchar2, element.DataDefNChar, element.DataDefNVarChar2, element.DataDefCharacter, element.DataDefCharacterVarying, element.DataDefCharVarying, element.DataDefNCharVarying, element.DataDefVarchar, element.DataDefNationalCharacter, element.DataDefNationalCharacterVarying, element.DataDefNationalChar, element.DataDefNationalCharVarying, element.DataDefXMLType:
			name = "sql.NullString" //nolint
		case element.DataDefDate, element.DataDefTimestamp:
			name = "sql.NullTime"
		case element.DataDefDecimal, element.DataDefDec, element.DataDefNumeric:
			name = "sql.NullString" //nolint
		default:
			return "UnSupport"
		}
	} else {
		switch datatype.DataDef() {
		case element.DataDefInteger, element.DataDefInt, element.DataDefSmallInt:
			name = "int"
		case element.DataDefLong, element.DataDefLongRaw:
			name = "int64" //nolint
		case element.DataDefFloat, element.DataDefReal, element.DataDefBinaryFloat, element.DataDefNumber, element.DataDefBinaryDouble, element.DataDefDoublePrecision:
			name = "float64"
		case element.DataDefChar, element.DataDefVarchar2, element.DataDefNChar, element.DataDefNVarChar2, element.DataDefCharacter, element.DataDefCharacterVarying, element.DataDefCharVarying, element.DataDefNCharVarying, element.DataDefVarchar, element.DataDefNationalCharacter, element.DataDefNationalCharacterVarying, element.DataDefNationalChar, element.DataDefNationalCharVarying, element.DataDefXMLType:
			name = "string"
		case element.DataDefDate, element.DataDefTimestamp:
			name = "time.Time"
		case element.DataDefDecimal, element.DataDefDec, element.DataDefNumeric:
			name = "string"
		case element.DataDefRowId, element.DataDefURowId:
			name = "java.sql.RowId"
		case element.DataDefBlob, element.DataDefRaw, element.DataDefBFile:
			name = "java.sql.Blob"
		case element.DataDefClob, element.DataDefNClob, element.DataDefIntervalYear, element.DataDefIntervalDay:
			name = "UnSupport"
		}
		// if style == NullInPointer {
		// 	name = "*" + name
		// }
	}
	return
}
