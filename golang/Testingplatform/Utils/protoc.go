package Utils

import (
	"TestingPlatform/conf"
	"TestingPlatform/protoc/fsmohp"
	"TestingPlatform/protoc/fsmpmm"
	"TestingPlatform/protoc/fsmvci"
	"encoding/json"
	"errors"
	"github.com/golang/protobuf/proto"
	"strconv"
	"time"
)

var ConfProtoc map[string]interface{}

func init () {
	confInterface, err := ReadFile("./conf/protoc.json", "")
	if err != nil {
		return
	}
	ConfProtoc = confInterface.(map[string]interface{})
}
/* 外部调用方法 */

// EnCodeByProto
// Proto message body Endoce
// Protoc 消息体编码
func EnCodeByProto (msgTypeCode uint, protoMsg proto.Message) ([]byte, error) {
	return addHeaderBytes(msgTypeCode, protoMsg)
}

// DeCodeByProto
// Proto message body decoding
// Protoc 消息体解码 Server 根据端口区分
func DeCodeByProto (port int, bytes []byte) (proto.Message, error) {
	header := bytes[:4]
	length := bytesToUint16(header[1:3])
	msgTypeCode := bytesToUint8(header[3:4])

	if port == 9000 || port == 9010 || port == 9020 {
		return byteToProtoMsgWithVCI(conf.VCIMessage[msgTypeCode], bytes[4:length+4])
	}
	if port == 9030 ||port == 9040 {
		return byteToProtoMsgWithPMM(conf.PMMMessage[msgTypeCode], bytes[4:length+4])
	}
	return nil, errors.New("don't use bed prot")
}

