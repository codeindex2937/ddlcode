package ddlcode

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/codeindex2937/oracle-sql-parser/ast/element"
	"github.com/iancoleman/strcase"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type JavaConfig struct {
	ExportDir          string
	Package            string
	Schema             string
	Table              *Table
	Template           *template.Template
	PrimaryKeyTemplate *template.Template
	DaoTemplate        *template.Template
}

var JavaFuncMap = template.FuncMap{
	"ToCamel":               strcase.ToCamel,
	"ToLowerCamel":          strcase.ToLowerCamel,
	"ToTypeName":            toJavaType,
	"IsCompositePrimaryKey": isCompositePrimaryKey,
	"GetAllFields":          getAllFields,
	"CompareFields":         compareJavaFields,
	"GetPkFields":           getPkFields,
	"ComparePkFields":       compareJavaPkFields,
	"GetImportPaths":        getJavaImportPaths,
	"GetPkImportPaths":      getJavaPkImportPaths,
	"GetPkType": func(table *Table) string {
		if isCompositePrimaryKey(table) {
			return strcase.ToCamel(table.Name) + "PK"
		}
		for _, col := range table.Columns {
			if col.Attribute.IsPrimaryKey() {
				return toJavaType(col.Type)
			}
		}
		return "Unknown"
	},
}

var JavaEntityTemplate = `package {{.Package}}.jpa;

import javax.persistence.*;
import java.util.Objects;
{{ GetImportPaths .Table }}

@Entity
@Table(name = "{{.Table.Name}}"{{if gt (len .Schema) 0}}, schema = "{{.Schema}}"{{end}})
{{- if IsCompositePrimaryKey .Table}}
@IdClass({{ToCamel .Table.Name}}PK.class)
{{- end}}
public class {{ToCamel .Table.Name}}Entity {
{{ $table := .Table}}
{{- range .Table.Columns}}
    {{- if (.Attribute.IsPrimaryKey) }}
    @Id
    {{- end}}
    @Column(name = "{{.Name}}")
    private {{ToTypeName .Type }} {{ToLowerCamel .Name}};
{{ end }}

{{- range .Table.Columns}}
    public {{ToTypeName .Type }} get{{ToCamel .Name}}() {
        return this.{{ToLowerCamel .Name}};
    }

    public void set{{ToCamel .Name}}({{ToTypeName .Type }} {{ToLowerCamel .Name}}) {
        this.{{ToLowerCamel .Name}} = {{ToLowerCamel .Name}};
    }
{{end}}

    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (o == null || getClass() != o.getClass()) {
            return false;
        }

        {{ToCamel .Table.Name}}Entity that = ({{ToCamel .Table.Name}}Entity)o;
        return {{CompareFields .Table "that"}};
    }

    @Override
    public int hashCode() {
        return Objects.hash({{GetAllFields .Table}});
    }
}
`

var JavaPrimaryKeyTemplate = `package {{.Package}}.jpa;

import javax.persistence.*;
import java.util.Objects;
import java.io.Serializable;
{{ GetPkImportPaths .Table }}

public class {{ToCamel .Table.Name}}PK implements Serializable {
{{range .Table.Columns}}
{{- if .Attribute.IsPrimaryKey}}
    @Id
    @Column(name = "{{.Name}}")
    private {{ToTypeName .Type }} {{ToLowerCamel .Name}};
{{end -}}
{{end}}
{{- range .Table.Columns}}
{{- if .Attribute.IsPrimaryKey}}
    public {{ToTypeName .Type }} get{{ToCamel .Name}}() {
        return this.{{ToLowerCamel .Name}};
    }

    public void set{{ToCamel .Name}}({{ToTypeName .Type }} {{ToLowerCamel .Name}}) {
        this.{{ToLowerCamel .Name}} = {{ToLowerCamel .Name}};
    }
{{end -}}
{{end}}

  public boolean equals(Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }

    {{ToCamel .Table.Name}}Entity that = ({{ToCamel .Table.Name}}Entity)o;
    return {{CompareFields .Table "that"}};
  }

  public int hashCode() {
    return Objects.hash({{GetAllFields .Table}});
  }
}
`

var JavaDaoTemplate = `package {{.Package}}.dao;

import javax.persistence.*;
import org.springframework.data.repository.CrudRepository;
import {{.Package}}.jpa.{{ToCamel .Table.Name}}Entity ;
{{- if IsCompositePrimaryKey .Table }}
import {{.Package}}.jpa.{{ToCamel .Table.Name}}PK;
{{- end}}
{{- $pkType := GetPkType .Table }}

public interface {{ToCamel .Table.Name}}Dao extends CrudRepository<{{ToCamel .Table.Name}}Entity, {{$pkType}}> {
}
`

