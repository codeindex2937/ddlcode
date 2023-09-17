# ddlcode
convert DDL to code

## Reference
https://github.com/zhufuyi/gotool/tree/main/pkg/sql2code
https://github.com/ikaiguang/go-sqlparser

## Example Code
```go

func main() {
	sql := `CREATE TABLE TBL (
		ID1 INTEGER NOT NULL,
		ID2 INTEGER NOT NULL,
		DATE TIMESTAMP WITH TIMEZONE DEFAULT 0,
		CONSTRAINT PK PRIMARY KEY (ID1, ID2)
	);
	CREATE TABLE TBL2 (
		ID INTEGER NOT NULL,
		TEXT BLOB UNIQUE,
		CONSTRAINT PK PRIMARY KEY (ID)
	);`

	sql = ddlcode.Generalize(sql)
	tables := ddlcode.Parse(sql)

	generateDrawio(tables)
	generateGorm(tables)
	generateJavaCode(tables)
}

func generateDrawio(tables []*ddlcode.Table) {
	config := ddlcode.GetDefaultDrawioConfig()
	config.ExportPath = "codegen.drawio"
	config.Tables = tables
	config.Width = 1100
	config.Height = 850
	file, err := ddlcode.GenerateDrawio(config)
	if err != nil {
		log.Fatal(err)
	}
	file.Flush()
}

func generateGorm(tables []*ddlcode.Table) {
	config := ddlcode.GetDefaultGormConfig()
	config.Package = "model"

	for _, config.Table = range tables {
		files, err := ddlcode.GenerateGorm(config)
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			dirPath := filepath.Dir(f.Path)
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				if err := os.Mkdir(dirPath, 0750); err != nil {
					log.Fatal(err)
				}
			}
			if err := f.Flush(); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func generateJavaCode(tables []*ddlcode.Table) {
	config := ddlcode.GetDefaultJavaConfig()
	config.Package = "com.codegen"
	config.Schema = "schema"

	for _, config.Table = range tables {
		files, err := ddlcode.GenerateJava(config)
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			dirPath := filepath.Dir(f.Path)
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				if err := os.Mkdir(dirPath, 0750); err != nil {
					log.Fatal(err)
				}
			}
			if err := f.Flush(); err != nil {
				log.Fatal(err)
			}
		}
	}
}
```
