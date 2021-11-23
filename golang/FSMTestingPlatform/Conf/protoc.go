package conf

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

var Float32Value []float32
var Float64Value []float64
var IntValue []int
var Uint32Value []uint32
var BoolValue []bool

var VCIMessage = make(map[uint8]string)
var PMMMessage = make(map[uint8]string)
var OHPMessage = make(map[uint8]string)

// TODO 还需要 PMM、OHP的故障列表

var VCIFaultList []uint32
var PMMAlarmList []uint32
var PMMFaultList []uint32

var OHPFaultList []uint32
var OHPAlarmList []uint32

var VCIRedisListName []string
var PMMRedisListNameAlarm []string
var PMMRedisListNameFault []string
var OHPRedisListNameAlarm []string

var PMMContactorState = []uint32{3, 3, 3, 3, 3, 3}       // 主接触器状态
var PMMADModuleOnOffState = []uint32{1, 1, 1, 1, 1, 1}   // 模块状态
var PMMMatrixContactorState = []uint32{3, 3, 3, 3, 3, 3} // 阵列接触器状态

var GinEngine *gin.Engine

func init() {
	// Float32
	Float32Value = goldenSectionFloat32(-math.SmallestNonzeroFloat32, math.MaxFloat32, 15)
	Float32Value = append(Float32Value, goldenSectionFloat32(-math.MaxFloat32, math.SmallestNonzeroFloat32, 15)...)
	Float32Value = append(Float32Value, []float32{-math.MaxFloat32, -99.9, -math.SmallestNonzeroFloat32, 2e-149, math.SmallestNonzeroFloat32, 0.1, 100.1, 250.1, 500.1, 1099.9, math.MaxFloat32}...)

	// IntValue Uint32Value
	for i := 0; i < 500; i++ {
		IntValue = append(IntValue, i)
		Uint32Value = append(Uint32Value, uint32(i))
	}

	// BoolValue
	BoolValue = []bool{true, false}

	// Float64
	Float64Value = goldenSectionFloat64(-math.SmallestNonzeroFloat64, math.MaxFloat64, 15)
	Float64Value = append(Float64Value, goldenSectionFloat64(-math.MaxFloat64, math.SmallestNonzeroFloat64, 15)...)
	Float64Value = append(Float64Value, []float64{-1000.00, 0.00, 250.50, 500.50, 750.50, 1000.50}...)

	// VCI
	// Main线程通讯帧
	// VCI主线程注册信息帧(0x00)
	VCIMessage[0x00] = "VCIManagerRegisterInfo"
	// VCI主线程注册信息帧.响应(0x80)
	VCIMessage[0x80] = "VCIManagerRegisterReponse"

	// VCI主线程心跳周期信息帧(0x02)
	VCIMessage[0x02] = "VCIManagerHeartBeatInfo"
	// VCI主线程心跳周期信息帧.响应(0x82)
	VCIMessage[0x82] = "VCIManagerHeartBeatReponse"

	// 插枪线程(father)通讯帧
	// 插枪线程心跳周期信息帧(0x04)
	//VCIMessage[0x04] = "VCIPluggedHeartbeatReq"
	//// 插枪线程心跳周期信息帧.响应(0x84)
	//VCIMessage[0x84] = "VCIPluggedHeartbeatAns"

	// 充电线程(son)通讯帧
	// 充电线程心跳周期信息帧(0x06)
	VCIMessage[0x06] = "VCIChargerHeartBeatInfo"
	// 充电线程心跳周期信息帧.响应(0x86)
	VCIMessage[0x86] = "VCIChargerHeartBeatResponse"

	// 充电线程realtimepush
	// 充电线程突发上传信息帧(0x08)
	VCIMessage[0x08] = "VCIChargerRTInfo"
	// 充电线程realtimepull
	// 充电线程突发接收信息帧(0x88)
	VCIMessage[0x88] = "VCIChargerRTRsponse"

	// PMM
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

	// OHP
	// 订单流水线注册信息帧(0x00)
	OHPMessage[0x00] = "OrderPipelineLogin"
	// 订单流水线注册信息帧.响应(0x80)
	OHPMessage[0x80] = "OrderPipelineLoginAns"

	// OHP订单流水线心跳周期信息帧(0x02)
	OHPMessage[0x02] = "OrderPipelineHeartbeatReq"
	// OHP订单流水线心跳周期信息帧.响应(0x82)
	OHPMessage[0x82] = "OrderPipelineHeartbeatAns"

	// 订单流水线突发上传信息帧(0x04)
	OHPMessage[0x04] = "OrderPipelineRTpush"
	// 订单流水线突发接收信息帧(0x84)
	OHPMessage[0x84] = "OrderPipelineRTpull"

	// VCIFaultList
	VCIFaultList = []uint32{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c,
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f,
		0x30, 0x31, 0x32, 0x33,
		0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f,
		0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f,
		0x60,
	}

	// PMMAlarmList
	// 故障告警类型枚举
	PMMAlarmList = []uint32{
		0x01, 0x02, 0x03,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E,
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
	}

	// PMMFaultList
	// 故障码存入
	PMMFaultList = []uint32{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E,
	}

	// 主接触器状态
	PMMContactorState = []uint32{3, 3, 3, 3, 3, 3}
	// 模块状态
	PMMADModuleOnOffState = []uint32{1, 1, 1, 1, 1, 1}
	// 阵列接触器状态
	PMMMatrixContactorState = []uint32{3, 3, 3, 3, 3, 3}

	// OHPFaultList
	OHPAlarmList = []uint32{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
		0x10, 0x11, 0x12,
		0x20, 0x21, 0x22, 0x23,
	}

	// RedisListName
	VCIRedisListName = []string{"vci_fault_model3", "vci_fault_model5"}

	PMMRedisListNameAlarm = []string{"pmm_alarm_model3", "pmm_alarm_model5"}
	PMMRedisListNameFault = []string{"pmm_fault_model3", "pmm_fault_model5"}

	OHPRedisListNameAlarm = []string{"ohp_alarm_model3", "ohp_alarm_model5"}
	// Gin
	GinEngine = gin.Default()
	//ping pone
	GinEngine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "pong"})
	})

	GinEngine.StaticFile("/favicon.ico", "favicon.ico")
}

// goldenSectionFloat32
// Float32
func goldenSectionFloat32(min, max float32, n int) (golden []float32) {
	if n <= 0 {
		return golden
	}
	rangeValue := max - min
	goldenNumber := rangeValue*0.618 + min
	golden = append(golden, goldenNumber)
	minList, maxList := goldenSectionFloat32(min, goldenNumber, n-1), goldenSectionFloat32(goldenNumber, max, n-1)
	golden = append(append(golden, minList...), maxList...)
	return golden
}

// goldenSectionFloat64
// Float64
func goldenSectionFloat64(min, max float64, n int) (golden []float64) {
	if n <= 0 {
		return golden
	}
	rangeValue := max - min
	goldenNumber := rangeValue*0.618 + min
	golden = append(golden, goldenNumber)
	minList, maxList := goldenSectionFloat64(min, goldenNumber, n-1), goldenSectionFloat64(goldenNumber, max, n-1)
	golden = append(append(golden, minList...), maxList...)
	return golden
}
