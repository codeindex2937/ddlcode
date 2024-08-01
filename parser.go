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
	createStmts := getCreateTableStatements(stmts)
	for _, createStmt := range createStmts {
		table := translateTable(createStmt)
		tableMap[table.Name] = table
	}

	for _, createStmt := range createStmts {
		table := tableMap[createStmt.TableName.Table.Value]
		for _, spec := range getReferenceConstraints(createStmt.RelTable.TableStructs) {
			switch spec.InlineConstraint.Type {
			case ast.ConstraintTypeReferences:
				refTable := tableMap[spec.Reference.Table.Table.Value]
				assignRefColumns(table, refTable, spec)
			}
		}
	}

	for _, alterStmt := range getAlterTableStatements(stmts) {
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

	tables := []*Table{}
	for _, t := range tableMap {
		tables = append(tables, t)
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

func getCreateTableStatements(stmts []ast.Node) []*ast.CreateTableStmt {
	createTableStmts := []*ast.CreateTableStmt{}
	for _, s := range stmts {
		switch stmt := s.(type) {
		case *ast.CreateTableStmt:
			createTableStmts = append(createTableStmts, stmt)
		}
	}
	return createTableStmts
}

func getAlterTableStatements(stmts []ast.Node) []*ast.AlterTableStmt {
	alterTableStmts := []*ast.AlterTableStmt{}
	for _, s := range stmts {
		switch stmt := s.(type) {
		case *ast.AlterTableStmt:
			alterTableStmts = append(alterTableStmts, stmt)
		}
	}
	return alterTableStmts
}

func getPkConstraints(stmts []ast.TableStructDef) (constraints []*ast.OutOfLineConstraint) {
	for _, t := range stmts {
		switch constraint := t.(type) {
		case *ast.OutOfLineConstraint:
			if constraint.Type == ast.ConstraintTypePK {
				constraints = append(constraints, constraint)
			}
		}
	}
	return
}

func getReferenceConstraints(stmts []ast.TableStructDef) (constraints []*ast.OutOfLineConstraint) {
	for _, t := range stmts {
		switch constraint := t.(type) {
		case *ast.OutOfLineConstraint:
			if constraint.Type == ast.ConstraintTypeReferences {
				constraints = append(constraints, constraint)
			}
		}
	}
	return
}

func getColumnDefs(stmts []ast.TableStructDef) (defs []*ast.ColumnDef) {
	for _, t := range stmts {
		switch constraint := t.(type) {
		case *ast.ColumnDef:
			defs = append(defs, constraint)
		}
	}
	return
}

func getAddConstraint(stmt *ast.AlterTableStmt) (clauses []*ast.AddConstraintClause) {
	for _, clause := range stmt.AlterTableClauses {
		switch constraint := clause.(type) {
		case *ast.AddConstraintClause:
			clauses = append(clauses, constraint)
		}
	}
	return
}