func GetDefaultJavaConfig() JavaConfig {
	var err error
	config := JavaConfig{
		ExportDir: ".",
	}

	config.Template, err = template.New("javaEntity").Funcs(JavaFuncMap).Parse(JavaEntityTemplate)
	if err != nil {
		log.Fatal(err)
	}

	config.PrimaryKeyTemplate, err = template.New("javaPK").Funcs(JavaFuncMap).Parse(JavaPrimaryKeyTemplate)
	if err != nil {
		log.Fatal(err)
	}

	config.DaoTemplate, err = template.New("javaDao").Funcs(JavaFuncMap).Parse(JavaDaoTemplate)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func GenerateJava(config JavaConfig) ([]File, error) {
	files := []File{}
	entityName := strcase.ToCamel(config.Table.Name)
	entityFile, err := generateFile(config.Template, filepath.Join(config.ExportDir, "jpa", entityName+"Entity.java"), config)
	if err != nil {
		return nil, err
	}
	files = append(files, entityFile)

	if isCompositePrimaryKey(config.Table) {
		pkFile, err := generateFile(config.PrimaryKeyTemplate, filepath.Join(config.ExportDir, "jpa", entityName+"PK.java"), config)
		if err != nil {
			return nil, err
		}
		files = append(files, pkFile)
	}

	if config.DaoTemplate != nil {
		entityFile, err := generateFile(config.DaoTemplate, filepath.Join(config.ExportDir, "dao", entityName+"Dao.java"), config)
		if err != nil {
			return nil, err
		}
		files = append(files, entityFile)
	}

	return files, nil
}

func generateFile(tmpl *template.Template, path string, config any) (File, error) {
	f := File{
		Path: path,
	}

	buf := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(buf, config); err != nil {
		return f, err
	}

	f.Content = buf.Bytes()
	return f, nil
}

func compareJavaFields(table *Table, otherName string) string {
	columnNames := []string{}
	for _, c := range table.Columns {
		entityName := strcase.ToLowerCamel(c.Name)
		columnNames = append(columnNames, fmt.Sprintf("Objects.equals(this.%v,%v.%v)", entityName, otherName, entityName))
	}
	return strings.Join(columnNames, " && ")
}

func compareJavaPkFields(table *Table, otherName string) string {
	columnNames := []string{}
	for _, c := range table.Columns {
		if !c.Attribute.IsPrimaryKey() {
			continue
		}
		entityName := strcase.ToLowerCamel(c.Name)
		columnNames = append(columnNames, fmt.Sprintf("Objects.equals(this.%v,%v.%v)", entityName, otherName, entityName))
	}
	return strings.Join(columnNames, " && ")
}

func getJavaPkImportPaths(table *Table) (name string) {
	cols := []*Column{}
	for _, c := range table.Columns {
		if !c.Attribute.IsPrimaryKey() {
			continue
		}
		cols = append(cols, c)
	}
	importPaths := getImportPaths(cols)
	return strings.Join(importPaths, "\n")
}

func getJavaImportPaths(table *Table) (name string) {
	importPaths := getImportPaths(table.Columns)
	return strings.Join(importPaths, "\n")
}

func getImportPaths(cs []*Column) []string {
	importPaths := map[string]struct{}{}
	for _, c := range cs {
		javaType := toJavaType(c.Type)
		switch javaType {
		case "RowId":
			importPaths["import java.sql.RowId;"] = struct{}{}
		case "Blob":
			importPaths["import java.sql.Clob;"] = struct{}{}
		case "Clob":
			importPaths["import java.sql.Blob;"] = struct{}{}
		case "Timestamp":
			importPaths["import java.sql.Timestamp;"] = struct{}{}
		case "Date":
			importPaths["import java.util.Date;"] = struct{}{}
		case "BigDecimal":
			importPaths["import java.math.BigDecimal;"] = struct{}{}
		}
	}

	paths := maps.Keys(importPaths)
	slices.Sort(paths)
	return paths
}

func toJavaType(datatype element.Datatype) (name string) {
	switch datatype.DataDef() {
	case element.DataDefChar, element.DataDefVarchar2, element.DataDefNChar, element.DataDefNVarChar2, element.DataDefCharacter, element.DataDefCharacterVarying, element.DataDefCharVarying, element.DataDefNCharVarying, element.DataDefVarchar, element.DataDefNationalCharacter, element.DataDefNationalCharacterVarying, element.DataDefNationalChar, element.DataDefNationalCharVarying, element.DataDefXMLType:
		name = "String"
	case element.DataDefInteger, element.DataDefInt, element.DataDefSmallInt:
		name = "Integer"
	case element.DataDefLong, element.DataDefLongRaw:
		name = "Long"
	case element.DataDefFloat, element.DataDefReal, element.DataDefBinaryFloat:
		name = "Float"
	case element.DataDefBinaryDouble, element.DataDefDoublePrecision:
		name = "Double"
	case element.DataDefNumber, element.DataDefDecimal, element.DataDefDec, element.DataDefNumeric:
		if datatype.(*element.Number).Scale == nil || *datatype.(*element.Number).Scale == 0 {
			name = "Long"
		} else {
			name = "BigDecimal"
		}
	case element.DataDefDate:
		name = "Date"
	case element.DataDefTimestamp:
		name = "Timestamp"
	case element.DataDefRowId, element.DataDefURowId:
		name = "RowId"
	case element.DataDefBlob, element.DataDefRaw, element.DataDefBFile:
		name = "Blob"
	case element.DataDefClob, element.DataDefNClob:
		name = "Clob"
	case element.DataDefIntervalYear, element.DataDefIntervalDay:
		name = "UnSupport"
	default:
		name = "UnSupport"
	}
	return
}
