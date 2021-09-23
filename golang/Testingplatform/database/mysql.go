package database

import (
	"TestingPlatform/Utils"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

func mysql_connect() (*xorm.Engine, error) {

	mysqlConf, err := Utils.ReadFile("./conf/database.json", "mysql")
	if err != nil {
		fmt.Printf("ReadFile is error : %s", err.Error())
		return nil, err
	}

	mysqlConfData := mysqlConf.(map[string]interface{})
	host := mysqlConfData["host"].(string)
	port := mysqlConfData["port"].(string)
	user := mysqlConfData["user"].(string)
	password := mysqlConfData["password"].(string)
	database := mysqlConfData["database"].(string)

	showsql := mysqlConfData["showsql"].(bool)

	sourceName :=  user + ":" + password + "@tcp(" + host + ":" + port + ")/" + database +  "?charset=utf8"

	engine, err := xorm.NewEngine("mysql", sourceName)
	if err != nil {
		fmt.Printf("NewEngine is error : %s", err.Error())
		return nil, err
	}
	engine.ShowSQL(showsql)
	return engine, nil
}
