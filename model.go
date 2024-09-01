package ddlcode

import (
	"github.com/codeindex2937/oracle-sql-parser/ast"
	"github.com/codeindex2937/oracle-sql-parser/ast/element"
	"golang.org/x/exp/slices"
)

type AttributeMap map[ast.ConstraintType]*ast.ColumnDefault
type NullStyle int

type Column struct {
	CharacterMaximumLength string           `json:"character_maximum_length"`
	Collation              string           `json:"collation"`
	Default                string           `json:"default"`
	Name                   string           `json:"name"`
	Nullable               string           `json:"nullable"`
	OrdinalPosition        int              `json:"ordinal_position"`
	Precision              string           `json:"precision"`
	Schema                 string           `json:"schema"`
	Table                  string           `json:"table"`
	Type                   string           `json:"type"`
	DataType               element.Datatype `json:"-"`
	Attribute              AttributeMap     `json:"-"`
	ForeignColumn          *Column          `json:"-"`
	ForeignTable           *Table           `json:"-"`
}

type Table struct {
	Collation string    `json:"collation"`
	Engine    string    `json:"engine"`
	Rows      int       `json:"rows"`
	Schema    string    `json:"schema"`
	Table     string    `json:"table"`
	Type      string    `json:"type"`
	Columns   []*Column `json:"-"`
}

type PkInfo struct {
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	FieldCount int    `json:"field_count"`
	PkColumn   string `json:"pk_column"`
	PkDef      string `json:"pk_def"`
}

type FkInfo struct {
	Schema          string `json:"schema"`
	Table           string `json:"table"`
	Column          string `json:"column"`
	FkDef           string `json:"fk_def"`
	ForeignKeyName  string `json:"foreign_key_name"`
	ReferenceColumn string `json:"reference_column"`
	ReferenceTable  string `json:"reference_table"`
}

type IndexInfo struct {
	Schema      string `json:"schema"`
	Table       string `json:"table"`
	Column      string `json:"column"`
	Cardinality string `json:"cardinality"`
	Direction   string `json:"direction"`
	IndexType   string `json:"index_type"`
	Name        string `json:"name"`
	Size        string `json:"size"`
	Unique      string `json:"unique"`
}

type Database struct {
	Columns      []*Column   `json:"columns"`
	Tables       []*Table    `json:"tables"`
	Version      string      `json:"version"`
	Views        []any       `json:"views"`
	DatabaseName string      `json:"database_name"`
	PkInfo       []PkInfo    `json:"pk_info"`
	FkInfo       []FkInfo    `json:"fk_info"`
	Indexes      []IndexInfo `json:"indexes"`
}

func (t Table) getColumn(name string) *Column {
	index := slices.IndexFunc(t.Columns, func(c *Column) bool { return c.Name == name })
	if index < 0 {
		return nil
	}
	return t.Columns[index]
}

func (attr AttributeMap) IsPrimaryKey() bool {
	if _, ok := attr[ast.ConstraintTypePK]; ok {
		return true
	}
	return false
}

func (attr AttributeMap) IsNotNull() bool {
	if _, ok := attr[ast.ConstraintTypeNotNull]; ok {
		return true
	}
	return false
}

func (attr AttributeMap) IsAllowNull() bool {
	if _, ok := attr[ast.ConstraintTypeNull]; ok {
		return true
	}
	return false
}

func (attr AttributeMap) IsAutoIncrement() bool {
	// FIXME
	return false
}

func (attr AttributeMap) IsUnique() bool {
	if _, ok := attr[ast.ConstraintTypeUnique]; ok {
		return true
	}
	return false
}
