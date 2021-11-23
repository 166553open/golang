package database

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"io/ioutil"
	"xorm.io/xorm"
)

var
(
	MySQL *xorm.Engine
	Redis *redis.Client
	err error
	RedisPipe  redis.Pipeliner
)
func init () {
	MySQL, err = mysqlConnect()
	if err != nil {
		return
	}

	Redis = redisConnect()
	RedisPipe = Redis.Pipeline()
}

func readFile(fs, name string) (interface{}, error) {
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
		return dataMap[name], nil
	}else{
		return dataMap, nil
	}
}
