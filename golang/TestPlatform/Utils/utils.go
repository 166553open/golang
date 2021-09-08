package Utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
)

//Discard
//The string type message appends header, and then returns []byte
// Deprecated: Use Utils.EnCodeByProto instead.
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

//Discard
//[]byte split and return message type and message
// Deprecated: Use Utils.DeCodeByProto instead.
func DeCode (bytes []byte, length int) (msgType string, msg string) {
	if length > 1024 {
		length = 1024
	}
	head := bytes[:4]
	msgType = string(head[2:])
	bodyStr := string(bytes[4:length])
	return msgType, bodyStr
}

func StringToUint32AndLimit (str string) (uint32, error)  {
	strUint64, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		fmt.Printf("Parse uint64 error : \n", err.Error())
		return 0, err
	}
	if strUint64 > 6 || strUint64 < 0 {
		return 0, errors.New("Gun id is out of the specified range, range 1, 6\n")
	}
	return uint32(strUint64), nil
}

//Uint16 type to byte type
func uint16ToBytes(n uint16) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

//Uint8 type to byte type
func uint8ToBytes(n uint8) []byte {
	bytebuff := bytes.NewBuffer([]byte{})
	binary.Write(bytebuff, binary.BigEndian, n)
	return bytebuff.Bytes()
}

//Byte type to uint16 type
func bytesToUint16(bys []byte) uint16 {
	bytebuff := bytes.NewBuffer(bys)
	var data uint16
	binary.Read(bytebuff, binary.BigEndian, &data)
	return data
}

//Byte type to uint8 type
func bytesToUint8(bys []byte) uint8 {
	bytebuff := bytes.NewBuffer(bys)
	var data uint8
	binary.Read(bytebuff, binary.BigEndian, &data)
	return data
}

