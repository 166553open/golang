package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/go-redis/redis/v8"
	"xorm.io/xorm"
)

var (
	MySQL     *xorm.Engine
	Redis     *redis.Client
	err       error
	RedisPipe redis.Pipeliner
)

func init() {
	MySQL, err = mysqlConnect()
	if err != nil {
		return
	}

	Redis = redisConnect()
	RedisPipe = Redis.Pipeline()
}

// readFile
// ------------------------------------------------------------------------------------
// 从json文件读取配置信息
// ------------------------------------------------------------------------------------
func readFile(fs, name string) (map[string]interface{}, error) {
	byte, err := ioutil.ReadFile(fs)
	if err != nil {
		fmt.Printf("this file %s open file, error info %s \n", fs, err.Error())
		return nil, err
	}
	dataMap := make(map[string]interface{})
	err = json.Unmarshal(byte, &dataMap)
	if err != nil {
		fmt.Printf("byte data to map data error: %s \n", err.Error())
		return nil, err
	}
	if _, ok := dataMap[name]; ok {
		return dataMap[name].(map[string]interface{}), nil
	} else {
		return dataMap, nil
	}
}
