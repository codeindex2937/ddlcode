# ddlcode
Generate code from Oracle DDL

## Support
CREATE TABLE
ADD CONSTRAINT ... FOREIGN KEY ... REFERENCES ...

## Reference
[sql2code](https://github.com/zhufuyi/gotool/sql2code)
[go-sqlparser](https://github.com/ikaiguang/go-sqlparser)

## Example Code
```go

func main() {

	sql := `CREATE TABLE TBL (
		ID1 INTEGER NOT NULL,
		ID2 INTEGER NOT NULL,
		CREATED TIMESTAMP WITH TIME ZONE DEFAULT 0,
		TEXT VARCHAR(1),
		CONSTRAINT PK PRIMARY KEY (ID1, ID2)
	);
	CREATE TABLE TBL2 (
		ID3 INTEGER NOT NULL,
		ID4 INTEGER NOT NULL,
		TEXT BLOB UNIQUE,
		CONSTRAINT PK PRIMARY KEY (ID3, ID4)
	);
	ALTER TABLE TBL2 ADD CONSTRAINT fk_name FOREIGN KEY (ID1,ID2) REFERENCES TBL(ID3,ID4);`

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
