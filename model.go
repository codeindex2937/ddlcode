package ddlcode

import (
	"slices"

	"github.com/blastrain/vitess-sqlparser/tidbparser/ast"
	"github.com/blastrain/vitess-sqlparser/tidbparser/dependency/types"
)

type NullStyle int
type Column struct {
	Name          string
	Type          *types.FieldType
	Attribute     map[ast.ColumnOptionType]ast.ExprNode
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
