package ddlcode

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	parser "github.com/codeindex2937/oracle-sql-parser"
	"github.com/codeindex2937/oracle-sql-parser/ast"
	"github.com/codeindex2937/oracle-sql-parser/ast/element"
)

func Parse(sql string) Database {
	db := Database{
		DatabaseName: "oracle",
		Version:      "3.35.5",
		Views:        []any{},
		Columns:      []*Column{},
		Tables:       []*Table{},
	}

	stmts, err := parser.Parser(sql)
	if err != nil {
		log.Fatal(err)
	}

	tableMap := map[string]*Table{}
	createStmts := cast(stmts, castCreateTableStmt)
	for _, createStmt := range createStmts {
		table, pkInfo := translateTable(createStmt)
		tableMap[table.Table] = table

		if pkInfo.FieldCount > 0 {
			db.PkInfo = append(db.PkInfo, pkInfo)
		}
	}
	indexStmts := cast(stmts, castCreateIndexStmt)
	for _, indexStmt := range indexStmts {
		var indexSchema string
		indexTable := indexStmt.Index.TableName.Table.Value
		uniqueIndex := "false"
		if indexStmt.Type == "unique" {
			uniqueIndex = "true"
		}
		if indexStmt.IndexName.Schema != nil {
			indexSchema = indexStmt.IndexName.Schema.Value
		}
		for _, expr := range indexStmt.Index.IndexExprs {
			db.Indexes = append(db.Indexes, IndexInfo{
				Schema:      indexSchema,
				Table:       indexTable,
				Column:      expr.Column.Value,
				Cardinality: "",
				Direction:   expr.Direction,
				IndexType:   "B-TREE",
				Name:        indexStmt.IndexName.Index.Value,
				Size:        "",
				Unique:      uniqueIndex,
			})
		}
	}

	for _, createStmt := range createStmts {
		table := tableMap[createStmt.TableName.Table.Value]
		for _, spec := range cast(createStmt.RelTable.TableStructs, castRefConstraint) {
			switch spec.InlineConstraint.Type {
			case ast.ConstraintTypeReferences:
				refTable := tableMap[spec.Reference.Table.Table.Value]
				db.FkInfo = append(db.FkInfo, assignRefColumns(table, refTable, spec)...)
			}
		}
	}

	for _, alterStmt := range cast(stmts, castAlterTableStmt) {
		table := tableMap[alterStmt.TableName.Table.Value]
		for _, clause := range cast(alterStmt.AlterTableClauses, castAddConstraintStmt) {
			for _, spec := range clause.Constraints {
				switch spec.InlineConstraint.Type {
				case ast.ConstraintTypeReferences:
					refTable := tableMap[spec.Reference.Table.Table.Value]
					db.FkInfo = append(db.FkInfo, assignRefColumns(table, refTable, spec)...)
				}
			}
		}
	}

	for _, t := range tableMap {
		db.Tables = append(db.Tables, t)
		for _, c := range t.Columns {
			db.Columns = append(db.Columns, c)
		}
	}
	return db
}

func assignRefColumns(table, refTable *Table, spec *ast.OutOfLineConstraint) []FkInfo {
	fkInfos := []FkInfo{}

	fkDef := fmt.Sprintf("FOREIGN KEY (%v) REFERENCES %v(%v)%v%v",
		joinStr(mapping(spec.Columns, getColumnName)),
		refTable.Table,
		joinStr(mapping(spec.Reference.Columns, getColumnName)),
		refActionStr(spec.UpdateAction, " ON UPDATE"),
		refActionStr(spec.DeleteAction, " ON DELETE"),
	)

	for i, k := range spec.Columns {
		columnName := spec.Reference.Columns[i].Value
		c := table.getColumn(k.Value)
		c.ForeignTable = refTable
		c.ForeignColumn = refTable.getColumn(columnName)

		refColumn := refTable.getColumn(columnName)
		if refColumn == nil {
			log.Fatalf("unknown ref. column: %v.%v => %v.%v", table.Table, k.Value, refTable.Table, columnName)
		}

		fkInfos = append(fkInfos, FkInfo{
			Schema:          table.Schema,
			Table:           table.Table,
			Column:          c.Name,
			FkDef:           fkDef,
			ForeignKeyName:  spec.Name.Value,
			ReferenceTable:  refTable.Table,
			ReferenceColumn: refTable.getColumn(columnName).Name,
		})
	}

	return fkInfos
}