// PB2Json
// Protoc 消息体转化为 Json 字符串
func PB2Json (message proto.Message) (string, error)  {
	bytes, err := json.Marshal(message)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Json2PB
// Json 字符串填充 Protoc 消息体
func Json2PB (jsonStr string, message proto.Message) (proto.Message, error) {
	err := json.Unmarshal([]byte(jsonStr), message)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func FsmpmmId (Id int32) *fsmpmm.Int32Value {
	return &fsmpmm.Int32Value{Value: Id}
}

func FsmpmmCurrentTime (DateTimeLong uint64) *fsmpmm.DateTimeLong {
	if DateTimeLong == 0 {
		DateTimeLong = uint64(time.Now().UnixMilli())
	}
	return &fsmpmm.DateTimeLong{Time: DateTimeLong}
}

func FsmpmmInterval (Interval uint32) *fsmpmm.Uint32Value {
	if Interval == 0 {
		Interval = 200
	}
	return &fsmpmm.Uint32Value{Value: Interval}
}


// VCIPramInit
// VCI gun config
// 车桩交互模块当前配置（每一把枪）
// 用于VCI主线程注册信息帧·响应 (0x80)
// VCIPramList []*VCIPram
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

// SysParameterInit
// System parater config
// 系统参数配置，从config文件中获取初始化信息
// 用于VCI主线程注册信息帧·响应 (0x80)
// SysParameterList *SysParameter
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

// PluggedRecoverInit
// 插枪线程恢复数据初始化
// 用于VCI主线程注册信息帧·响应 (0x80)
// PluggedList []*PluggedRecover
func PluggedRecoverInit (GunId uint32, ChargerState, IsVINStart bool) *fsmvci.PluggedRecover {
	return &fsmvci.PluggedRecover{
		ID:           &fsmvci.Uint32Value{Value: GunId},
		ChargerState: &fsmvci.BoolEnum{Value: ChargerState},
		IsVINStart:   &fsmvci.BoolEnum{Value: IsVINStart},
	}
}

// ChargingRecoverInit
// 充电线程恢复数据初始化
// 用于VCI主线程注册信息帧·响应 (0x80)
// ChargingList []*ChargingRecover
// TODO 初始化全面的数据详情
func ChargingRecoverInit (GunId uint32) *fsmvci.ChargingRecover {
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

// GunBaseStatusInit
// 枪头基础状态信息初始化
// 用于VCI主线程心跳周期信息帧 (0x02)
// GunBaseList  []*GunBaseStatus
func GunBaseStatusInit (GunId uint32, LinkState, PositionStatus bool ) *fsmvci.GunBaseStatus {
	return &fsmvci.GunBaseStatus{
		ID:             &fsmvci.Uint32Value{Value: GunId},
		LinkState:      &fsmvci.BoolEnum{Value: LinkState},
		PositionStatus: &fsmvci.BoolEnum{Value: PositionStatus},
	}
}

// ADModuleAttrInit
// 模块注册参数描述
// 用于PMM主线程模块注册信息帧 (0x02)
// ADModuleAList []*ADModuleAttr
func ADModuleAttrInit (Id int32) *fsmpmm.ADModuleAttr {
	return &fsmpmm.ADModuleAttr{
		ID:                  FsmpmmId(Id),
		DCModuleSN:          "0x1100",
		DCModuleSoftVersion: "0.1",
		DCModuleHardVersion: "0.1",
		LimitI:              &fsmpmm.FloatValue{Value: 36.8},
		LimitP:              &fsmpmm.FloatValue{Value: 459.5},
		MaxV:                &fsmpmm.FloatValue{Value: 350.4},
		MinV:                &fsmpmm.FloatValue{Value: 254.8},
		NormalV:             &fsmpmm.FloatValue{Value: 1.0},
		NormalPower:         &fsmpmm.FloatValue{Value: 254.8},
		NormalI:             &fsmpmm.FloatValue{Value: 1.0},
		NormalInV:           &fsmpmm.FloatValue{Value: 254.8},
	}
}

// ADModuleParamInit
// 模块运行实时参数（设备管理用）
// 用于PMM主线程模块注册信息帧 (0x02)
// ADModulePList []*ADModuleParam
func ADModuleParamInit (Id int32) *fsmpmm.ADModuleParam {
	return &fsmpmm.ADModuleParam{
		ID:             FsmpmmId(Id),
		Va:             &fsmpmm.FloatValue{Value: 220},
		Vb:             &fsmpmm.FloatValue{Value: 220},
		Vc:             &fsmpmm.FloatValue{Value: 220},
		Vdc:            &fsmpmm.FloatValue{Value: 380},
		Ia:             &fsmpmm.FloatValue{Value: 50},
		Ib:             &fsmpmm.FloatValue{Value: 50},
		Ic:             &fsmpmm.FloatValue{Value: 50},
		Idc:            &fsmpmm.FloatValue{Value: 80},
		N:              &fsmpmm.FloatValue{Value: 60},
		P:              &fsmpmm.FloatValue{Value: 360},
		Q:              &fsmpmm.FloatValue{Value: 45},
		PF:             &fsmpmm.FloatValue{Value: 40},
		VU:             &fsmpmm.FloatValue{Value: 41},
		CU:             &fsmpmm.FloatValue{Value: 42},
		Fap:            &fsmpmm.FloatValue{Value: 43},
		Rap:            &fsmpmm.FloatValue{Value: 44},
		Frp:            &fsmpmm.FloatValue{Value: 45},
		Rrp:            &fsmpmm.FloatValue{Value: 46},
		LimitI:         &fsmpmm.FloatValue{Value: 47},
		Tmprt0:         &fsmpmm.FloatValue{Value: 48},
		Tmprt1:         &fsmpmm.FloatValue{Value: 49},
		SystemType:     &fsmpmm.Uint32Value{Value: 40},
		RunTimeCunt:    &fsmpmm.Uint32Value{Value: 40},
		OnOffCunt:      &fsmpmm.Uint32Value{Value: 40},
		PositionStatus: &fsmpmm.BoolEnum{Value: true},
	}
}

// ADModuleAlarmInit
// 故障告警描述
// 用于PMM主线程模块注册信息帧 (0x02)
// AlarmList []*ADModuleAlarm
func ADModuleAlarmInit () *fsmpmm.ADModuleAlarm {
	alarmTime :=time.Now().UnixMilli()
	downTime, _ := time.ParseDuration("5s")
	downTime1 := time.Now().Add(5 * downTime).UnixMilli()

	return &fsmpmm.ADModuleAlarm{
		AlarmType:     fsmpmm.AlarmTypeEnum(1),
		AlarmState:    fsmpmm.AlarmStateEnum(1),
		AlarmTime:     &fsmpmm.DateTimeShort{Time: uint32(alarmTime)},
		AlarmDownTime: &fsmpmm.DateTimeShort{Time: uint32(downTime1)},
	}
}

// AlarmDataTypeInit
// 复活告警信号描述
// 用于PMM主线程模块注册信息帧 (0x02)
// AlarmDataList []*AlarmDataType
func AlarmDataTypeInit (Id int32) *fsmpmm.AlarmDataType {
	return &fsmpmm.AlarmDataType{
		ID:           FsmpmmId(Id),
		AlarmState40: &fsmpmm.Uint32Value{Value: 0},
		AlarmState42: &fsmpmm.Uint32Value{Value: 0},
		AlarmState43: &fsmpmm.Uint32Value{Value: 0},
	}
}

// MainStatusInit
// 主接触器状态描述
// TODO
func MainStatusInit (Id int32) *fsmpmm.MainStatus {
	return &fsmpmm.MainStatus{
		ID:           &fsmpmm.Int32Value{Value: Id},
		MainMode:     fsmpmm.ContactorStateEnum(3),
		MatrixID:     nil,
		ADModuleID:   nil,
		BatVol:       nil,
		ModVol:       nil,
		AlarmAnsList: nil,
	}
}

// MatrixStatusInit
// 阵列接触器状态描述
// TODO
func MatrixStatusInit (Id int32) *fsmpmm.MatrixStatus {
	return &fsmpmm.MatrixStatus{
		ID:           &fsmpmm.Int32Value{Value: Id},
		MatrixMode:   fsmpmm.ContactorStateEnum(3),
		AlarmAnsList: nil,
	}
}

// OHPSettlementModuleEnum
// 订单结算通讯模块枚举
func OHPSettlementModuleEnum (SettlementModule ...fsmohp.SettlementModuleEnum) (SettlementModuleEnum []fsmohp.SettlementModuleEnum) {
	for _, v := range SettlementModule {
		SettlementModuleEnum = append(SettlementModuleEnum, v)
	}
	return
}

