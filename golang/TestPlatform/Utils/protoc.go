package Utils

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"goWork/conf"
	"goWork/protoc/fsmvci"
	"strconv"
)

var ConfProtoc map[string]interface{}

func init () {
	confInterface, _ := ReadFile("./conf/protoc.json", "")
	ConfProtoc = confInterface.(map[string]interface{})
}

//Proto message body Endoce
func EnCodeByProto (msgTypeCode uint, protoMsg proto.Message) ([]byte, error) {
	return addHeaderBytes(msgTypeCode, protoMsg)
}

//Proto message body decoding
func DeCodeByProto (bytes []byte) (proto.Message, error) {
	header := bytes[:4]
	length := bytesToUint16(header[1:3])
	msgTypeCode := bytesToUint8(header[3:4])
	msgTypeName :=  conf.ProtoMessage[msgTypeCode]

	return byteToProtoMsg(msgTypeName, bytes[4:length+4])
}

//VCI gun config
func  VCIPramInit (GunId uint32, GunType int32,  AuxType, ElockEnable bool) *fsmvci.VCIPram {
	confMap := ConfProtoc["VCIPram"].(map[string]interface{})

	GunAmount := uint32(confMap["GunAmount"].(float64))

	GunIdStr := strconv.FormatUint(uint64(GunId), 10)
	confGun := confMap[GunIdStr].(map[string]interface{})

	return &fsmvci.VCIPram{
		GunAmount:   &fsmvci.Uint32Value{Value: GunAmount},
		ID:          &fsmvci.Uint32Value{Value: GunId},
		Type:         fsmvci.GunTypeEnum(GunType),
		LimitI:      &fsmvci.FloatValue{Value: float32(confGun["LimitI"].(float64))},
		LimitV:      &fsmvci.FloatValue{Value: float32(confGun["LimitV"].(float64))},
		MaxP:        &fsmvci.FloatValue{Value: float32(confGun["MaxP"].(float64))},
		AuxType:     &fsmvci.BoolEnum{Value: AuxType},
		ElockEnable: &fsmvci.BoolEnum{Value: ElockEnable},
	}
}

//System parater config
func SysParameterInit() *fsmvci.SysParameter {
	confMap := ConfProtoc["SysParameter"].(map[string]interface{})

	SysVolMax := confMap["SysVolMax"].(float64)
	SysCurrMax := confMap["SysCurrMax"].(float64)
	SysVolVMin := confMap["SysVolVMin"].(float64)
	SysVolCMin := confMap["SysVolCMin"].(float64)
	SysCurrMin := confMap["SysCurrMin"].(float64)

	return &fsmvci.SysParameter{
		SysVolMax:  &fsmvci.DoubleValue{Value: SysVolMax},
		SysCurrMax: &fsmvci.DoubleValue{Value: SysCurrMax},
		SysVolVMin: &fsmvci.DoubleValue{Value: SysVolVMin},
		SysVolCMin: &fsmvci.DoubleValue{Value: SysVolCMin},
		SysCurrMin: &fsmvci.DoubleValue{Value: SysCurrMin},
	}
}

//
func PluggedRecoverInit (GunId uint32, ChargerState, IsVINStart bool) *fsmvci.PluggedRecover {
	return &fsmvci.PluggedRecover{
		ID:           &fsmvci.Uint32Value{Value: GunId},
		ChargerState: &fsmvci.BoolEnum{Value: ChargerState},
		IsVINStart:   &fsmvci.BoolEnum{Value: IsVINStart},
	}
}

//relive
func ChargingRecover (GunId uint32) *fsmvci.ChargingRecover {
	return &fsmvci.ChargingRecover{
		ID:            &fsmvci.Uint32Value{Value: GunId},
		FaultState1:   nil,
		FaultState2:   nil,
		FaultState3:   nil,
		BMSCommState:  nil,
		BMSRecvState:  nil,
		BMSType:       &fsmvci.Int32Value{Value: 1},
		BMSTimeoutCnt: nil,
		ElockState:    nil,
		AuxPowerState: nil,
		BMSCurrMax:    nil,
		BMSVolMax:     nil,
		CellVolMax:    nil,
		CellTempMax:   nil,
		InsultState:   nil,
		InsultResult:  nil,
		InsultVol:     nil,
	}
}

func GunBaseStatusInit (GunId uint32, LinkState, PositionStatus bool ) *fsmvci.GunBaseStatus {
	return &fsmvci.GunBaseStatus{
		ID:             &fsmvci.Uint32Value{Value: GunId},
		LinkState:      &fsmvci.BoolEnum{Value: LinkState},
		PositionStatus: &fsmvci.BoolEnum{Value: PositionStatus},
	}
}



//Message body decoding by message type name
func byteToProtoMsg(msgTypeName string, bytes []byte) (proto.Message, error) {
	switch msgTypeName {
	case "VehicleChargingInterfaceLogin":
		protoMsg := &fsmvci.VehicleChargingInterfaceLogin{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VehicleChargingInterfaceLoginAns":
		protoMsg := &fsmvci.VehicleChargingInterfaceLoginAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil

	case "VCImainHeartbeatReq":
		protoMsg := &fsmvci.VCImainHeartbeatReq{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VCImainHeartbeatAns":
		protoMsg := &fsmvci.VCImainHeartbeatAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil

	case "VCIPluggedHeartbeatReq":
		protoMsg := &fsmvci.VCIPluggedHeartbeatReq{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VCIPluggedHeartbeatAns":
		protoMsg := &fsmvci.VCIPluggedHeartbeatAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil

	case "VCIChargingHeartbeatReq":
		protoMsg := &fsmvci.VCIChargingHeartbeatReq{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VCIChargingHeartbeatAns":
		protoMsg := &fsmvci.VCIChargingHeartbeatAns{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil

	case "VCIChargingRTpush":
		protoMsg := &fsmvci.VCIChargingRTpush{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil
	case "VCIChargingRTpull":
		protoMsg := &fsmvci.VCIChargingRTpull{}
		err := proto.Unmarshal(bytes, protoMsg)
		if err != nil {
			fmt.Printf("Proto marshal is error : %s\n", err.Error())
			return nil, err
		}
		return protoMsg, nil

	default:
		fmt.Printf("Msg type name is no found\n")
		return nil, errors.New("msg type name is no found\n")
	}
}

//Add header to byte type proto message
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