func refActionStr(v *ast.ReferenceOption, prefix string) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v %v", prefix, ReferenceOptionString(v))
}

func ReferenceOptionString(v *ast.ReferenceOption) string {
	switch v.Type {
	case ast.RefOptNoAction:
		return "NO ACTION"
	case ast.RefOptCascade:
		return "CASCADE"
	case ast.RefOptRestrict:
		return "RESTRICT"
	case ast.RefOptSetNull:
		return "SET NULL"
	case ast.RefOptSetDefault:
		return "DEFAULT"
	}
	return ""
}

func getColumnName(c *element.Identifier) string {
	return c.Value
}

func joinStr(segs []string) string {
	b := strings.Builder{}
	for i, s := range segs {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(s)
	}
	return b.String()
}

func mapping[T any](src []T, fn func(T) string) []string {
	result := []string{}
	for _, item := range src {
		result = append(result, fn(item))
	}
	return result
}

func cast[T any, U any](src []T, fn func(T) *U) []*U {
	result := []*U{}
	for _, item := range src {
		u := fn(item)
		if u != nil {
			result = append(result, u)
		}
	}
	return result
}

func castCreateTableStmt(v ast.Node) *ast.CreateTableStmt { r, _ := v.(*ast.CreateTableStmt); return r }
func castCreateIndexStmt(v ast.Node) *ast.CreateIndexStmt { r, _ := v.(*ast.CreateIndexStmt); return r }
func castAlterTableStmt(v ast.Node) *ast.AlterTableStmt   { r, _ := v.(*ast.AlterTableStmt); return r }
func castAddConstraintStmt(v ast.AlterTableClause) *ast.AddConstraintClause {
	r, _ := v.(*ast.AddConstraintClause)
	return r
}
func castColDefTableStmt(v ast.TableStructDef) *ast.ColumnDef { r, _ := v.(*ast.ColumnDef); return r }
func castPkConstraint(v ast.TableStructDef) *ast.OutOfLineConstraint {
	switch constraint := v.(type) {
	case *ast.OutOfLineConstraint:
		if constraint.Type == ast.ConstraintTypePK {
			return constraint
		}
	}
	return nil
}
func castRefConstraint(v ast.TableStructDef) *ast.OutOfLineConstraint {
	switch constraint := v.(type) {
	case *ast.OutOfLineConstraint:
		if constraint.Type == ast.ConstraintTypeReferences {
			return constraint
		}
	}
	return nil
}

func translateTable(ct *ast.CreateTableStmt) (*Table, PkInfo) {
	var pkInfo PkInfo
	isPrimaryKey := make(map[string]ast.Node)
	var schema string

	if ct.TableName.Schema != nil {
		schema = ct.TableName.Schema.Value
	}

	table := &Table{
		Schema:  schema,
		Table:   ct.TableName.Table.Value,
		Columns: []*Column{},
		Rows:    -1,
		Type:    "table",
	}

	for _, constraint := range cast(ct.RelTable.TableStructs, castPkConstraint) {
		colNames := joinStr(mapping(constraint.Columns, getColumnName))
		pkInfo = PkInfo{
			Schema:     schema,
			Table:      ct.TableName.Table.Value,
			FieldCount: len(constraint.Columns),
			PkColumn:   colNames,
			PkDef:      fmt.Sprintf("PRIMARY KEY (%v)", colNames),
		}

		for _, col := range constraint.Columns {
			isPrimaryKey[col.Value] = nil
		}
	}

	for i, def := range cast(ct.RelTable.TableStructs, castColDefTableStmt) {
		opts := make(AttributeMap)
		for _, con := range def.Constraints {
			if con.Type == ast.ConstraintTypeDefault {
				opts[con.Type] = def.Default
			} else {
				opts[con.Type] = nil
			}
		}
		c := &Column{
			Name:            def.ColumnName.Value,
			DataType:        def.Datatype,
			Type:            typeStr(def.Datatype.DataDef()),
			Attribute:       opts,
			OrdinalPosition: i,
			Schema:          schema,
			Table:           table.Table,
		}
		if def.Collation != nil {
			c.Collation = def.ColumnName.Value
		}
		if _, ok := opts[ast.ConstraintTypeDefault]; ok {
			c.Default = def.ColumnName.Value
		}
		if _, ok := opts[ast.ConstraintTypeNull]; ok {
			c.Nullable = "true"
		} else {
			c.Nullable = "false"
		}
		setCharacterMaximumLength(c, def.Datatype)
		setPrecision(c, def.Datatype)
		if _, ok := isPrimaryKey[c.Name]; ok {
			c.Attribute[ast.ConstraintTypePK] = nil
		}
		table.Columns = append(table.Columns, c)
	}

	return table, pkInfo
}

