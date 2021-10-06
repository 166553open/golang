package Utils

import (
	"TestingPlatform/protoc/fsmpmm"
	"TestingPlatform/protoc/fsmvci"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
)

//Uint16 type to byte type
func uint16ToBytes (n uint16) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

//Uint8 type to byte type
func uint8ToBytes (n uint8) []byte {
	bytebuff := bytes.NewBuffer([]byte{})
	binary.Write(bytebuff, binary.BigEndian, n)
	return bytebuff.Bytes()
}

//Byte type to uint16 type
func bytesToUint16 (bys []byte) uint16 {
	bytebuff := bytes.NewBuffer(bys)
	var data uint16
	binary.Read(bytebuff, binary.BigEndian, &data)
	return data
}

//Byte type to uint8 type
func bytesToUint8 (bys []byte) uint8 {
	bytebuff := bytes.NewBuffer(bys)
	var data uint8
	binary.Read(bytebuff, binary.BigEndian, &data)
	return data
}

// Message body decoding by message type name
// Protoc 消息体反序列化
func byteToProtoMsgWithPMM (msgTypeName string, bytes []byte) (proto.Message, error) {
	switch msgTypeName {
	case "PowerMatrixLogin" :
		protoMsg := &fsmpmm.PowerMatrixLogin{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "PowerMatrixLoginAns" :
		protoMsg := &fsmpmm.PowerMatrixLoginAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "ADModuleLogin" :
		protoMsg := &fsmpmm.ADModuleLogin{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "ADModuleLoginAns" :
		protoMsg := &fsmpmm.ADModuleLoginAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "PMMHeartbeatReq" :
		protoMsg := &fsmpmm.PMMHeartbeatReq{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "PMMHeartbeatAns" :
		protoMsg := &fsmpmm.PMMHeartbeatAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "MainContactorHeartbeatReq" :
		protoMsg := &fsmpmm.MainContactorHeartbeatReq{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "MainContactorHeartbeatAns" :
		protoMsg := &fsmpmm.MainContactorHeartbeatAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "MainContactorRTpush" :
		protoMsg := &fsmpmm.MainContactorRTpush{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "MainContactorRTpull" :
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

// Message body decoding by message type name
// Protoc 消息体反序列化
func byteToProtoMsgWithVCI (msgTypeName string, bytes []byte) (proto.Message, error) {
	switch msgTypeName {
		case "VehicleChargingInterfaceLogin":
			protoMsg := &fsmvci.VehicleChargingInterfaceLogin{}
			err := proto.Unmarshal(bytes, protoMsg)
			if err != nil {
				fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
				return nil, err
			}
			return protoMsg, nil
		case "VehicleChargingInterfaceLoginAns":
			protoMsg := &fsmvci.VehicleChargingInterfaceLoginAns{}
			err := proto.Unmarshal(bytes, protoMsg)
			if err != nil {
				fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
				return nil, err
			}
			return protoMsg, nil

		case "VCImainHeartbeatReq":
			protoMsg := &fsmvci.VCImainHeartbeatReq{}
			err := proto.Unmarshal(bytes, protoMsg)
			if err != nil {
				fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
				return nil, err
			}
			return protoMsg, nil
		case "VCImainHeartbeatAns":
			protoMsg := &fsmvci.VCImainHeartbeatAns{}
			err := proto.Unmarshal(bytes, protoMsg)
			if err != nil {
				fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
				return nil, err
			}
			return protoMsg, nil

		case "VCIPluggedHeartbeatReq":
			protoMsg := &fsmvci.VCIPluggedHeartbeatReq{}
			err := proto.Unmarshal(bytes, protoMsg)
			if err != nil {
				fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
				return nil, err
			}
			return protoMsg, nil
		case "VCIPluggedHeartbeatAns":
			protoMsg := &fsmvci.VCIPluggedHeartbeatAns{}
			err := proto.Unmarshal(bytes, protoMsg)
			if err != nil {
				fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
				return nil, err
			}
			return protoMsg, nil

		case "VCIChargingHeartbeatReq":
			protoMsg := &fsmvci.VCIChargingHeartbeatReq{}
			err := proto.Unmarshal(bytes, protoMsg)
			if err != nil {
				fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
				return nil, err
			}
			return protoMsg, nil
		case "VCIChargingHeartbeatAns":
			protoMsg := &fsmvci.VCIChargingHeartbeatAns{}
			err := proto.Unmarshal(bytes, protoMsg)
			if err != nil {
				fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
				return nil, err
			}
			return protoMsg, nil

		case "VCIChargingRTpush":
			protoMsg := &fsmvci.VCIChargingRTpush{}
			err := proto.Unmarshal(bytes, protoMsg)
			if err != nil {
				fmt.Printf("Proto %+v  marshal is error : %s\n", protoMsg, err.Error())
				return nil, err
			}
			return protoMsg, nil
		case "VCIChargingRTpull":
			protoMsg := &fsmvci.VCIChargingRTpull{}
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


// Add header to byte type proto message
// Protoc 消息体序列化后，添加head头
func addHeaderBytes (msgTypeCode uint, message proto.Message) ([]byte, error) {
	bodyMsg, err := proto.Marshal(message)
	if err != nil {
		fmt.Printf("Proto unmarshal is error : %s\n", err.Error())
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