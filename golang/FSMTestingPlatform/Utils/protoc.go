package Utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	conf "FSMTestingPlatform/Conf"
	database "FSMTestingPlatform/Database"
	"FSMTestingPlatform/Protoc/fsmohp"
	"FSMTestingPlatform/Protoc/fsmpmm"
	"FSMTestingPlatform/Protoc/fsmvci"

	"github.com/golang/protobuf/proto"
)

/* 外部调用方法 */

// EnCodeByProto
// ----------------------------------------------------------------
// Protoc 消息体编码
// ----------------------------------------------------------------
func EnCodeByProto(msgTypeCode uint, protoMsg proto.Message) ([]byte, error) {
	return addHeaderBytes(msgTypeCode, protoMsg)
}

// DeCodeByProto
// ----------------------------------------------------------------
// Protoc 消息体解码 Server 根据端口区分
// TODO 修改端口号
// ----------------------------------------------------------------
func DeCodeByProto(port int, bytes []byte) (proto.Message, error) {
	header := bytes[:4]
	length := bytesToUint16(header[1:3])
	msgTypeCode := bytesToUint8(header[3:4])

	if port == 9000 || port == 9010 || port == 9020 {
		return byteToProtoMsgWithVCI(conf.VCIMessage[msgTypeCode], bytes[4:length+4])
	}
	if port == 9030 || port == 9040 {
		return byteToProtoMsgWithPMM(conf.PMMMessage[msgTypeCode], bytes[4:length+4])
	}
	return nil, errors.New("don't use bed prot")
}

