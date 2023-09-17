package ddlcode

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/blastrain/vitess-sqlparser/tidbparser/dependency/mysql"
	"github.com/blastrain/vitess-sqlparser/tidbparser/dependency/types"
	"github.com/iancoleman/strcase"
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
	"IsPrimaryKey":          isPrimaryKey,
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
			if isPrimaryKey(col.Attribute) {
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
    {{- if (IsPrimaryKey .Attribute) }}
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
    if (o == null || getClass() != o.getCleass()) {
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

var JavaPrimaryKeyTemplate = `package {{.Package}}.jpa;

import javax.persistence.*;
import java.util.Objects;
import java.io.Serializable;
{{ GetPkImportPaths .Table }}
public class {{ToCamel .Table.Name}}PK implements Serializable {
{{range .Table.Columns}}
{{- if IsPrimaryKey .Attribute}}
    @Id
    @Column(name = "{{.Name}}")
    private {{ToTypeName .Type }} {{ToLowerCamel .Name}};
{{end -}}
{{end}}
{{- range .Table.Columns}}
{{- if IsPrimaryKey .Attribute}}
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

public interface {{ToCamel .Table.Name}}Dao extends CrudRepository<{{ToCamel .Table.Name}}Entity,{{$pkType}}> {
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
		if !isPrimaryKey(c.Attribute) {
			continue
		}
		entityName := strcase.ToLowerCamel(c.Name)
		columnNames = append(columnNames, fmt.Sprintf("Objects.equals(this.%v,%v.%v)", entityName, otherName, entityName))
	}
	return strings.Join(columnNames, " && ")
}

func getJavaPkImportPaths(table *Table) (name string) {
	importPaths := []string{}
	for _, c := range table.Columns {
		if !isPrimaryKey(c.Attribute) {
			continue
		}
		javaType := toJavaType(c.Type)
		switch javaType {
		case "Date":
			importPaths = append(importPaths, "import java.util.Date;")
		case "BigDecimal":
			importPaths = append(importPaths, "importjava.math.BigDecimal;")
		}
	}
	return strings.Join(importPaths, "\n")
}

func getJavaImportPaths(table *Table) (name string) {
	importPaths := []string{}
	for _, c := range table.Columns {
		javaType := toJavaType(c.Type)
		switch javaType {
		case "Date":
			importPaths = append(importPaths, "import java.util.Date;\n")
		case "BigDecimal":
			importPaths = append(importPaths, "importjava.math.BigDecimal;\n")
		}
	}
	return strings.Join(importPaths, "")
}

func toJavaType(colTp *types.FieldType) (name string) {
	switch colTp.Tp {
	case mysql.TypeTiny, mysql.TypeShort, mysql.TypeInt24, mysql.TypeLong, mysql.TypeLonglong:
		name = "Long"
	case mysql.TypeFloat, mysql.TypeDouble:
		name = "Float64"
	case mysql.TypeString, mysql.TypeVarchar, mysql.TypeVarString,
		mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob:
		name = "String"
	case mysql.TypeTimestamp, mysql.TypeDatetime, mysql.TypeDate:
		name = "Date"
	case mysql.TypeDecimal, mysql.TypeNewDecimal:
		name = "BigDecimal"
	case mysql.TypeJSON:
		name = "String"
	default:
		return "UnSupport"
	}
	return
}
