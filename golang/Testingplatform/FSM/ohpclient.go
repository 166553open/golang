package fsm

import (
	"TestingPlatform/Utils"
	"TestingPlatform/form"
	"TestingPlatform/protoc/fsmohp"
	"TestingPlatform/udp"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type ThreadOHPList struct {
	muThread sync.Mutex
	MainLive bool
	MainChan chan *udp.Client
	MainThread *Thread
}

func NewThreadOHPList (main *Thread) *ThreadOHPList {
	return &ThreadOHPList{
		muThread:    sync.Mutex{},
		MainLive:    false,
		MainChan:    make(chan *udp.Client),
		MainThread:   main,
	}
}

func ThreadOHPInit () *ThreadOHPList  {
	addrMain := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9050,
	}
	mainThread := NewThread(addrMain, 5)
	return NewThreadOHPList(mainThread)
}

// OrderPipelineLoginInit
// TODO init 初始化消息体 protoc 暂时有误，无法生存 .go文件
// OHP订单流水线注册信息帧 0x00
func (t *ThreadOHPList) OrderPipelineLoginInit (client *udp.Client) *fsmohp.OrderPipelineLogin {
	SettlementModuleEnum := make([]fsmohp.SettlementModuleEnum, 0)
	SettlementModuleEnum = append(SettlementModuleEnum, 1,3,4,5)
	return &fsmohp.OrderPipelineLogin{
		OrderPipelineProtoVersion: "0.0.1",
		OrderPipelineVendor:       "0.0.1",
		SelfCheckRul:              1,
		MeterCount:                &fsmohp.Uint32Value{Value: 6},
		ModuleID:                  SettlementModuleEnum,
	}
}

// OrderPipelineHeartbeatReqInit
// OHP订单流水线心跳周期信息帧 0x02
func (t *ThreadOHPList) OrderPipelineHeartbeatReqInit (MeterId uint32, client *udp.Client) *fsmohp.OrderPipelineHeartbeatReq {
	client.HeartbeatCtr++

	RuningOrderState := &fsmohp.RuningOrderState{
		ModuleID:          1,
		OrderUUID:         nil,
		StartMeterReadOut: nil,
		NowMeterReadOut:   nil,
		RuningRateList:    nil,
	}
	orderPipelineState := &fsmohp.OrderPipelineState{
		MeterID:           &fsmohp.Uint32Value{Value: MeterId},
		ModuleID:          Utils.OHPSettlementModuleEnum(1, 3, 4, 5),
		OnLineState:       &fsmohp.BoolEnum{Value: true},
		LockState:         &fsmohp.BoolEnum{Value: true},
		RuningState:       RuningOrderState,
		MeterStateRefresh: nil,
		AlarmAnsList:      nil,
	}
	return &fsmohp.OrderPipelineHeartbeatReq{
		HeartbeatCtr:  &fsmohp.Uint32Value{Value: uint32(client.HeartbeatCtr)},
		PipelineState: []*fsmohp.OrderPipelineState{orderPipelineState},
		ModuleState:   []*fsmohp.SettlementModuleState{},
		CurrentTime:   &fsmohp.DateTimeLong{Time: uint64(time.Now().UnixMilli())},
		Interval:      &fsmohp.Uint32Value{Value: client.Interval},
	}
}

// OrderPipelineRTpushInit 0x04
func (t *ThreadOHPList) OrderPipelineRTpushInit (MeterId uint32, client *udp.Client) *fsmohp.OrderPipelineRTpush {
	client.RTpushCtr++
	return &fsmohp.OrderPipelineRTpush{
		MeterID:     &fsmohp.Uint32Value{Value: MeterId},
		ModuleID:    1,
		RTpushCtr:   &fsmohp.Uint32Value{Value: client.RTpushCtr},
		SysCtrlList: &fsmohp.SysCtrlCmd{
			ID:        &fsmohp.Uint32Value{Value: MeterId},
			StartCmd:  &fsmohp.BoolEnum{Value: true},
			StartType: &fsmohp.BoolEnum{Value: false},
			StopCmd:   &fsmohp.BoolEnum{Value: false},
			ElockCmd:  &fsmohp.BoolEnum{Value: true},
		},
		Interval:    &fsmohp.Uint32Value{Value: client.Interval},
	}
}


func (t *ThreadOHPList) HeartForMain (Id uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			client.Timer = time.NewTimer(time.Duration(t.MainThread.DefaultInterval)*time.Millisecond)
			if _, ok := t.MainThread.WorkList[Id]; !ok {
				goto HeartDie
			}
			// TODO 发送数据
			client.WriteMsg(0x00, t.OrderPipelineHeartbeatReqInit(Id, client))
		}
	}
	HeartDie:
		fmt.Printf("")
}

func (t *ThreadOHPList) buildMainHttp (c *gin.Context) {
	var meter form.Meter
	c.ShouldBindBodyWith(&meter, binding.JSON)
	if _, ok := t.MainThread.AmountList[meter.Id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未定义识别电表Id",
		})
		return
	}
	if _, ok := t.MainThread.WorkList[meter.Id]; ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"线程冲突",
		})
		return
	}
	if !t.MainLive {
		t.MainLive = true
	}
	t.MainThread.WorkList[meter.Id] = t.MainThread.AmountList[meter.Id]
	//TODO send Register
	//t.MainThread.WorkList[meter.Id].WriteMsg()

	go t.HeartForMain(meter.Id, t.MainThread.WorkList[meter.Id])
	go t.MainThread.AmountList[meter.Id].ReadMsg()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

func (t *ThreadOHPList) deleteMainHttp (c *gin.Context) {
	var meter form.Meter
	c.ShouldBindBodyWith(&meter, binding.JSON)
	if _, ok := t.MainThread.WorkList[meter.Id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未定义识别电表Id",
		})
		return
	}
	delete(t.MainThread.WorkList, meter.Id)
	if len(t.MainThread.WorkList) == 0 {
		t.MainLive = false
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

func (t *ThreadOHPList) RecvCommand () {
	logfile, err := os.Create("./gin_PMM.log")
	if err != nil {
		fmt.Println("Could not create log file")
	}
	gin.SetMode(gin.DebugMode)
	gin.DefaultWriter = io.MultiWriter(logfile)

	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		v1.GET("ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"code":"0",
				"msg":"pong",
			})
		})

		v1.POST("/ohp/build", t.buildMainHttp)
		v1.POST("/ohp/delete", t.deleteMainHttp)
	}

}