// PB2Json
// ----------------------------------------------------------------
// Protoc 消息体转化为 Json 字符串
// ----------------------------------------------------------------
func PB2Json(message proto.Message) (string, error) {
	bytes, err := json.Marshal(message)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Json2PB
// ----------------------------------------------------------------
// Json 字符串填充 Protoc 消息体
// ----------------------------------------------------------------
func Json2PB(jsonStr string, message proto.Message) (proto.Message, error) {
	err := json.Unmarshal([]byte(jsonStr), message)
	if err != nil {
		return nil, err
	}
	return message, nil
}

/* VCI Register START*/

// VCIRegister
// ----------------------------------------------------------------
// VCI Manager 注册数据帧 0x00
// ----------------------------------------------------------------
func VCIRegister(state int) *fsmvci.VCIManagerRegisterInfo {
	protocMap, err := ReadFile("./Conf/protoc.json", "VCI/Common")
	ProtocolVer, SelfCheckResult := "0.0.0", "0.0.0"
	if err == nil {
		ProtocolVer = protocMap["VehicleChargingProtoVersion"].(string)
		SelfCheckResult = protocMap["MainStateMachineVendor"].(string)
	}

	return &fsmvci.VCIManagerRegisterInfo{
		ProtocolVer:     ProtocolVer,
		EditionNumber:   SelfCheckResult,
		SelfCheckResult: fsmvci.SelfCheckState(state),
	}
}

// VCIRegisterHeart
// ----------------------------------------------------------------
// VCI Manager 线程 0x02
// ----------------------------------------------------------------
func VCIRegisterHeart(hBeatCnt, hBeatPeriod uint32, timeNow uint64, gunBaseList []*fsmvci.GunBasicState, connectList []*fsmvci.GunConnectState) *fsmvci.VCIManagerHeartBeatInfo {
	return &fsmvci.VCIManagerHeartBeatInfo{
		HeartBeatCnt:          hBeatCnt,
		TimeNow:               timeNow,
		HeartBeatPeriod:       hBeatPeriod,
		GunBasicStateListLong: uint32(len(gunBaseList)),
		ConnectStateListLong:  uint32(len(connectList)),
		GunBasicStateList:     gunBaseList,
		ConnectStateList:      connectList,
	}
}

// GunBaseStatusInit
// ----------------------------------------------------------------
// VCI枪头基础状态信息 (0x02)
// ----------------------------------------------------------------
func GunBaseStatusInit(gunId, linkState, positionStatus uint32) *fsmvci.GunBasicState {
	return &fsmvci.GunBasicState{
		GunID:           gunId,
		LinkState:       linkState,
		PositionedState: positionStatus,
	}
}

// ConnectStateList
// ----------------------------------------------------------------
// VCI枪头连接阶段状态信息 (0x02)
// ----------------------------------------------------------------
func ConnectStateList(auxDriver, auxFeedback, eLockDriver, eLockFeedback uint32) *fsmvci.GunConnectState {
	return &fsmvci.GunConnectState{
		AuxDriver:     auxDriver,
		AuxFeedback:   auxFeedback,
		ELockDriver:   eLockDriver,
		ELockFeedback: eLockFeedback,
		TempPositive:  conf.Float32Value[RandValue(len(conf.IntValue))],
		TempNegative:  conf.Float32Value[RandValue(len(conf.IntValue))],
	}
}

/* VCI Register END*/

/* VCI Charging 充电线程 START*/

// VCIChargingRegister
// ----------------------------------------------------------------
// VCI Charger注册数据帧 0x01
// ----------------------------------------------------------------
func VCIChargingRegister(port uint32) *fsmvci.VCIChargerRegisterInfo {
	return &fsmvci.VCIChargerRegisterInfo{
		Register:     1,
		RegisterPort: port,
		RegisterTime: uint64(time.Now().UnixMilli()),
	}
}

// VCIChargingHeart
// ----------------------------------------------------------------
// VCI Charging 充电心跳帧 0x06
// ----------------------------------------------------------------
func VCIChargingHeart(gunId, hBeatCnt, hBeatPeriod uint32, faultList []*fsmvci.ChargerFaultState) *fsmvci.VCIChargerHeartBeatInfo {
	return &fsmvci.VCIChargerHeartBeatInfo{
		ChargingID:      gunId,
		HeartbeatCnt:    hBeatCnt,
		TimeNow:         uint64(time.Now().UnixMilli()),
		HeartBeatPeriod: hBeatPeriod,
		FaultListLong:   uint32(len(faultList)),
		GunHandShake:    VCIGunHandShake(),
		GunVerifier:     VCIGunVerifier(),
		GunBMSconfig:    VCIGunBMSconfig(),
		GunCharging:     VCIGunCharging(),

		GunTimeout:        VCIGunTimeout(),
		GunChargingEnd:    VCIGunChargingEnd(),
		ReConnect:         VCIReConnect(),
		FaultList:         faultList,
		ChargingReviveMsg: VCIChargingReviveMsg(gunId),
	}
}

// VCIGunHandShake
// ----------------------------------------------------------------
// 枪头充电阶段状态信息 (0x06)
// ----------------------------------------------------------------
func VCIGunHandShake() *fsmvci.BMSHandShake {
	protocMap, err := ReadFile("./Conf/protoc.json", "VCI/GunHandShake")
	gbTProtoVersion := ""
	if err == nil {
		gbTProtoVersion = protocMap["GBTProtoVersion"].(string)
	}

	// 随机数产生
	VoltageRated := RandValue(len(conf.Float64Value))
	return &fsmvci.BMSHandShake{
		BmsVolMaxAllowed: conf.Float64Value[VoltageRated],
		GBTProtoVersion:  gbTProtoVersion,
	}
}

// VCIGunVerifier
// ----------------------------------------------------------------
// 辨识阶段BMS信息（辨识充电桩）(0x06)
// ----------------------------------------------------------------
func VCIGunVerifier() *fsmvci.BMSVerification {
	protocMap, err := ReadFile("./Conf/protoc.json", "VCI/GunVerifier")

	var batteryType uint32 = 0
	var batterySN uint32 = 0
	var batterChargeCnt uint32 = 0
	var propertyRight uint32 = 0
	var chargerVerifyResult uint32 = 0
	var chargerNo uint32 = 0

	batProductor, batProduceDate, bmsVersion, bmsVIN, chargerArea := "", "", "", "", ""

	// 随机数产生
	length := len(conf.Float32Value)
	VoltageRated := RandValue(length)
	CapacityRated := RandValue(length)

	if err == nil {
		batteryType = uint32(protocMap["BatteryType"].(float64))
		batterySN = uint32(protocMap["BatterySN"].(float64))
		batterChargeCnt = uint32(protocMap["BatterChargeCnt"].(float64))
		propertyRight = uint32(protocMap["PropertyRight"].(float64))
		chargerVerifyResult = uint32(protocMap["ChargerVerifyResult"].(float64))
		chargerNo = uint32(protocMap["ChargerNo"].(float64))

		batProductor, batProduceDate = protocMap["BatProductor"].(string), protocMap["BatProduceDate"].(string)
		bmsVersion, bmsVIN = protocMap["BmsVersion"].(string), protocMap["BmsVIN"].(string)
		chargerArea = protocMap["ChargerArea"].(string)
	}
	return &fsmvci.BMSVerification{
		BatteryType:         batteryType,
		BatterySN:           batterySN,
		BatterChargeCnt:     batterChargeCnt,
		PropertyRight:       propertyRight,
		CapacityRated:       conf.Float32Value[VoltageRated],
		VoltageRated:        conf.Float32Value[CapacityRated],
		BatProductor:        batProductor,
		BatProduceDate:      batProduceDate,
		BmsVersion:          bmsVersion,
		BmsVIN:              bmsVIN,
		ChargerVerifyResult: chargerVerifyResult,
		ChargerNo:           chargerNo,
		ChargerArea:         chargerArea,
	}
}

// VCIGunBMSconfig
// ----------------------------------------------------------------
// 参数配置阶段BMS信息 (0x06)
// ----------------------------------------------------------------
func VCIGunBMSconfig() *fsmvci.BMSConfig {
	length := len(conf.Float32Value)
	return &fsmvci.BMSConfig{
		MonoVolMaxAllowed:  conf.Float32Value[RandValue(length)],
		CurAllowedMax:      conf.Float32Value[RandValue(length)],
		TotalNominalEnergy: conf.Float32Value[RandValue(length)],
		VolAllowedMax:      conf.Float32Value[RandValue(length)],
		TempAllowedMax:     conf.Float32Value[RandValue(length)],
		StartSOC:           conf.Float32Value[RandValue(length)],
		VolBatNow:          conf.Float32Value[RandValue(length)],
		VolChargerMax:      conf.Float32Value[RandValue(length)],
		VolChargerMin:      conf.Float32Value[RandValue(length)],
		CurChargerMax:      conf.Float32Value[RandValue(length)],
		CurChargerMin:      conf.Float32Value[RandValue(length)],
		BmsReady:           uint32(RandValue(2)),
		ChargerReady:       uint32(RandValue(2)),
	}
}

// VCIGunCharging
// ----------------------------------------------------------------
// 充电阶段BMS信息 (0x06)
// ----------------------------------------------------------------
func VCIGunCharging() *fsmvci.BMSCharging {
	length := len(conf.Uint32Value)
	return &fsmvci.BMSCharging{
		ChargeMode:           fsmvci.ChargingMode(RandValue(2) + 1),
		HeatingMode:          conf.Uint32Value[RandValue(length)],
		TotalChgTime:         conf.Uint32Value[RandValue(length)],
		MonoBatVolMaxCode:    conf.Uint32Value[RandValue(length)],
		TimeRemain:           conf.Uint32Value[RandValue(length)],
		VolMaxGroupNum:       conf.Uint32Value[RandValue(length)],
		TempMaxCode:          conf.Uint32Value[RandValue(length)],
		TempMinCode:          conf.Uint32Value[RandValue(length)],
		MonoBatVolMinCode:    conf.Uint32Value[RandValue(length)],
		VolDemand:            conf.Float32Value[RandValue(length)],
		CurDemand:            conf.Float32Value[RandValue(length)],
		VolMeasured:          conf.Float32Value[RandValue(length)],
		CurMeasured:          conf.Float32Value[RandValue(length)],
		MonoBatVolMax:        conf.Float32Value[RandValue(length)],
		SocNow:               conf.Float32Value[RandValue(length)],
		MonoBatVolMin:        conf.Float32Value[RandValue(length)],
		TempMax:              conf.Float32Value[RandValue(length)],
		TempMin:              conf.Float32Value[RandValue(length)],
		MonoBatVolOver:       conf.Uint32Value[RandValue(2)],
		MonoBatVolUnder:      conf.Uint32Value[RandValue(2)],
		SocOver:              conf.Uint32Value[RandValue(2)],
		SocUnder:             conf.Uint32Value[RandValue(2)],
		BatCurOver:           conf.Uint32Value[RandValue(2)],
		BatTempOver:          conf.Uint32Value[RandValue(2)],
		InsulationAbnormal:   conf.Uint32Value[RandValue(2)],
		OutConnectedAbnormal: conf.Uint32Value[RandValue(2)],
		BmsAllowchg:          conf.Uint32Value[RandValue(2)],
		ChargerAllowchg:      conf.Uint32Value[RandValue(2)],
	}
}

// VCIGunTimeout
// ----------------------------------------------------------------
// BMS 超时阶段报文 (0x06)
// ----------------------------------------------------------------
func VCIGunTimeout() *fsmvci.BMSTimeout {
	return &fsmvci.BMSTimeout{
		BmsErrorFrame:     conf.VCIFaultList[RandValue(13)+1],
		ChargerErrorFrame: conf.VCIFaultList[RandValue(13)+13],
	}
}

// VCIGunChargingEnd
// ----------------------------------------------------------------
// BmsChargeFinish-充电结束阶段BMS信息 (0x06)
// ----------------------------------------------------------------
func VCIGunChargingEnd() *fsmvci.BMSChargingEnd {
	length := len(conf.Float32Value)
	return &fsmvci.BMSChargingEnd{
		EndSOC:             conf.Float32Value[RandValue(length)],
		MonoBatVolMin:      conf.Float32Value[RandValue(length)],
		MonoBatVolMax:      conf.Float32Value[RandValue(length)],
		BatTempMin:         conf.Float32Value[RandValue(length)],
		BatTempMax:         conf.Float32Value[RandValue(length)],
		BmsStopReason:      conf.VCIFaultList[RandValue(12)+1],
		BmsFaultReason:     conf.VCIFaultList[RandValue(12)+1],
		BmsErrorReason:     conf.VCIFaultList[RandValue(12)+1],
		ChargerStopReason:  conf.VCIFaultList[RandValue(13)+16],
		ChargerFaultReason: conf.VCIFaultList[RandValue(13)+16],
		ChargerErrorReason: conf.VCIFaultList[RandValue(13)+16],
	}
}

// VCIReConnect
// ----------------------------------------------------------------
// BMS重连事件 (0x06)
// ----------------------------------------------------------------
func VCIReConnect() *fsmvci.BMSReConnectEvent {
	return &fsmvci.BMSReConnectEvent{
		TimeOutState:   uint32(RandValue(2)),
		BmsTimeoutType: fsmvci.BMSTimeoutEnum(int32(RandValue(7) + 1)),
		ReconnectCnt:   uint32(RandValue(7) + 1),
		NextState:      uint32(RandValue(4) + 1),
	}
}

// VCIChargingReviveMsg
// ----------------------------------------------------------------
// 充电线程恢复数据 (0x06)
// ----------------------------------------------------------------
func VCIChargingReviveMsg(gunId uint32) *fsmvci.ChargerRevive {
	lengthVCI := len(conf.VCIFaultList)
	lengthFloat32 := len(conf.Float32Value)
	return &fsmvci.ChargerRevive{
		ChargerID:      gunId,
		IsCharging:     conf.Uint32Value[RandValue(2)],
		IsVINStart:     conf.Uint32Value[RandValue(2)],
		FaultState1:    conf.VCIFaultList[RandValue(lengthVCI)],
		FaultState2:    conf.VCIFaultList[RandValue(lengthVCI)],
		FaultState3:    conf.VCIFaultList[RandValue(lengthVCI)],
		BmsCommState:   uint32(RandValue(5) + 1),
		BmsRecvState:   uint32(RandValue(5) + 1),
		BmsType:        uint32(RandValue(2) + 1),
		BmsTimeoutCnt:  uint32(RandValue(5) + 1),
		ElockState:     uint32(RandValue(2) + 1),
		AuxPowerState:  uint32(RandValue(2) + 1),
		BmsCurrentMax:  conf.Float32Value[RandValue(lengthFloat32)],
		BmsVoltageMax:  conf.Float32Value[RandValue(lengthFloat32)],
		CellVoltageMax: conf.Float32Value[RandValue(lengthFloat32)],
		CellTempMax:    conf.Float32Value[RandValue(lengthFloat32)],
		InsultState:    uint32(RandValue(2) + 1),
		InsultResult:   uint32(RandValue(3) + 1),
		InsultVoltage:  conf.Float32Value[RandValue(lengthFloat32)],
	}
}

/* VCI Charging 充电心跳帧 END*/

/* Rt数据帧 START*/

// VCIChargerRTInfo
// ----------------------------------------------------------------
// Rt数据帧 0x08
// ----------------------------------------------------------------
func VCIChargerRTInfo(gunId, pushCnt, rtPeriod uint32, faultList []*fsmvci.ChargerFaultState) *fsmvci.VCIChargerRTInfo {
	return &fsmvci.VCIChargerRTInfo{
		ChargerID:     gunId,
		PushCnt:       pushCnt,
		RtPeriod:      rtPeriod,
		ApplyMsg:      VCIApplyMsg(),
		FaultListLong: uint32(len(faultList)),
		ConnectEvent:  VCIConnectEvent(),
		FaultList:     faultList,
	}
}

// VCIApplyMsg
// ----------------------------------------------------------------
// 需求上传信息 (0x08)
// ----------------------------------------------------------------
func VCIApplyMsg() *fsmvci.GunApplyInfo {
	length := len(conf.Float32Value)
	return &fsmvci.GunApplyInfo{
		VoltageApplied:   conf.Float32Value[RandValue(length)],
		CurrentApplied:   conf.Float32Value[RandValue(length)],
		VoltagePTP:       conf.Float32Value[RandValue(length)],
		CurrentPTP:       conf.Float32Value[RandValue(length)],
		ConnectorApplied: conf.Uint32Value[RandValue(2)],
	}
}

// VCIHaltMsg
// ----------------------------------------------------------------
// 需求上传信息 (0x08)
// ----------------------------------------------------------------
func VCIHaltMsg() *fsmvci.GunCaredInfo {
	length := len(conf.Float32Value)
	return &fsmvci.GunCaredInfo{
		AllocaOK:       conf.Uint32Value[RandValue(2)],
		OutConnectorFb: conf.Uint32Value[RandValue(2)],
		MeterVoltage:   conf.Float32Value[RandValue(length)],
		MeterCurrent:   conf.Float32Value[RandValue(length)],
		BatVoltage:     conf.Float32Value[RandValue(length)],
		ModVoltage:     conf.Float32Value[RandValue(length)],
		MeterEnergy:    conf.Float64Value[RandValue(len(conf.Float64Value))],
	}
}

// VCIConnectEvent
// ----------------------------------------------------------------
// BMS重连事件 (0x08)
// ----------------------------------------------------------------
func VCIConnectEvent() *fsmvci.BMSReConnectEvent {
	return &fsmvci.BMSReConnectEvent{
		TimeOutState:   uint32(RandValue(2)),
		BmsTimeoutType: fsmvci.BMSTimeoutEnum(RandValue(7) + 1),
		ReconnectCnt:   uint32(RandValue(10)),
		NextState:      uint32(RandValue(5)),
	}
}

/* Rt数据帧 END*/

/* PMM 功率矩阵线程 START*/

// PMMPowerMatrixLogin
// ----------------------------------------------------------------
// 功率矩阵线程 注册信息帧 0x00
// ----------------------------------------------------------------
func PMMPowerMatrixLogin(state int) *fsmpmm.PowerMatrixLogin {
	// 从config读取默认的配置信息
	protocMap, err := ReadFile("./Conf/protoc.json", "PMM/Common")

	var mainContactorAmount uint32 = 0
	var matrixContactorAmount uint32 = 0
	var adModuleAmount uint32 = 0
	powerModuleProtoVersion, powerModuleVendor := "", ""
	if err == nil {
		mainContactorAmount = uint32(protocMap["MainContactorAmount"].(float64))
		matrixContactorAmount = uint32(protocMap["MatrixContactorAmount"].(float64))
		adModuleAmount = uint32(protocMap["ADModuleAmount"].(float64))
		powerModuleProtoVersion = protocMap["PowerModuleProtoVersion"].(string)
		powerModuleVendor = protocMap["PowerModuleVendor"].(string)
	}
	return &fsmpmm.PowerMatrixLogin{
		MainContactorAmount:     mainContactorAmount,
		MatrixContactorAmount:   matrixContactorAmount,
		ADModuleAmount:          adModuleAmount,
		SelfCheckRul:            fsmpmm.SelfCheckType(state),
		PowerModuleProtoVersion: powerModuleProtoVersion,
		PowerModuleVendor:       powerModuleVendor,
	}
}

// PMMADModuleLogin
// ----------------------------------------------------------------
// 功率矩阵线程 模块注册信息 0x02
// ----------------------------------------------------------------
func PMMADModuleLogin(alarmList []*fsmpmm.ADModuleAlarm, alarmDataList []*fsmpmm.AlarmDataType) *fsmpmm.ADModuleLogin {
	protocMap, err := ReadFile("./Conf/protoc.json", "PMM/Common")
	var adModuleAmount uint32 = 0
	if err == nil {
		adModuleAmount = uint32(protocMap["ADModuleAmount"].(float64))
	}
	adModuleAList := make([]*fsmpmm.ADModuleAttr, 0)
	adModuleParam := make([]*fsmpmm.ADModuleParam, 0)
	for i := uint32(0); i < adModuleAmount; i++ {
		adModuleAList = append(adModuleAList, PMMADModuleAttr(i))
		adModuleParam = append(adModuleParam, PMMADModuleParam(i))
	}

	return &fsmpmm.ADModuleLogin{
		ADModuleAmount: adModuleAmount,
		ADModuleAList:  adModuleAList,
		ADModulePList:  adModuleParam,
		AlarmList:      alarmList,
		AlarmDataList:  alarmDataList,
	}
}

// PMMADModuleAttr
// ----------------------------------------------------------------
// 功率矩阵线程 模块注册参数描述 (0x02)
// ----------------------------------------------------------------
func PMMADModuleAttr(id uint32) *fsmpmm.ADModuleAttr {
	protocMap, err := ReadFile("./Conf/protoc.json", "PMM/ADModuleAttr")
	DCModuleSN, SoftVersion, HardVersion := "", "", ""

	if err == nil {
		DCModuleSN = protocMap["DCModuleSN"].(string)
		SoftVersion = protocMap["SoftVersion"].(string)
		HardVersion = protocMap["HardVersion"].(string)
	}
	return &fsmpmm.ADModuleAttr{
		ID:            id,
		CurrMax:       conf.Float32Value[RandValue(len(conf.Float32Value))],
		LimitPower:    conf.Float32Value[RandValue(len(conf.Float32Value))],
		VolMax:        conf.Float32Value[RandValue(len(conf.Float32Value))],
		VolMin:        conf.Float32Value[RandValue(len(conf.Float32Value))],
		RatedVol:      conf.Float32Value[RandValue(len(conf.Float32Value))],
		RatedPower:    conf.Float32Value[RandValue(len(conf.Float32Value))],
		RatedCurr:     conf.Float32Value[RandValue(len(conf.Float32Value))],
		RatedInputVol: conf.Float32Value[RandValue(len(conf.Float32Value))],
		DCModuleSN:    DCModuleSN,
		SoftVersion:   SoftVersion,
		HardVersion:   HardVersion,
	}
}

// PMMADModuleParam
// ----------------------------------------------------------------
// 功率矩阵线程 模块运行实时参数（设备管理用）(0x02)
// ----------------------------------------------------------------
func PMMADModuleParam(id uint32) *fsmpmm.ADModuleParam {
	length := len(conf.Float32Value)
	lengthUtiles := len(conf.Uint32Value)
	return &fsmpmm.ADModuleParam{
		ID:                    id,
		OutVol:                conf.Float32Value[RandValue(length)],
		OutCurr:               conf.Float32Value[RandValue(length)],
		LimitCurr:             conf.Float32Value[RandValue(length)],
		IntputVolA:            conf.Float32Value[RandValue(length)],
		IntputVolB:            conf.Float32Value[RandValue(length)],
		IntputVolC:            conf.Float32Value[RandValue(length)],
		IntputCurrA:           conf.Float32Value[RandValue(length)],
		IntputCurrB:           conf.Float32Value[RandValue(length)],
		IntputCurrC:           conf.Float32Value[RandValue(length)],
		IntputCurrN:           conf.Float32Value[RandValue(length)],
		ActivePower:           conf.Float32Value[RandValue(length)],
		ReactivePower:         conf.Float32Value[RandValue(length)],
		PowerFactor:           conf.Float32Value[RandValue(length)],
		VoltUnbalanceRate:     conf.Float32Value[RandValue(length)],
		CurrUnbalanceRate:     conf.Float32Value[RandValue(length)],
		ForwardActiveEnergy:   conf.Float32Value[RandValue(length)],
		ReverseActiveEnergy:   conf.Float32Value[RandValue(length)],
		ForwardReactiveEnergy: conf.Float32Value[RandValue(length)],
		ReverseReactiveEnergy: conf.Float32Value[RandValue(length)],
		AmbientTemp:           conf.Float32Value[RandValue(length)],
		SensorTemp:            conf.Float32Value[RandValue(length)],
		SystemType:            conf.Uint32Value[RandValue(lengthUtiles)],
		RunTimeCount:          conf.Uint32Value[RandValue(lengthUtiles)],
		OnOffCunt:             conf.Uint32Value[RandValue(lengthUtiles)],
		PositionStatus:        conf.Uint32Value[RandValue(2)],
	}
}

// PMMADModuleAlarm
// ----------------------------------------------------------------
// 功率矩阵线程 故障告警描述 (0x02)
// ----------------------------------------------------------------
func PMMADModuleAlarm(key int) *fsmpmm.ADModuleAlarm {
	alarmTime := time.Now().UnixMilli()
	downTime, _ := time.ParseDuration("5s")
	downTime1 := time.Now().Add(5 * downTime).UnixMilli()

	return &fsmpmm.ADModuleAlarm{
		AlarmType:     fsmpmm.AlarmTypeEnum(key),
		AlarmState:    fsmpmm.AlarmStateEnum(RandValue(3)),
		AlarmTime:     uint32(alarmTime),
		AlarmDownTime: uint32(downTime1),
	}
}

// PMMAlarmDataType
// ----------------------------------------------------------------
// 功率矩阵线程 复活告警信号描述 (0x02)
// ----------------------------------------------------------------
func PMMAlarmDataType(id uint32, key []int) *fsmpmm.AlarmDataType {
	return &fsmpmm.AlarmDataType{
		ID:           id,
		AlarmState40: uint32(key[0]),
		AlarmState42: uint32(key[1]),
		AlarmState43: uint32(key[2]),
	}
}

func PMMADModuleOnOffState() fsmpmm.ADModuleOnOffStateEnum {
	state := RandValue(8)
	return fsmpmm.ADModuleOnOffStateEnum(state)
}

// PMMHeartbeatReq
// ----------------------------------------------------------------
// 功率矩阵线程 心跳数据 0x04
// ----------------------------------------------------------------
func PMMHeartbeatReq(hBeatCtr, interval uint32, cTime uint64, mainList []*fsmpmm.MainStatus,
	matrixList []*fsmpmm.MatrixStatus, adModuleList []*fsmpmm.ADModuleAttr,
	param []*fsmpmm.ADModuleParam, adModuleAlarm []*fsmpmm.ADModuleAlarm) *fsmpmm.PMMHeartbeatReq {
	return &fsmpmm.PMMHeartbeatReq{
		HeartbeatCtr:  hBeatCtr,
		Interval:      interval,
		CurrentTime:   cTime,
		MainList:      mainList,
		MatrixList:    matrixList,
		ADModuleList:  adModuleList,
		ADModulePList: param,
		AlarmList:     adModuleAlarm,
	}
}

/* PMM 功率矩阵线程 END*/

/* PMM主接触器线程 START */

// PMMMainContactorHeartbeatReq
// ----------------------------------------------------------------
// 心主接触器线程 跳数据 0x06 alarmAnsList由API传入
// ----------------------------------------------------------------
func PMMMainContactorHeartbeatReq(id, hBeatCtr, interval uint32, cTime uint64,
	mainMode fsmpmm.ContactorStateEnum, matrixId, adModelId []uint32) *fsmpmm.MainContactorHeartbeatReq {
	length := len(conf.Float32Value)
	return &fsmpmm.MainContactorHeartbeatReq{
		ID:           id,
		HeartbeatCtr: hBeatCtr,
		Interval:     interval,
		CurrentTime:  cTime,
		BatVol:       conf.Float32Value[RandValue(length)],
		ModVol:       conf.Float32Value[RandValue(length)],
		MainMode:     mainMode,
		MatrixID:     matrixId,
		ADModuleID:   adModelId,
	}
}

// PMMMainContactorRTpush
// ----------------------------------------------------------------
// RT数据 （0x08）
// ----------------------------------------------------------------
func PMMMainContactorRTpush(id, rtPushCtr, interval uint32,
	mainMode fsmpmm.ContactorStateEnum, mainAlarmList fsmpmm.FaultStopEnum) *fsmpmm.MainContactorRTpush {
	return &fsmpmm.MainContactorRTpush{
		ID:            id,
		RTpushCtr:     rtPushCtr,
		Interval:      interval,
		MainMode:      mainMode,
		MainAlarmList: mainAlarmList,
	}
}

/* PMM主接触器通讯帧 END */

/* OHP订单流水线注册信息帧 START */

// OHPOrderPipelineLogin
// ----------------------------------------------------------------
// ohp 注册 0x00
// ----------------------------------------------------------------
func OHPOrderPipelineLogin(state int) *fsmohp.OrderPipelineLogin {
	protocMap, err := ReadFile("./Conf/protoc.json", "OHP/Common")
	orderPipelineProtoVersion, orderPipelineVendor := "", ""
	meterCount := uint32(0)
	if err == nil {
		orderPipelineProtoVersion = protocMap["OrderPipelineProtoVersion"].(string)
		orderPipelineVendor = protocMap["OrderPipelineVendor"].(string)
		meterCount = uint32(protocMap["MeterCount"].(float64))
	}

	return &fsmohp.OrderPipelineLogin{
		OrderPipelineProtoVersion: orderPipelineProtoVersion,
		OrderPipelineVendor:       orderPipelineVendor,
		SelfCheckRul:              fsmohp.SelfCheckType(state),
		MeterCount:                meterCount,
	}
}

// OHPOrderPipelineHeartbeatReq
// ----------------------------------------------------------------
// ohp 心跳数据 0x02
// ----------------------------------------------------------------
func OHPOrderPipelineHeartbeatReq(hbeatCtr, interval uint32, cTime uint64,
	pipeLine []*fsmohp.OrderPipelineState, moduleLine []*fsmohp.SettlementModuleState,
) *fsmohp.OrderPipelineHeartbeatReq {

	return &fsmohp.OrderPipelineHeartbeatReq{
		HeartbeatCtr:      hbeatCtr,
		PipelineStateSize: uint32(len(pipeLine)),
		PipelineState:     pipeLine,
		ModuleStateSize:   uint32(len(moduleLine)),
		ModuleState:       moduleLine,
		CurrentTime:       cTime,
		Interval:          interval,
	}
}

// OrderPipelineState
// ----------------------------------------------------------------
// ohp 心跳数据 (0x02) PipelineState 流水线状态列表
// ----------------------------------------------------------------
func OrderPipelineState(meterId uint32, moduleID []fsmohp.SettlementModuleEnum,
	runningState *fsmohp.RuningOrderState) *fsmohp.OrderPipelineState {
	alarmAnsList := make([]fsmohp.AlarmTypeEnum, 0)

	refreshTime := uint64(0)
	minute := time.Now().Minute()
	if minute == 0 || minute == 30 {
		refreshTime = uint64(time.Now().UnixMilli())
	}
	length := len(conf.Float64Value)

	ohpAlarm := []string{"ohp_alarm_model3", "ohp_alarm_model5"}
	faultName := ohpAlarm[RandValue(2)]
	listIndex := database.Redis.LLen(context.Background(), faultName).Val()
	intSlice, _ := Redis2Data(faultName, listIndex)

	for _, v := range intSlice {
		alarmAnsList = append(alarmAnsList, fsmohp.AlarmTypeEnum(v))
	}

	return &fsmohp.OrderPipelineState{
		MeterID:      meterId,
		ModuleIDSize: uint32(len(moduleID)),
		ModuleID:     moduleID,
		OnLineState:  conf.BoolValue[RandValue(2)],
		LockState:    conf.BoolValue[RandValue(2)],
		RuningState:  runningState,
		MeterStateRefresh: &fsmohp.MeterState{
			MeterWReadOut: conf.Float64Value[RandValue(length)],
			MeterIReadOut: conf.Float64Value[RandValue(length)],
			MeterVReadOut: conf.Float64Value[RandValue(length)],
			MeterOffLine:  conf.BoolValue[RandValue(2)],
			MeterCheck:    conf.BoolValue[RandValue(2)],
			RefreshTime:   refreshTime,
		},
		AlarmAnsListSize: uint32(len(alarmAnsList)),
		AlarmAnsList:     alarmAnsList,
	}
}

func RuningOrderState(moduleId int, uuid []uint64, runningRateList []*fsmohp.OrderRate) *fsmohp.RuningOrderState {
	return &fsmohp.RuningOrderState{
		ModuleID: fsmohp.SettlementModuleEnum(moduleId),
		OrderUUID: &fsmohp.UUIDValue{
			Value0: uuid[0],
			Value1: uuid[1],
		},
		StartMeterReadOut:  conf.Float64Value[RandValue(len(conf.Float64Value))],
		NowMeterReadOut:    conf.Float64Value[RandValue(len(conf.Float64Value))],
		RuningRateListSize: uint32(len(runningRateList)),
		RuningRateList:     runningRateList,
	}
}

// SettlementModuleState
// ----------------------------------------------------------------
// ohp 心跳数据 (0x02) ModuleState 订单结算通讯模块状态列表
// ----------------------------------------------------------------
func SettlementModuleState(moduleId int32) *fsmohp.SettlementModuleState {
	return &fsmohp.SettlementModuleState{
		ModuleID:              fsmohp.SettlementModuleEnum(moduleId),
		OffLineStrategy:       fsmohp.OrderStrategyEnum(RandValue(6) + 1),
		NormalStrategy:        fsmohp.OrderStrategyEnum(RandValue(6) + 1),
		RegisterState:         conf.BoolValue[RandValue(2)],
		PeriodicCommunication: conf.BoolValue[RandValue(2)],
		Jurisdiction:          fsmohp.OrderJurisdictionEnum(RandValue(6)),
	}
}

// OHPOrderPipelineRTpush
// ohp RT数据 (0x04)
func OHPOrderPipelineRTpush(meterID, rtPushCtr, interval uint32,
	moduleID int, sysCtrlList *fsmohp.SysCtrlCmd,
) *fsmohp.OrderPipelineRTpush {

	return &fsmohp.OrderPipelineRTpush{
		MeterID:     meterID,
		ModuleID:    fsmohp.SettlementModuleEnum(moduleID),
		RTpushCtr:   rtPushCtr,
		SysCtrlList: sysCtrlList,
		Interval:    interval,
	}
}

// VCIPramInit
// ----------------------------------------------------------------
// 用于VCI Manager线程注册信息帧·响应 (0x80)
// ----------------------------------------------------------------
func VCIPramInit(gunId, auxType, elockEnable uint32, gunType int32) *fsmvci.VCIGunPrameters {

	protocMap, err := ReadFile("./Conf/protoc.json", "VCI/VCIPram")
	var LimitI float32 = 0.0
	var LimitV float32 = 0.0
	var MaxP float32 = 0.0
	if err == nil {
		LimitI = float32(protocMap["LimitI"].(float64))
		LimitV = float32(protocMap["LimitV"].(float64))
		MaxP = float32(protocMap["MaxP"].(float64))
	}

	return &fsmvci.VCIGunPrameters{
		GunID:          gunId,
		GunType:        fsmvci.GunTypeEnum(gunType),
		CurrentLimited: LimitI,
		VoltageLimited: LimitV,
		PowerMax:       MaxP,
		AuxType:        auxType,
		ELockEnable:    elockEnable,
	}
}

// SysParameterInit
// ----------------------------------------------------------------
// 系统参数配置，从config文件中获取初始化信息
// 用于VCI Manager线程注册信息帧·响应 (0x80)
// ----------------------------------------------------------------
func SysParameterInit() *fsmvci.VCISysParameters {

	protocMap, err := ReadFile("./Conf/protoc.json", "VCI/SysParameter")

	var SysVolMax float32 = 0.0
	var SysCurrMax float32 = 0.0
	var SysVolVMin float32 = 0.0
	var SysVolCMin float32 = 0.0
	var SysCurrMin float32 = 0.0
	if err != nil {
		SysVolMax = float32(protocMap["SysVolMax"].(float64))
		SysCurrMax = float32(protocMap["SysCurrMax"].(float64))
		SysVolVMin = float32(protocMap["SysVolVMin"].(float64))
		SysVolCMin = float32(protocMap["SysVolCMin"].(float64))
		SysCurrMin = float32(protocMap["SysCurrMin"].(float64))
	}

	return &fsmvci.VCISysParameters{
		SysVolMax:         SysVolMax,
		SysCurMax:         SysCurrMax,
		SysConstVolMinVol: SysVolVMin,
		SysConstCurMinVol: SysVolCMin,
		SysCurMin:         SysCurrMin,
	}
}

// SysCtrlConnectStageInit
// ----------------------------------------------------------------
// 系统控制链接阶段初始化
// ----------------------------------------------------------------
func SysCtrlConnectStageInit() *fsmvci.SysCtrlConnectStage {
	return &fsmvci.SysCtrlConnectStage{
		ElockCmd:  1,
		StartCmd:  0,
		StartType: 0,
	}
}

// ChargingRecoverInit
// ----------------------------------------------------------------
// 用于VCI Manager线程注册信息帧·响应 (0x80)
// TODO 初始化全面的数据详情
// ----------------------------------------------------------------
func ChargingRecoverInit(gunId uint32) *fsmvci.ChargerRevive {
	return &fsmvci.ChargerRevive{
		ChargerID:      gunId,
		IsCharging:     1,
		IsVINStart:     0,
		FaultState1:    0,
		FaultState2:    0,
		FaultState3:    0,
		BmsCommState:   0,
		BmsRecvState:   0,
		BmsType:        1,
		BmsTimeoutCnt:  0,
		ElockState:     0,
		AuxPowerState:  0,
		BmsCurrentMax:  250.00,
		BmsVoltageMax:  0,
		CellVoltageMax: 0,
		CellTempMax:    0,
		InsultState:    0,
		InsultResult:   0,
		InsultVoltage:  0,
	}
}

// VCIChargerRegisterResponse
// ----------------------------------------------------------------
// VCI Charger线程的注册数据帧 0x81
// ----------------------------------------------------------------
func VCIChargerRegisterResponse(reConfirm, port uint32) *fsmvci.VCIChargerRegisterResponse {
	return &fsmvci.VCIChargerRegisterResponse{
		ReConfirm: reConfirm,
		RePort:    port,
		ReTime:    uint64(time.Now().UnixMilli()),
	}
}

// FaultStateAtVCI
// ----------------------------------------------------------------
// 实例化故障基础类
// ----------------------------------------------------------------
func FaultStateAtVCI(faultType, faultState int) *fsmvci.ChargerFaultState {
	if faultType == 0 {
		faultType = int(conf.VCIFaultList[RandValue(len(conf.VCIFaultList))])
	}
	if faultState == 0 {
		faultState = RandValue(2) + 1
	}

	return &fsmvci.ChargerFaultState{
		FaultType:      fsmvci.FaultEnum(faultType),
		FaultState:     fsmvci.FaultState(faultState),
		FaultRaiseTime: uint64(time.Now().UnixMilli()),
		FaultDownTime:  0,
	}
}

// Redis2FaultVCI
// ----------------------------------------------------------------
// 从缓存中读取fault数据，并返回故障message的详情数据
// @id message数据的id值
// @faultName redis 存储的数据名称
// @listIndex redis 存储数据中的下标
// Todo 其他配置信息需要修改
// ----------------------------------------------------------------
func Redis2FaultVCI(id, pushCnt, rtPeriod uint32, faultName string, listIndex int64) (*fsmvci.VCIChargerRTInfo, error) {
	intSlice, err := Redis2Data(faultName, listIndex)
	if err != nil {
		return nil, errors.New("查询无数据，请查看Redis是否存在数据\n")
	}

	faultList := make([]*fsmvci.ChargerFaultState, 0)
	for _, v := range intSlice {
		faultList = append(faultList, FaultStateAtVCI(v, 1))
	}
	return &fsmvci.VCIChargerRTInfo{
		ChargerID:     id,
		PushCnt:       pushCnt,
		RtPeriod:      rtPeriod,
		FaultListLong: uint32(len(faultList)),
		ApplyMsg:      VCIApplyMsg(),
		ConnectEvent:  VCIConnectEvent(),
		FaultList:     faultList,
	}, nil
}

func Redis2Data(faultName string, listIndex int64) ([]int, error) {

	// 获取Redis中alarm的总长度
	length := database.Redis.LLen(context.Background(), faultName).Val()
	if listIndex == -1 {
		if length == 0 {
			fmt.Printf("无法从数据库获取故障列表 %s\n", faultName)
		}
		listIndex = RandValue64(length)
	}
	str := database.Redis.LIndex(context.Background(), faultName, listIndex).Val()
	if str == "" {
		return nil, errors.New("查询无数据，请查看Redis是否存在数据\n")
	}
	// redis只能保存string类型
	// 字符串切割 得到[]string{}
	strSlice := strings.Split(str, ",")
	// []string{} to []int{}
	intSlice := StringArr2IntArr(strSlice)
	return intSlice, nil
}

// PMMMainStatus
// ----------------------------------------------------------------
// 主接触器状态描述
// ----------------------------------------------------------------
func PMMMainStatus(id uint32, matrixId, adModuleId []uint32, alarmAnsList []fsmpmm.FaultStopEnum) *fsmpmm.MainStatus {
	length := len(conf.Float32Value)

	return &fsmpmm.MainStatus{
		ID:           id,
		BatVol:       conf.Float32Value[RandValue(length)],
		ModVol:       conf.Float32Value[RandValue(length)],
		MainMode:     fsmpmm.ContactorStateEnum(3),
		MatrixID:     matrixId,
		ADModuleID:   adModuleId,
		AlarmAnsList: alarmAnsList,
	}
}

// PMMMatrixStatus
// ----------------------------------------------------------------
// 阵列接触器状态描述
// ----------------------------------------------------------------
func PMMMatrixStatus(id uint32, alarmAnsList []fsmpmm.ArrayContactorAlarm) *fsmpmm.MatrixStatus {
	return &fsmpmm.MatrixStatus{
		ID:           id,
		MatrixMode:   fsmpmm.ContactorStateEnum(RandValue(8)),
		AlarmAnsList: alarmAnsList,
	}
}

// OHPSettlementModuleEnum
// ----------------------------------------------------------------
// 订单结算通讯模块枚举
// ----------------------------------------------------------------
func OHPSettlementModuleEnum(settlementModule ...fsmohp.SettlementModuleEnum) (settlementModuleEnum []fsmohp.SettlementModuleEnum) {
	for _, v := range settlementModule {
		settlementModuleEnum = append(settlementModuleEnum, v)
	}
	return
}
