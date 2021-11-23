package fsm

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	conf "FSMTestingPlatform/Conf"
	form "FSMTestingPlatform/Form"
	"FSMTestingPlatform/Protoc/fsmohp"
	udp "FSMTestingPlatform/Udp"
	"FSMTestingPlatform/Utils"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
)

// ThreadOHPList
// ----------------------------------------------------------------
// OHP结构体
// ----------------------------------------------------------------
type ThreadOHPList struct {
	muThread   sync.Mutex         // 互斥锁
	MainLive   bool               // 第一线程在线状态
	UUID       []uint64           // 订单UUID
	FeeRate    map[uint32]float64 // 分时费率
	MainChan   chan *udp.Client   // 第一线程的管道
	MainThread *Thread            // 第一线程对象实体
}

// NewThreadOHPList
// ----------------------------------------------------------------
// 构建OHP结构体指针
// ----------------------------------------------------------------
func NewThreadOHPList(main *Thread) *ThreadOHPList {
	uuid, err := uuid.NewUUID()
	if err != nil {
		fmt.Printf("uuid created error:%s\n", err.Error())
	}
	return &ThreadOHPList{
		muThread:   sync.Mutex{},
		MainLive:   false,
		MainChan:   make(chan *udp.Client),
		MainThread: main,
		UUID:       uuidToUint64(uuid),
	}

}

// ThreadOHPInit
// ----------------------------------------------------------------
// 初始化OHP结构体，返回其指针
// TODO address 写入json
// ----------------------------------------------------------------
func ThreadOHPInit() *ThreadOHPList {
	addrMain := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9050,
	}
	mainThread := NewThread(addrMain, 5)
	return NewThreadOHPList(mainThread)
}

// HeartForMain
// ----------------------------------------------------------------
// 检测工作线程、定时发送OHP主线程的心跳
// ----------------------------------------------------------------
func (t *ThreadOHPList) HeartForMain(id uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			// timer.C 管道类型数据。定时器功能，到指定时间会发出数据（time）。
			// 触发消息之后，要重新赋值下次出发的时间
			client.Timer = time.NewTimer(time.Duration(client.Interval) * time.Millisecond)
			if _, ok := t.MainThread.WorkList[id]; !ok {
				// 工作列表中不存在线程信息，心跳中止
				goto HeartDie
			}
			// TODO 发送数据
			client.WriteMsg(0x00, t.OrderPipelineHeartbeatReqInit(id, client))
		}
	}
HeartDie:
	fmt.Printf("订单管线线程断开 %d 心跳被中止 \n", id)
}

// RtStart
// ----------------------------------------------------------------
// 定时发OHP的RT数据
// ----------------------------------------------------------------
func (t *ThreadOHPList) RtStart() {
	// 指针
	for {
		select {
		case <-t.MainThread.RtTimer.C:
			t.MainThread.RtTimer = time.NewTimer(time.Duration(t.MainThread.RtMessageInterval) * time.Millisecond)
			for k, v := range t.MainThread.WorkList {
				if t.MainThread.RtMessageList[k] {
					t.MainThread.WorkList[k].WriteMsg(0x04, t.OrderPipelineRTpushInit(k, Utils.RandValue(6), v))
				}
			}
		}
	}
}

// RecvCommand
// ----------------------------------------------------------------
// 定义Http接口，外部通过API修改线程内部状态
// ----------------------------------------------------------------
func (t *ThreadOHPList) RecvCommand(r *gin.Engine) {

	// 输出gin日志到 gin_http 文件
	logfile, err := os.Create("./gin_OHP.log")
	if err != nil {
		fmt.Println("Could not create log file")
	}
	gin.SetMode(gin.DebugMode)
	gin.DefaultWriter = io.MultiWriter(logfile)

	// http 路由组
	v1 := r.Group("/api/v1")
	{
		v1.GET("ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"code": "0",
				"msg":  "pong",
			})
		})
		order := v1.Group("/ohp-order")
		{
			order.POST("/build", t.buildOrderHttp)
			order.POST("/register", t.registerOrderPipeline)
			order.POST("/change-state", t.heartOrderPipeline)
			order.POST("/change-rtState", t.changeRTStatus)
			order.POST("/delete", t.deleteOrderHttp)
		}
	}
}

