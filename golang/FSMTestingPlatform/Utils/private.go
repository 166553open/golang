package Utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"FSMTestingPlatform/Protoc/fsmpmm"
	"FSMTestingPlatform/Protoc/fsmvci"

	"github.com/golang/protobuf/proto"
)

// uint16ToBytes
// ----------------------------------------------------------------------------
// uint16 类型数据转换[]byte类型
// ----------------------------------------------------------------------------
func uint16ToBytes(n uint16) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

// uint8ToBytes
// ----------------------------------------------------------------------------
// uint8 类型数据转换[]byte类型
// ----------------------------------------------------------------------------
func uint8ToBytes(n uint8) []byte {
	bytebuff := bytes.NewBuffer([]byte{})
	binary.Write(bytebuff, binary.BigEndian, n)
	return bytebuff.Bytes()
}

// bytesToUint16
// ----------------------------------------------------------------------------
// []byte 类型数据转换 uint16 类型
// ----------------------------------------------------------------------------
func bytesToUint16(bys []byte) uint16 {
	bytebuff := bytes.NewBuffer(bys)
	var data uint16
	binary.Read(bytebuff, binary.BigEndian, &data)
	return data
}

// bytesToUint8
// ----------------------------------------------------------------------------
// []byte 类型数据转换 uint8 类型
// ----------------------------------------------------------------------------
func bytesToUint8(bys []byte) uint8 {
	bytebuff := bytes.NewBuffer(bys)
	var data uint8
	binary.Read(bytebuff, binary.BigEndian, &data)
	return data
}

// byteToProtoMsgWithPMM
// ----------------------------------------------------------------------------
// Protoc 消息体反序列化
// ----------------------------------------------------------------------------
func byteToProtoMsgWithPMM(msgTypeName string, bytes []byte) (proto.Message, error) {
	switch msgTypeName {
	case "PowerMatrixLogin":
		protoMsg := &fsmpmm.PowerMatrixLogin{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "PowerMatrixLoginAns":
		protoMsg := &fsmpmm.PowerMatrixLoginAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "ADModuleLogin":
		protoMsg := &fsmpmm.ADModuleLogin{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "ADModuleLoginAns":
		protoMsg := &fsmpmm.ADModuleLoginAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "PMMHeartbeatReq":
		protoMsg := &fsmpmm.PMMHeartbeatReq{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "PMMHeartbeatAns":
		protoMsg := &fsmpmm.PMMHeartbeatAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "MainContactorHeartbeatReq":
		protoMsg := &fsmpmm.MainContactorHeartbeatReq{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "MainContactorHeartbeatAns":
		protoMsg := &fsmpmm.MainContactorHeartbeatAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "MainContactorRTpush":
		protoMsg := &fsmpmm.MainContactorRTpush{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "MainContactorRTpull":
		protoMsg := &fsmpmm.MainContactorRTpull{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	default:
		fmt.Printf("Msg type name '%s' is no found\n ", msgTypeName)
		return nil, errors.New("msg type name is no found\n")
	}
}

// byteToProtoMsgWithVCI
// ----------------------------------------------------------------------------
// Protoc 消息体反序列化
// ----------------------------------------------------------------------------
func byteToProtoMsgWithVCI(msgTypeName string, bytes []byte) (proto.Message, error) {
	switch msgTypeName {
	case "VCIManagerRegisterInfo":
		protoMsg := &fsmvci.VCIManagerRegisterInfo{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VCIManagerRegisterReponse":
		protoMsg := &fsmvci.VCIManagerRegisterReponse{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil

	case "VCIManagerHeartBeatInfo":
		protoMsg := &fsmvci.VCIManagerHeartBeatInfo{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VCIManagerHeartBeatReponse":
		protoMsg := &fsmvci.VCIManagerHeartBeatReponse{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil

	case "VCIChargerRegisterInfo":
		protoMsg := &fsmvci.VCIChargerRegisterInfo{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VCIChargerRegisterResponse":
		protoMsg := &fsmvci.VCIChargerRegisterResponse{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil

	case "VCIChargerHeartBeatInfo":
		protoMsg := &fsmvci.VCIChargerHeartBeatInfo{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VCIChargerHeartBeatResponse":
		protoMsg := &fsmvci.VCIChargerHeartBeatResponse{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil

	case "VCIChargerRTInfo":
		protoMsg := &fsmvci.VCIChargerRTInfo{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VCIChargerRTRsponse":
		protoMsg := &fsmvci.VCIChargerRTRsponse{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	default:
		fmt.Printf("Msg type name '%s' is no found\n ", msgTypeName)
		return nil, errors.New("msg type name is no found\n")
	}
}

// addHeaderBytes
// ----------------------------------------------------------------------------
// Protoc 消息体序列化后，添加head头
// ----------------------------------------------------------------------------
func addHeaderBytes(msgTypeCode uint, message proto.Message) ([]byte, error) {
	bodyMsg, err := proto.Marshal(message)
	if err != nil {
		fmt.Printf("Proto marshal is error : %s\n", err.Error())
		return nil, err
	}

	headerInt := uint8ToBytes(uint8(0xFA))
	headerLen := uint16ToBytes(uint16(uint(len(bodyMsg))))
	headerType := uint8ToBytes(uint8(msgTypeCode))

	bytes := make([]byte, 0)
	bytes = append(bytes, headerInt...)
	bytes = append(bytes, headerLen...)
	bytes = append(bytes, headerType...)
	bytes = append(bytes, bodyMsg...)
	return bytes, nil
}
