package Utils

import (
	database "FSMTestingPlatform/Database"
	bytes2 "bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

/*外部调用方法*/

// EnCode Discard
//The string type message appends header, and then returns []byte
// Deprecated: Use Utils.EnCodeByProto instead, only in protoc.
func EnCode(msgType, msg string) []byte {

	headInt := uint8ToBytes(uint8(0xFA))
	headLen := uint16ToBytes(uint16(len([]byte(msg))))
	headTyp := []byte(msgType)
	bodyMsg := []byte(msg)

	bytes := make([]byte, 0)
	bytes = append(bytes, headInt...)
	bytes = append(bytes, headLen...)
	bytes = append(bytes, headTyp...)
	bytes = append(bytes, bodyMsg...)

	return bytes
}

// DeCode Discard
//[]byte split and return message type and message
// Deprecated: Use Utils.DeCodeByProto instead, only in protoc.
func DeCode (bytes []byte, length int) (msgType string, msg string) {
	if length > 1024 {
		length = 1024
	}
	head := bytes[:4]
	msgType = string(head[2:])
	bodyStr := string(bytes[4:length])
	return msgType, bodyStr
}

// StringToUint32
// 字符转为uint32
func StringToUint32 (str string) (uint32, error)  {
	strUint64, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		fmt.Printf("Parse uint64 error : \n", err.Error())
		return 0, err
	}
	//if strUint64 > 6 || strUint64 < 0 {
	//	return 0, errors.New("Gun id is out of the specified range, range 0, 6\n")
	//}
	return uint32(strUint64), nil
}

func ByteToUint8 (bytes[]byte) uint8 {
	var msgTypeCode uint8
	binary.Read(bytes2.NewBuffer(bytes), binary.BigEndian, &msgTypeCode)
	return msgTypeCode
}

func ByteToUint16 (bytes[]byte) uint16 {
	var msgTypeCode uint16
	binary.Read(bytes2.NewBuffer(bytes), binary.BigEndian, &msgTypeCode)
	return msgTypeCode
}

// ArrayFilterWithLen3
// 随机不重复定长组合 3
// arr 原切片
// flag 下标 []int{0, 1, 2}
// length arr的长度
func ArrayFilterWithLen3 (arr, flag []int, length int) (tempArr [][]int) {

	if flag[0] == length - 3 {
		tempArr = append(tempArr, arr[flag[0]:])
		return tempArr
	}
	for n := flag[2]; n<length; n++ {
		tempArr = append(tempArr, []int{arr[flag[0]], arr[flag[1]], arr[n]})
	}

	if flag[1] == length - 2 {
		flag[0]++
		flag[1] = flag[0] + 1
		flag[2] = flag[1] + 1
		tempArr = append(tempArr, ArrayFilterWithLen3(arr, flag, length)...)
	}
	flag[1]++
	flag[2]++
	tempArr = append(tempArr, ArrayFilterWithLen3(arr, flag, length)...)
	return
}

// ArrayFilterWithLen5
// 随机不重复定长组合 5
// arr 原切片
// flag 下标 []int{0, 1, 2, 3, 4}
// length arr的长度
func ArrayFilterWithLen5 (arr, flag []int, length int) (tempArr [][]int) {
	if flag[0] == length - 5 {
		tempArr = append(tempArr, arr[flag[0]:])
		return tempArr
	}
	for n := flag[4]; n<length; n++ {
		tempArr = append(tempArr,[]int{arr[flag[0]], arr[flag[1]], arr[flag[2]], arr[flag[3]], arr[n]})
	}

	if flag[3] == length - 2 {
		if flag[2] == length - 3  {
			if  flag[1] == length - 4 {
				flag[0] ++
				flag[1] = flag[0] + 1
				flag[2] = flag[1] + 1
				flag[3] = flag[2] + 1
				flag[4] = flag[3] + 1
				tempArr = append(tempArr, ArrayFilterWithLen5(arr, flag, length)...)
			}

			flag[1] ++
			flag[2] = flag[1] + 1
			flag[3] = flag[2] + 1
			flag[4] = flag[3] + 1
			tempArr = append(tempArr, ArrayFilterWithLen5(arr, flag, length)...)
		}

		flag[2] ++
		flag[3] = flag[2] + 1
		flag[4] = flag[3] + 1
		tempArr = append(tempArr, ArrayFilterWithLen5(arr, flag, length)...)
	}

	flag[3]++
	flag[4]++
	tempArr = append(tempArr, ArrayFilterWithLen5(arr, flag, length)...)
	return
}