// buildOrderHttp
// ----------------------------------------------------------------
// 创建OHP线程
// ----------------------------------------------------------------
func (t *ThreadOHPList) buildOrderHttp(c *gin.Context) {
	// 识别参数数据
	var order form.OHPConfig
	err := c.ShouldBindBodyWith(&order, binding.JSON)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "数据解析错误:" + err.Error(),
		})
		return
	}

	// 异常数据场景 消息发送前端
	if _, ok := t.MainThread.AmountList[order.Id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未定义识别电表Id",
		})
		return
	}
	if _, ok := t.MainThread.WorkList[order.Id]; ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "线程冲突",
		})
		return
	}

	// 启用互斥锁
	t.muThread.Lock()
	if !t.MainLive {
		t.MainLive = true
	}
	// 从全局udp列表 COPY 到 work udp列表中
	t.MainThread.WorkList[order.Id] = t.MainThread.AmountList[order.Id]
	t.MainThread.WorkList[order.Id].WriteMsg(0x00, t.OHPOrderPipelineLogin(1))

	t.muThread.Unlock()
	go t.HeartForMain(order.Id, t.MainThread.WorkList[order.Id])
	go t.MainThread.AmountList[order.Id].ReadMsg()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// registerOrderPipeline
// ----------------------------------------------------------------
// 发送OHP的注册消息
// ----------------------------------------------------------------
func (t *ThreadOHPList) registerOrderPipeline(c *gin.Context) {
	// 识别参数数据
	var register form.OHPRegister
	err := c.ShouldBindBodyWith(&register, binding.JSON)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "数据解析错误:" + err.Error(),
		})
		return
	}
	// 异常数据场景 消息发送前端
	client, ok := t.MainThread.WorkList[register.Id]
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未定义识别电表Id",
		})
		return
	}

	client.WriteMsg(0x00, t.OHPOrderPipelineLogin(register.State))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// heartOrderPipeline
// ----------------------------------------------------------------
// 修改OHP的心跳状态
// ----------------------------------------------------------------
func (t *ThreadOHPList) heartOrderPipeline(c *gin.Context) {
	// 识别参数数据
	var heatBeat form.OHPHeartBeat
	err := c.ShouldBindBodyWith(&heatBeat, binding.JSON)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "数据解析错误:" + err.Error(),
		})
		return
	}
	client, ok := t.MainThread.WorkList[heatBeat.Id]
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处于工作中的线程",
		})
		return
	}

	t.muThread.Lock()
	if heatBeat.Interval > 0 {
		client.Interval = heatBeat.Interval
	}
	client.HeartbeatCtr = heatBeat.HeartbeatCtr
	t.muThread.Unlock()

	client.WriteMsg(0x02, t.OrderPipelineHeartbeatReqInit(heatBeat.Id, client))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// changeRTStatus
// ----------------------------------------------------------------
// 修改OHP的RT数据状态
// ----------------------------------------------------------------
func (t *ThreadOHPList) changeRTStatus(c *gin.Context) {
	ohpRTInfo := form.OHPRTInfo{}
	c.ShouldBindBodyWith(&ohpRTInfo, binding.JSON)

	err := ohpRTInfo.VerifyVCIRTInfo()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  err.Error(),
		})
		return
	}

	t.muThread.Lock()
	defer t.muThread.Unlock()
	if _, ok := t.MainThread.WorkList[ohpRTInfo.GunId]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体Id",
		})
		return
	}
	if ohpRTInfo.HeartBeatPeriod == 0 {
		ohpRTInfo.HeartBeatPeriod = t.MainThread.WorkList[ohpRTInfo.GunId].Interval
	}
	if ohpRTInfo.PushCnt == 0 {
		ohpRTInfo.PushCnt = t.MainThread.WorkList[ohpRTInfo.GunId].RTpushCtr
	}

	t.MainThread.RtMessageList[ohpRTInfo.GunId] = ohpRTInfo.Status
	t.MainThread.RtMessageInterval = ohpRTInfo.HeartBeatPeriod

	t.MainThread.WorkList[ohpRTInfo.GunId].RTpushCtr = ohpRTInfo.PushCnt
	t.MainThread.WorkList[ohpRTInfo.GunId].Interval = ohpRTInfo.HeartBeatPeriod

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "修改成功",
	})
}

// deleteOrderHttp
// ----------------------------------------------------------------
// 删除OHP线程
// ----------------------------------------------------------------
func (t *ThreadOHPList) deleteOrderHttp(c *gin.Context) {
	// 识别参数数据
	var order form.OHPConfig
	c.ShouldBindBodyWith(&order, binding.JSON)

	// 异常数据场景 消息发送前端
	if _, ok := t.MainThread.WorkList[order.Id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未定义识别电表Id",
		})
		return
	}

	t.muThread.Lock()
	// 从工作列表删除
	delete(t.MainThread.WorkList, order.Id)
	if len(t.MainThread.WorkList) == 0 {
		t.MainLive = false
	}
	t.muThread.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// OHPOrderPipelineLogin
