package main

import (
	"fmt"
	"os"

	"github.com/pschlump/Go-FTL/server/sizlib"
	"github.com/pschlump/MiscLib"
)

type PGColumn struct {
	ColumnName string
}

type PGTable struct {
	TableName string
	Columns   []PGColumn
}

func PGCheckTableColumns(tab PGTable) (err error) {
	err = CheckTable("public", tab.TableName)
	if err != nil {
		return
	}
	cm, _ := GetColumnMap("public", tab.TableName)
	for _, cc := range tab.Columns {
		if !cm[cc.ColumnName] {
			return fmt.Errorf("Missing column [%s] in table [%s]\n", cc.ColumnName, tab.TableName)
		}
	}
	return
}

// func (hdlr *TabServer2Type) GetTableInformationSchema(conn *sizlib.MyDb, TableName string) (rv DbTableType, err error) {

func CheckTable(DbSchema, TableName string) (err error) {
	qry := `SELECT * FROM information_schema.tables WHERE table_schema = $1 and table_name = $2`
	data := sizlib.SelData(DB, qry, DbSchema, TableName)
	if data == nil || len(data) == 0 {
		fmt.Fprintf(os.Stderr, "%sError(190532): Missing table:%s%s\n", MiscLib.ColorRed, TableName, MiscLib.ColorReset)
		err = fmt.Errorf("Error(190532): Missing table:%s", TableName)
		return
	}
	return
}

func GetColumnMap(DbSchema, TableName string) (cm map[string]bool, err error) {
	qry := `SELECT * FROM information_schema.columns WHERE table_schema = $1 and table_name = $2`
	// cols := sizlib.SelData(conn.Db, qry, g_schema, TableName)
	cols := sizlib.SelData(DB, qry, DbSchema, TableName)

	// fmt.Printf("data=%s\n", lib.SVarI(data))
	// fmt.Printf("cols=%s\n", lib.SVarI(cols))
	cm = make(map[string]bool)
	for _, vv := range cols {
		cm[vv["column_name"].(string)] = true
		// rv.DbColumns = append(rv.DbColumns, DbColumnsType{
		// 	ColumnName: vv["column_name"].(string),
		// 	DBType:     vv["data_type"].(string),
		// 	TypeCode:   GetTypeCode(vv["data_type"].(string)),
		// })
	}
	// godebug.Db2Printf(db83, "rv=%s\n", lib.SVarI(rv))
	return
}