func setCharacterMaximumLength(c *Column, dataType element.Datatype) {
	switch implType := dataType.(type) {
	case *element.Char:
		if implType.Size != nil {
			c.CharacterMaximumLength = strconv.Itoa(*implType.Size)
		}
	case *element.Varchar2:
		if implType.Size != nil {
			c.CharacterMaximumLength = strconv.Itoa(*implType.Size)
		}
	case *element.NVarchar2:
		if implType.Size != nil {
			c.CharacterMaximumLength = strconv.Itoa(*implType.Size)
		}
	case *element.NChar:
		if implType.Size != nil {
			c.CharacterMaximumLength = strconv.Itoa(*implType.Size)
		}
	}
}

func setPrecision(c *Column, dataType element.Datatype) {
	switch implType := dataType.(type) {
	case *element.IntervalYear:
		if implType.Precision != nil {
			c.Precision = strconv.Itoa(*implType.Precision)
		}
	case *element.IntervalDay:
		if implType.Precision != nil {
			c.Precision = strconv.Itoa(*implType.Precision)
		}
	case *element.Number:
		if implType.Precision != nil {
			if implType.Precision.IsAsterisk {
				c.Precision = "*"
			} else {
				c.Precision = strconv.Itoa(implType.Precision.Number)
			}
		}
	case *element.Float:
		if implType.Precision != nil {
			if implType.Precision.IsAsterisk {
				c.Precision = "*"
			} else {
				c.Precision = strconv.Itoa(implType.Precision.Number)
			}
		}
	}
}

func typeStr(v element.DataDef) string {
	switch v {
	case element.DataDefVarchar2:
		return "varchar"
	case element.DataDefNChar:
		return "char"
	case element.DataDefNVarChar2:
		return "varchar"
	case element.DataDefNumber:
		return "number"
	case element.DataDefFloat:
		return ""
	case element.DataDefBinaryFloat:
		return "float"
	case element.DataDefBinaryDouble:
		return "double"
	case element.DataDefLong:
		return "long"
	case element.DataDefLongRaw:
		return "long"
	case element.DataDefRaw:
		return "raw"
	case element.DataDefDate:
		return "date"
	case element.DataDefTimestamp:
		return "timestamp"
	case element.DataDefIntervalYear:
		return "interval_year"
	case element.DataDefIntervalDay:
		return "interval_day"
	case element.DataDefBlob:
		return "blob"
	case element.DataDefClob:
		return "clob"
	case element.DataDefNClob:
		return "clob"
	case element.DataDefBFile:
		return "bfile"
	case element.DataDefRowId:
		return "row_id"
	case element.DataDefURowId:
		return "urow_id"
	case element.DataDefCharacter:
		return "char"
	case element.DataDefCharacterVarying:
		return "vary"
	case element.DataDefCharVarying:
		return "char_vary"
	case element.DataDefNCharVarying:
		return "char_vary"
	case element.DataDefVarchar:
		return "varchar"
	case element.DataDefNationalCharacter:
		return "national_char"
	case element.DataDefNationalCharacterVarying:
		return "national_char_vary"
	case element.DataDefNationalChar:
		return "national_char"
	case element.DataDefNationalCharVarying:
		return "national_char_vary"
	case element.DataDefNumeric:
		return "numeric"
	case element.DataDefDecimal:
		return "decimal"
	case element.DataDefDec:
		return "dec"
	case element.DataDefInteger:
		return "integer"
	case element.DataDefInt:
		return "int"
	case element.DataDefSmallInt:
		return "smallint"
	case element.DataDefDoublePrecision:
		return "double_precision"
	case element.DataDefReal:
		return "real"
	case element.DataDefXMLType:
		return "xml"
	}
	return ""
}
