package Utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// ReadFile 读取文件中信息，返回 interface类型
func ReadFile(fs, name string) (interface{}, error) {
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