func ArrayLow2Redis (context context.Context ,keyName string, arr []string, flag []int, length int) {
	// 识别是否超出最大范围 超出则写入 tempArr 并终止程序
	if flag[0] == length - 3 {
		tmp := arr[flag[0]] + "," + arr[flag[0]+1] + "," + arr[flag[0]+2]
		database.RedisPipe.LPush(context, keyName, tmp)
		database.RedisPipe.Exec(context)
		database.RedisPipe.Close()
		return
	}

	// 末尾标识和取数位置比较，循环写入 tempArr
	for n := flag[2]; n<length; n++ {
		tmp := arr[flag[0]] + "," + arr[flag[1]] + "," + arr[n]
		database.RedisPipe.LPush(context, keyName, tmp)
	}

	// 判断“十分位”是否达到最大范围，超过则在“百分位”基础下，进位加1，并进入递归
	if flag[1] == length - 2 {
		flag[0]++
		flag[1] = flag[0] + 1
		flag[2] = flag[1] + 1
		ArrayLow2Redis(context, keyName, arr, flag, length)
	}

	// 循环结束后， 进位加1
	flag[1]++
	flag[2] = flag[1] + 1
	// “十分位”未达到最大范围，进入递归
	ArrayLow2Redis(context, keyName, arr, flag, length)
}


func ArrayHig2Redis (context context.Context, keyName string, arr []string, flag []int, length int) {
	if flag[0] == length - 5 {
		tmp := arr[flag[0]] + "," + arr[flag[0]+1] + "," + arr[flag[0]+2] + "," + arr[flag[0]+3] + "," + arr[flag[0]+4]
		database.RedisPipe.LPush(context, keyName, tmp)
		database.RedisPipe.Exec(context)
		database.RedisPipe.Close()
		return
	}
	for n := flag[4]; n<length; n++ {
		tmp := arr[flag[0]] + "," + arr[flag[1]] + "," + arr[flag[2]] + "," + arr[flag[3]] + "," + arr[n]
		database.RedisPipe.LPush(context, keyName, tmp)
	}

	// 判断“十分位”是否达到最大范围，超过则在“百分位”基础下，进位加1，并进入递归
	if flag[3] == length - 2 {

		// 判断“百分位”是否达到最大范围，超过则在“千分位”基础下，进位加1，并进入递归
		if flag[2] == length - 3  {

			// 判断“千分位”是否达到最大范围，超过则在“万分位”基础下，进位加1，并进入递归
			if  flag[1] == length - 4 {
				flag[0] ++
				flag[1] = flag[0] + 1
				flag[2] = flag[1] + 1
				flag[3] = flag[2] + 1
				flag[4] = flag[3] + 1
				ArrayHig2Redis(context, keyName, arr, flag, length)
			}

			flag[1] ++
			flag[2] = flag[1] + 1
			flag[3] = flag[2] + 1
			flag[4] = flag[3] + 1
			ArrayHig2Redis(context, keyName, arr, flag, length)
		}

		flag[2] ++
		flag[3] = flag[2] + 1
		flag[4] = flag[3] + 1
		ArrayHig2Redis(context, keyName, arr, flag, length)
	}

	flag[3]++
	flag[4]++
	ArrayHig2Redis(context, keyName, arr, flag, length)
}

// IntArr2StringArr
// []int{0, 1}
// []string{"0", "1"}
func IntArr2StringArr (arr []int) []string {
	strSlice := make([]string, len(arr))
	for k, v := range arr {
		strSlice[k] = strconv.Itoa(v)
	}
	return strSlice
}

// StringArr2IntArr
// []string{"0", "1"} to []int{0, 1}
func StringArr2IntArr (arr []string) []int {
	intSlice := make([]int, len(arr))
	for k, v := range arr {
		intSlice[k], _ = strconv.Atoi(v)
	}
	return intSlice
}

// RandValue
// 获取随机数字
func RandValue (n int) int {
	newSource := rand.NewSource(time.Now().UnixNano())
	r := rand.New(newSource)
	return r.Intn(n)
}

// RandValue64
// 获取随机数字
func RandValue64 (n int64) int64 {
	newSource := rand.NewSource(time.Now().UnixNano())
	r := rand.New(newSource)
	return r.Int63n(n)
}

// MysqlSql
// 数据库查询并返回结果
func MysqlSql (tableName string, where map[string]interface{}, in map[string][]string, cols []string) ([]map[string]string, error) {
	session := database.MySQL.Table(tableName)
	for k, v := range where {
		session.Where(k + " = ?", v)
	}
	for k, v := range in {
		session.In(k, v)
	}
	return session.Cols(cols...).QueryString()
}
