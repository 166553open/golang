package conf

var ProtoMessage = make(map[uint8]string)

func init () {
	//Main线程通讯帧
	//VCI主线程注册信息帧(0x00)
	ProtoMessage[0x00] = "VehicleChargingInterfaceLogin"
	//VCI主线程注册信息帧.响应(0x80)
	ProtoMessage[0x80] = "VehicleChargingInterfaceLoginAns"

	//VCI主线程心跳周期信息帧(0x02)
	ProtoMessage[0x02] = "VCImainHeartbeatReq"
	//VCI主线程心跳周期信息帧.响应(0x82)
	ProtoMessage[0x82] = "VCImainHeartbeatAns"

	//插枪线程(father)通讯帧
	//插枪线程心跳周期信息帧(0x04)
	ProtoMessage[0x04] = "VCIPluggedHeartbeatReq"
	//插枪线程心跳周期信息帧.响应(0x84)
	ProtoMessage[0x84] = "VCIPluggedHeartbeatAns"

	//充电线程(son)通讯帧
	//充电线程心跳周期信息帧(0x06)
	ProtoMessage[0x06] = "VCIChargingHeartbeatReq"
	//充电线程心跳周期信息帧.响应(0x86)
	ProtoMessage[0x86] = "VCIChargingHeartbeatAns"

	//充电线程realtimepush
	//充电线程突发上传信息帧(0x08)
	ProtoMessage[0x08] = "VCIChargingRTpush"
	//充电线程realtimepull
	//充电线程突发接收信息帧(0x88)
	ProtoMessage[0x88] = "VCIChargingRTpull"
}
