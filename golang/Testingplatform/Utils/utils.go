package Utils

import (
	bytes2 "bytes"
	"encoding/binary"
	"fmt"
	"strconv"
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
func ArrayFilterWithLen3 (arr, flag []int, n int) (tempArr [][]int) {
	if n > len(arr) {
		n=len(arr)
	}
	if flag[0] == len(arr) - n {
		tempArr = append(tempArr, arr[len(arr)-n:])
		return tempArr
	}
	for n := flag[2]; n<len(arr); n++ {
		testTemp := make([]int, 0)
		tempArr = append(tempArr, ElementArrWithInt(testTemp, arr[flag[0]], arr[flag[1]], arr[n]))
	}

	if flag[1] == len(arr) - 2 {
		flag[0]++
		flag[1] = flag[0] + 1
		flag[2] = flag[1] + 1
		return ArrayFilterWithLen3(arr, flag, n)
	}
	flag[1]++
	flag[2]++
	return ArrayFilterWithLen3(arr, flag, n)
}

// ArrayFilterWithLen5
// 随机不重复定长组合 5
func ArrayFilterWithLen5 (arr, flag []int, n int) (tempArr [][]int) {
	if n > len(arr) {
		n=len(arr)
	}
	if flag[0] == len(arr) - n {
		tempArr = append(tempArr, arr[len(arr)-n:])
		return tempArr
	}
	for n := flag[4]; n<len(arr); n++ {
		testTemp := make([]int, 0)
		tempArr = append(tempArr, ElementArrWithInt(testTemp, arr[flag[0]], arr[flag[1]], arr[flag[2]], arr[flag[3]], arr[n]))
	}

	if flag[3] == len(arr) - 2 {
		if flag[2] == len(arr) - 3  {
			if  flag[1] == len(arr) - 4 {
				flag[0] ++
				flag[1] = flag[0] + 1
				flag[2] = flag[1] + 1
				flag[3] = flag[2] + 1
				flag[4] = flag[3] + 1
				return ArrayFilterWithLen5(arr, flag, n)
			}

			flag[1] ++
			flag[2] = flag[1] + 1
			flag[3] = flag[2] + 1
			flag[4] = flag[3] + 1
			return ArrayFilterWithLen5(arr, flag, n)
		}

		flag[2] ++
		flag[3] = flag[2] + 1
		flag[4] = flag[3] + 1
		return ArrayFilterWithLen5(arr, flag, n)
	}

	flag[3]++
	flag[4]++
	return ArrayFilterWithLen5(arr, flag, n)
}

// GoldenSection
// a
func GoldenSection (min, max float32, n int) (golden []float32) {
	if n == 0 {
		return golden
	}
	rangeValue := max - min
	goldenNumber := rangeValue*0.618 + min
	golden = append(golden, goldenNumber)
	minList, maxList := GoldenSection(min, goldenNumber, n-1), GoldenSection(goldenNumber, max, n-1)
	golden = append(append(golden, minList...), maxList...)
	return golden
}

// ElementArrWithInt
// 多个元素生成数组
func ElementArrWithInt (arr []int, element ...int) []int {
	for _, v := range element {
		arr = append(arr, v)
	}
	return arr
}

// ElementArrWithInt32
// 多个元素生成数组
func ElementArrWithInt32 (arr []int32, element ...int32) []int32 {
	for _, v := range element {
		arr = append(arr, v)
	}
	return arr
}