// ----------------------------------------------------------------
// 创建OHP的注册数据帧 0x00
// ----------------------------------------------------------------
func (t *ThreadOHPList) OHPOrderPipelineLogin(state int) *fsmohp.OrderPipelineLogin {
	return Utils.OHPOrderPipelineLogin(state)
}

// OrderPipelineHeartbeatReqInit
// ----------------------------------------------------------------
// 创建OHP的心跳周期信息帧 0x02
// ----------------------------------------------------------------
func (t *ThreadOHPList) OrderPipelineHeartbeatReqInit(id uint32, client *udp.Client) *fsmohp.OrderPipelineHeartbeatReq {
	client.HeartbeatCtr++
	hbeatCtr := uint32(client.HeartbeatCtr)
	interval := client.Interval
	cTime := uint64(time.Now().UnixMilli())

	pipeLine := make([]*fsmohp.OrderPipelineState, 0)

	moduleStateList := make([]*fsmohp.SettlementModuleState, 0)
	moduleList := make([]fsmohp.SettlementModuleEnum, 0)
	moduleMap := fsmohp.SettlementModuleEnum_name

	for k, _ := range moduleMap {
		if k == 0 {
			continue
		}
		moduleStateList = append(moduleStateList, Utils.SettlementModuleState(k))
		moduleList = append(moduleList, fsmohp.SettlementModuleEnum(k))
		pipeLine = append(pipeLine, Utils.OrderPipelineState(id, moduleList, Utils.RuningOrderState(int(k), t.UUID, nil)))
	}

	return Utils.OHPOrderPipelineHeartbeatReq(hbeatCtr, interval, cTime, pipeLine, moduleStateList)
}

// OrderPipelineRTpushInit
// ----------------------------------------------------------------
// 创建OHP的RT周期信息帧 0x04
// ----------------------------------------------------------------
func (t *ThreadOHPList) OrderPipelineRTpushInit(meterId uint32, moduleID int, client *udp.Client) *fsmohp.OrderPipelineRTpush {
	client.RTpushCtr++
	rtPushCtr := client.RTpushCtr
	interval := client.Interval
	sysCtrlCmd := t.getSysCtrlList(meterId, Utils.RandValue(2))

	return Utils.OHPOrderPipelineRTpush(meterId, rtPushCtr, interval, moduleID, sysCtrlCmd)
}

// getSysCtrlList
// ----------------------------------------------------------------
// state=1 开机
// state=2 关机
// todo 开机、关机同时发送
// OrderPipelineRTpushInit
// ----------------------------------------------------------------
func (t *ThreadOHPList) getSysCtrlList(id uint32, state int) *fsmohp.SysCtrlCmd {
	if state > 0 {
		return &fsmohp.SysCtrlCmd{
			ID:        id,
			StartCmd:  true,
			StartType: conf.BoolValue[Utils.RandValue(2)],
			ElockCmd:  conf.BoolValue[Utils.RandValue(2)],
		}
	}
	return &fsmohp.SysCtrlCmd{
		ID:       id,
		StopCmd:  true,
		ElockCmd: conf.BoolValue[Utils.RandValue(2)],
	}
}

// uuidToUint64
// ----------------------------------------------------------------
// 创建uuid
// ----------------------------------------------------------------
func uuidToUint64(bytes uuid.UUID) []uint64 {
	bytes1 := bytes[:8]
	bytes2 := bytes[8:]
	uuid1 := uint64(bytes1[0]) | uint64(bytes1[1])<<8 | uint64(bytes1[2])<<16 | uint64(bytes1[3])<<24 | uint64(bytes1[4])<<32 | uint64(bytes1[5])<<40 | uint64(bytes1[6])<<48 | uint64(bytes1[7])<<56
	uuid2 := uint64(bytes2[0]) | uint64(bytes2[1])<<8 | uint64(bytes2[2])<<16 | uint64(bytes2[3])<<24 | uint64(bytes2[4])<<32 | uint64(bytes2[5])<<40 | uint64(bytes2[6])<<48 | uint64(bytes2[7])<<56
	return []uint64{uuid1, uuid2}
}
