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

