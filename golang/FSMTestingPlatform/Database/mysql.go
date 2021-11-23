package database

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

// mysqlConnect
// ----------------------------------------------------------------------------
// 链接到Mysql库
// ----------------------------------------------------------------------------
func mysqlConnect() (*xorm.Engine, error) {

	mysqlConfData, err := readFile("./Conf/database.json", "mysql")
	if err != nil {
		fmt.Printf("ReadFile is error : %s", err.Error())
		return nil, err
	}

	host := mysqlConfData["host"].(string)
	port := mysqlConfData["port"].(string)
	user := mysqlConfData["user"].(string)
	password := mysqlConfData["password"].(string)
	database := mysqlConfData["database"].(string)

	showsql := mysqlConfData["showsql"].(bool)

	sourceName := user + ":" + password + "@tcp(" + host + ":" + port + ")/" + database + "?charset=utf8"

	engine, err := xorm.NewEngine("mysql", sourceName)
	if err != nil {
		fmt.Printf("NewEngine is error : %s", err.Error())
		return nil, err
	}
	engine.ShowSQL(showsql)
	return engine, nil
}
