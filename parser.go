package ddlcode

import (
	"log"
	"regexp"

	"github.com/blastrain/vitess-sqlparser/tidbparser/ast"
	"github.com/blastrain/vitess-sqlparser/tidbparser/parser"
)

func Generalize(sql string) string {
	re := regexp.MustCompile(`(?i) TIMESTAMP WITH TIME ZONE`)
	sql = re.ReplaceAllString(sql, " TIMESTAMP")
	re = regexp.MustCompile(`(?i) NUMBER\(`)
	sql = re.ReplaceAllString(sql, ` NUMERIC(`)
	re = regexp.MustCompile(`(?i) VARCHAR2`)
	sql = re.ReplaceAllString(sql, ` VARCHAR`)
	re = regexp.MustCompile(`(?i) NVARCHAR2`)
	sql = re.ReplaceAllString(sql, ` VARCHAR`)
	return sql
}

func Parse(sql string) []*Table {
	stmts, err := parser.New().Parse(sql, "", "")
	if err != nil {
		log.Fatal(err)
	}

	tableMap := map[string]*Table{}
	tables := []*Table{}
	for _, stmt := range stmts {
		if ct, ok := stmt.(*ast.CreateTableStmt); ok {
			table := translateTable(ct)
			tables = append(tables, table)
			tableMap[table.Name] = table
		}

		if ct, ok := stmt.(*ast.AlterTableStmt); ok {
			table := tableMap[ct.Table.Name.String()]
			for _, spec := range ct.Specs {
				switch spec.Constraint.Tp {
				case ast.ConstraintForeignKey:
					for i, k := range spec.Constraint.Keys {
						refTable := tableMap[spec.Constraint.Refer.Table.Name.String()]
						columnName := spec.Constraint.Refer.IndexColNames[i].Column.OrigColName()
						c := table.getColumn(columnName)
						c.ForeignTable = refTable
						c.ForeignColumn = refTable.getColumn(k.Column.OrigColName())
					}
				}
			}
		}
	}

	return tables
}

func translateTable(ct *ast.CreateTableStmt) *Table {
	isPrimaryKey := make(map[string]ast.ExprNode)
	table := &Table{
		Name:    ct.Table.Name.String(),
		Columns: []*Column{},
	}

	for _, con := range ct.Constraints {
		if con.Tp == ast.ConstraintPrimaryKey {
			for _, k := range con.Keys {
				isPrimaryKey[k.Column.OrigColName()] = nil
			}
		}
	}

	for _, col := range ct.Cols {
		opts := make(map[ast.ColumnOptionType]ast.ExprNode)
		for _, opt := range col.Options {
			opts[opt.Tp] = opt.Expr
		}
		c := &Column{
			Name:      col.Name.OrigColName(),
			Type:      col.Tp,
			Attribute: opts,
		}
		if _, ok := isPrimaryKey[c.Name]; ok {
			c.Attribute[ast.ColumnOptionPrimaryKey] = nil
		}
		table.Columns = append(table.Columns, c)
	}

	return table
}
