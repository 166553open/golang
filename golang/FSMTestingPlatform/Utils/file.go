package Utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

// ReadFile 读取文件中信息，返回 interface类型
func ReadFile(fs, name string) (map[string]interface{}, error) {
	byteData, err := ioutil.ReadFile(fs)
	if err != nil {
		fmt.Printf("this file %s open file, error info %s \n", fs, err.Error())
		return nil, err
	}
	dataMap := make(map[string]interface{})
	err = json.Unmarshal(byteData, &dataMap)
	if err != nil {
		fmt.Printf("byte data to map data error: %s \n", err.Error())
		return nil, err
	}
	return SpecifyTupleUnderConfig(dataMap, name)
}

func SpecifyTupleUnderConfig (dataMap map[string]interface{},  name string) (map[string]interface{}, error) {
	names := strings.Split(name, "/")
	tempMap := dataMap

	for _, v := range names {
		if _, ok := tempMap[v].(map[string]interface{}); ok {
			tempMap = tempMap[v].(map[string]interface{})
		}else {
			return tempMap, nil
		}
	}
	return tempMap, nil
}
