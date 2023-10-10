package ddlcode

import (
	"github.com/codeindex2937/oracle-sql-parser/ast"
	"github.com/codeindex2937/oracle-sql-parser/ast/element"
	"golang.org/x/exp/slices"
)

type AttributeMap map[ast.ConstraintType]*ast.ColumnDefault
type NullStyle int
type Column struct {
	Name          string
	Type          element.Datatype
	Attribute     AttributeMap
	ForeignColumn *Column
	ForeignTable  *Table
}

type Table struct {
	Name    string
	Columns []*Column
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
