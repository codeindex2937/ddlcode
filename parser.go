package ddlcode

import (
	"log"

	parser "github.com/codeindex2937/oracle-sql-parser"
	"github.com/codeindex2937/oracle-sql-parser/ast"
)

func Parse(sql string) []*Table {
	stmts, err := parser.Parser(sql)
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

			for _, spec := range getForeignConstraints(ct.RelTable.TableStructs) {
				switch spec.InlineConstraint.Type {
				case ast.ConstraintTypeReferences:
					refTable := tableMap[spec.Reference.Table.Table.Value]
					assignRefColumns(table, refTable, spec)
				}
			}
		}

		if alterStmt, ok := stmt.(*ast.AlterTableStmt); ok {
			table := tableMap[alterStmt.TableName.Table.Value]
			for _, clause := range getAddConstraint(alterStmt) {
				for _, spec := range clause.Constraints {
					switch spec.InlineConstraint.Type {
					case ast.ConstraintTypeReferences:
						refTable := tableMap[spec.Reference.Table.Table.Value]
						assignRefColumns(table, refTable, spec)
					}
				}
			}
		}
	}

	return tables
}

func assignRefColumns(table, refTable *Table, spec *ast.OutOfLineConstraint) {
	for i, k := range spec.Columns {
		columnName := spec.Reference.Columns[i].Value
		c := table.getColumn(k.Value)
		c.ForeignTable = refTable
		c.ForeignColumn = refTable.getColumn(columnName)
	}
}

func translateTable(ct *ast.CreateTableStmt) *Table {
	isPrimaryKey := make(map[string]ast.Node)
	table := &Table{
		Name:    ct.TableName.Table.Value,
		Columns: []*Column{},
	}

	for _, constraint := range getPkConstraints(ct.RelTable.TableStructs) {
		for _, col := range constraint.Columns {
			isPrimaryKey[col.Value] = nil
		}
	}

	for _, def := range getColumnDefs(ct.RelTable.TableStructs) {
		opts := make(AttributeMap)
		for _, con := range def.Constraints {
			if con.Type == ast.ConstraintTypeDefault {
				opts[con.Type] = def.Default
			} else {
				opts[con.Type] = nil
			}
		}
		c := &Column{
			Name:      def.ColumnName.Value,
			Type:      def.Datatype,
			Attribute: opts,
		}
		if _, ok := isPrimaryKey[c.Name]; ok {
			c.Attribute[ast.ConstraintTypePK] = nil
		}
		table.Columns = append(table.Columns, c)
	}

	return table
}

func getPkConstraints(stmts []ast.TableStructDef) (constraints []*ast.OutOfLineConstraint) {
	for _, t := range stmts {
		if con, ok := t.(*ast.OutOfLineConstraint); ok {
			if con.Type == ast.ConstraintTypePK {
				constraints = append(constraints, con)
			}
		}
	}
	return
}

func getForeignConstraints(stmts []ast.TableStructDef) (constraints []*ast.OutOfLineConstraint) {
	for _, t := range stmts {
		if con, ok := t.(*ast.OutOfLineConstraint); ok {
			if con.Type == ast.ConstraintTypeReferences {
				constraints = append(constraints, con)
			}
		}
	}
	return
}

func getColumnDefs(stmts []ast.TableStructDef) (defs []*ast.ColumnDef) {
	for _, t := range stmts {
		if con, ok := t.(*ast.ColumnDef); ok {
			defs = append(defs, con)
		}
	}
	return
}

func getAddConstraint(stmt *ast.AlterTableStmt) (clauses []*ast.AddConstraintClause) {
	for _, clause := range stmt.AlterTableClauses {
		if constraint, ok := clause.(*ast.AddConstraintClause); ok {
			clauses = append(clauses, constraint)
		}
	}
	return
}
