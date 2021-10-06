package conf

var VCIMessage = make(map[uint8]string)
var PMMMessage = make(map[uint8]string)
var OHPMessage = make(map[uint8]string)


func init () {
	// Main线程通讯帧
	// VCI主线程注册信息帧(0x00)
	VCIMessage[0x00] = "VehicleChargingInterfaceLogin"
	// VCI主线程注册信息帧.响应(0x80)
	VCIMessage[0x80] = "VehicleChargingInterfaceLoginAns"

	// VCI主线程心跳周期信息帧(0x02)
	VCIMessage[0x02] = "VCImainHeartbeatReq"
	// VCI主线程心跳周期信息帧.响应(0x82)
	VCIMessage[0x82] = "VCImainHeartbeatAns"

	// 插枪线程(father)通讯帧
	// 插枪线程心跳周期信息帧(0x04)
	VCIMessage[0x04] = "VCIPluggedHeartbeatReq"
	// 插枪线程心跳周期信息帧.响应(0x84)
	VCIMessage[0x84] = "VCIPluggedHeartbeatAns"

	// 充电线程(son)通讯帧
	// 充电线程心跳周期信息帧(0x06)
	VCIMessage[0x06] = "VCIChargingHeartbeatReq"
	// 充电线程心跳周期信息帧.响应(0x86)
	VCIMessage[0x86] = "VCIChargingHeartbeatAns"

	// 充电线程realtimepush
	// 充电线程突发上传信息帧(0x08)
	VCIMessage[0x08] = "VCIChargingRTpush"
	// 充电线程realtimepull
	// 充电线程突发接收信息帧(0x88)
	VCIMessage[0x88] = "VCIChargingRTpull"

	//PMM主线程通讯帧
	// 功率矩阵注册信息帧(0x00)
	PMMMessage[0x00] = "PowerMatrixLogin"
	// 功率矩阵注册信息帧.响应(0x80)
	PMMMessage[0x80] = "PowerMatrixLoginAns"

	// 模块注册信息(0x02)
	PMMMessage[0x02] = "ADModuleLogin"
	// 模块注册信息.响应(0x82)
	PMMMessage[0x82] = "ADModuleLoginAns"

	// 心跳状态同步(0x04)
	PMMMessage[0x04] = "PMMHeartbeatReq"
	// 心跳状态同步.响应(0x84)
	PMMMessage[0x84] = "PMMHeartbeatAns"

	//PMM主接触器通讯帧
	// 主接触器线程心跳周期信息帧(0x06)
	PMMMessage[0x06] = "MainContactorHeartbeatReq"
	// 主接触器线程心跳周期信息帧.响应(0x86)
	PMMMessage[0x86] = "MainContactorHeartbeatAns"

	// 主接触器线程realtimepush
	// 主接触器线程突发上传信息帧(0x08)
	PMMMessage[0x08] = "MainContactorRTpush"

	// 主接触器线程realtimepull
	// 主接触器线程突发接收信息帧(0x88)
	PMMMessage[0x88] = "MainContactorRTpull"

	// 订单流水线注册信息帧(0x00)
	OHPMessage[0x00] = "OrderPipelineLogin"
	// 订单流水线注册信息帧.响应(0x80)
	OHPMessage[0x80] = "OrderPipelineLoginAns"

	// 模块注册信息(0x02)
	OHPMessage[0x02] = "ADModuleLogin"
	// 模块注册信息.响应(0x82)
	OHPMessage[0x82] = "ADModuleLoginAns"

	// 心跳状态同步(0x04)
	OHPMessage[0x04] = "PMMHeartbeatReq"
	// 心跳状态同步.响应(0x84)
	OHPMessage[0x84] = "PMMHeartbeatAns"

	//PMM主接触器通讯帧
	// 主接触器线程心跳周期信息帧(0x06)
	OHPMessage[0x06] = "MainContactorHeartbeatReq"
	// 主接触器线程心跳周期信息帧.响应(0x86)
	OHPMessage[0x86] = "MainContactorHeartbeatAns"

	// 主接触器线程realtimepush
	// 主接触器线程突发上传信息帧(0x08)
	OHPMessage[0x08] = "MainContactorRTpush"

	// 主接触器线程realtimepull
	// 主接触器线程突发接收信息帧(0x88)
	OHPMessage[0x88] = "MainContactorRTpull"
}
