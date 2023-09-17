package ddlcode

import (
	"github.com/blastrain/vitess-sqlparser/tidbparser/ast"
	"github.com/blastrain/vitess-sqlparser/tidbparser/dependency/types"
)

type NullStyle int
type Column struct {
	Name      string
	Type      *types.FieldType
	Attribute map[ast.ColumnOptionType]ast.ExprNode
}

type Table struct {
	Name    string
	Columns []Column
}
