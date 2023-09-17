package ddlcode

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/blastrain/vitess-sqlparser/tidbparser/ast"
	"github.com/blastrain/vitess-sqlparser/tidbparser/dependency/mysql"
	"github.com/blastrain/vitess-sqlparser/tidbparser/dependency/types"
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
	entityName := strcase.ToLowerCamel(config.Table.Name)
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
	gormTag.WriteString(col.Type.InfoSchemaStr())

	if isPrimaryKey(col.Attribute) {
		gormTag.WriteString(";primary_key")
	}
	isNotNull := false
	for o, expr := range col.Attribute {
		switch o {
		case ast.ColumnOptionPrimaryKey:
			if !isPrimaryKey(col.Attribute) {
				gormTag.WriteString(";primary_key")
			}
		case ast.ColumnOptionNotNull:
			isNotNull = true
		case ast.ColumnOptionAutoIncrement:
			gormTag.WriteString(";AUTO_INCREMENT")
		case ast.ColumnOptionDefaultValue:
			if value := getDefaultValue(expr); value != "" {
				gormTag.WriteString(";default:")
				gormTag.WriteString(value)
			}
		case ast.ColumnOptionUniqKey:
			gormTag.WriteString(";unique")
		case ast.ColumnOptionNull:
			// gormTag.WriteString(";NULL")
			// canNull = true
		case ast.ColumnOptionOnUpdate: // For Timestamp and Datetime only.
		case ast.ColumnOptionFulltext:
		case ast.ColumnOptionComment:
			// field.Comment = expr.GetDatum().GetString()
		default:
			//return "", nil, errors.Errorf(" unsupport option %d\n", o.Tp)
		}
	}
	if !isPrimaryKey(col.Attribute) && isNotNull {
		gormTag.WriteString(";NOT NULL")
	}

	return fmt.Sprintf(`"gorm:%v"`, gormTag.String())
}

func toGoType(colTp *types.FieldType, attrs map[ast.ColumnOptionType]ast.ExprNode) (name string) {
	if _, canNull := attrs[ast.ColumnOptionNull]; canNull {
		switch colTp.Tp {
		case mysql.TypeTiny, mysql.TypeShort, mysql.TypeInt24, mysql.TypeLong:
			name = "sql.NullInt32"
		case mysql.TypeLonglong:
			name = "sql.NullInt64"
		case mysql.TypeFloat, mysql.TypeDouble:
			name = "sql.NullFloat64"
		case mysql.TypeString, mysql.TypeVarchar, mysql.TypeVarString,
			mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob:
			name = "sql.NullString" //nolint
		case mysql.TypeTimestamp, mysql.TypeDatetime, mysql.TypeDate:
			name = "sql.NullTime"
		case mysql.TypeDecimal, mysql.TypeNewDecimal:
			name = "sql.NullString" //nolint
		case mysql.TypeJSON:
			name = "sql.NullString" //nolint
		default:
			return "UnSupport"
		}
	} else {
		switch colTp.Tp {
		case mysql.TypeTiny, mysql.TypeShort, mysql.TypeInt24, mysql.TypeLong:
			if mysql.HasUnsignedFlag(colTp.Flag) {
				name = "uint"
			} else {
				name = "int"
			}
		case mysql.TypeLonglong:
			if mysql.HasUnsignedFlag(colTp.Flag) {
				name = "uint64"
			} else {
				name = "int64" //nolint
			}
		case mysql.TypeFloat, mysql.TypeDouble:
			name = "float64"
		case mysql.TypeString, mysql.TypeVarchar, mysql.TypeVarString,
			mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob:
			name = "string"
		case mysql.TypeTimestamp, mysql.TypeDatetime, mysql.TypeDate:
			name = "time.Time"
		case mysql.TypeDecimal, mysql.TypeNewDecimal:
			name = "string"
		case mysql.TypeJSON:
			name = "string"
		default:
			return "UnSupport"
		}
		// if style == NullInPointer {
		// 	name = "*" + name
		// }
	}
	return
}
