package vci

import (
	"TestingPlatform/Utils"
	"TestingPlatform/form"
	"TestingPlatform/protoc/fsmpmm"
	"TestingPlatform/udp"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
)

type ThreadPMMList struct {
	muThread sync.Mutex
	MainLive bool
	FatherLive bool

	MainChan chan *udp.Client
	FatherChan chan *udp.Client

	MainThread *Thread
	FatherThread *Thread
}

func NewThreadPMMList (main, father *Thread) *ThreadPMMList {
	return &ThreadPMMList{
		muThread:    sync.Mutex{},
		MainLive:    false,
		FatherLive:  false,
		MainChan:    make(chan *udp.Client),
		FatherChan:  make(chan *udp.Client),
		MainThread:   main,
		FatherThread: father,
	}
}

func ThreadPMMInit () *ThreadPMMList  {
	addrMain := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9100,
	}
	addrFather := net.UDPAddr{
		IP: net.IPv4(0, 0, 0 ,0),
		Port: 9120,
	}
	mainThread := NewThread(addrMain, 5)
	fatherThread := NewThread(addrFather, 5)

	return NewThreadPMMList(mainThread, fatherThread)
}

// PowerMatrixLoginInit
// 功率矩阵注册信息帧(0x00)
// Init TODO 可以放入配置文件中
func (t *ThreadPMMList) PowerMatrixLoginInit() *fsmpmm.PowerMatrixLogin {
	return &fsmpmm.PowerMatrixLogin{
		MainContactorAmount:     &fsmpmm.Int32Value{Value: 6},
		MatrixContactorAmount:   &fsmpmm.Int32Value{Value: 6},
		ADModuleAmount:          &fsmpmm.Int32Value{Value: 6},
		PowerModuleProtoVersion: "0.1",
		PowerModuleVendor:       "0.1",
		SelfCheckRul:            1,
	}
}

// ADModuleLoginInit
// 模块注册信息(0x02)
func (t *ThreadPMMList) ADModuleLoginInit () *fsmpmm.ADModuleLogin {
	//
	ADModuleAttr := Utils.ADModuleAttrInit(1)
	ADModuleAttrList := make([]*fsmpmm.ADModuleAttr, 0)
	ADModuleAttrList = append(ADModuleAttrList, ADModuleAttr)

	// ADModuleParamInit
	ADModuleParam := Utils.ADModuleParamInit(1)
	ADModuleParamList := make([]*fsmpmm.ADModuleParam, 0)
	ADModuleParamList = append(ADModuleParamList, ADModuleParam)

	// ADModuleAlarmInit
	ADModuleAlarm := Utils.ADModuleAlarmInit()
	ADModuleAlarmList := make([]*fsmpmm.ADModuleAlarm, 0)
	ADModuleAlarmList = append(ADModuleAlarmList, ADModuleAlarm)

	// AlarmDataTypeInit
	AlarmDataType := Utils.AlarmDataTypeInit(1)
	AlarmDataTypeList := make([]*fsmpmm.AlarmDataType, 0)
	AlarmDataTypeList = append(AlarmDataTypeList, AlarmDataType)

	return &fsmpmm.ADModuleLogin{
		ADModuleAmount: &fsmpmm.Int32Value{Value: 6},
		ADModuleAList:  ADModuleAttrList,
		ADModulePList:  ADModuleParamList,
		AlarmList:      ADModuleAlarmList,
		AlarmDataList:  AlarmDataTypeList,
	}
}

// PMMHeartbeatReqInit
// 心跳状态同步(0x04)
func (t *ThreadPMMList) PMMHeartbeatReqInit (Id uint32) *fsmpmm.PMMHeartbeatReq {
	heartBeat := t.MainThread.WorkList[Id].HeartbeatCtr
	heartBeat++

	//
	mainStatus := Utils.MainStatusInit(1)
	mainStatusList := make([]*fsmpmm.MainStatus, 0)
	mainStatusList = append(mainStatusList, mainStatus)

	//
	MatrixStatus := Utils.MatrixStatusInit(1)
	MatrixStatusList := make([]*fsmpmm.MatrixStatus, 0)
	MatrixStatusList = append(MatrixStatusList, MatrixStatus)

	return &fsmpmm.PMMHeartbeatReq{
		HeartbeatCtr:  &fsmpmm.Uint32Value{Value: uint32(heartBeat)},
		MainList:      mainStatusList,
		MatrixList:    MatrixStatusList,
		ADModuleList:  nil,
		ADModulePList: nil,
		AlarmList:     nil,
		CurrentTime:   Utils.FsmpmmCurrentTime(0),
		Interval:      Utils.FsmpmmInterval(500),
	}
}

// MainContactorHeartbeatReqInit
// 主接触器线程心跳周期信息帧(0x06)
func (t *ThreadPMMList) MainContactorHeartbeatReqInit (Id uint32) *fsmpmm.MainContactorHeartbeatReq {
	HeartbeatCtr := t.MainThread.WorkList[Id].HeartbeatCtr
	HeartbeatCtr++
	return &fsmpmm.MainContactorHeartbeatReq{
		ID:            &fsmpmm.Uint32Value{Value: Id},
		HeartbeatCtr:  &fsmpmm.Uint32Value{Value: uint32(HeartbeatCtr)},
		MainMode:      3,
		MatrixID:      nil,
		ADModuleID:    nil,
		BatVol:        &fsmpmm.FloatValue{Value: 5.95},
		ModVol:        nil,
		AlarmAnsList:  0,
		MatrixList:    nil,
		ADModulePList: nil,
		ADModuleAList: nil,
		CurrentTime:   Utils.FsmpmmCurrentTime(0),
		Interval:      Utils.FsmpmmInterval(300),
	}
}

// MainContactorRTpushInit
// 主接触器线程突发上传信息帧(0x08)
func (t *ThreadPMMList) MainContactorRTpushInit (Id uint32) *fsmpmm.MainContactorRTpush {
	return &fsmpmm.MainContactorRTpush{
		ID:              &fsmpmm.Uint32Value{Value: Id},
		RTpushCtr:       nil,
		MainMode:        fsmpmm.ContactorStateEnum(3),
		MainAlarmList:   0,
		MatrixAlarmList: nil,
		ADModuleAList:   nil,
		Interval:        Utils.FsmpmmInterval(0),
	}
}

func (t *ThreadPMMList) HeartForMain (Id uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			if _, ok :=  t.MainThread.WorkList[Id]; !ok {
				goto HeartDie
			}
			client.WriteMsg(0x04, t.PMMHeartbeatReqInit(Id))
		}
	}
	HeartDie:
		fmt.Printf("主接触器断开 %d 心跳被中止 \n", Id)
}

func (t *ThreadPMMList) HeartForFather (Id uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			if _, ok :=  t.FatherThread.WorkList[Id]; !ok {
				goto HeartDie
			}
		}
	}
	HeartDie:
		fmt.Printf("主接触器断开 %d 心跳被中止 \n", Id)
}

func (t *ThreadPMMList) buildMainHttp(c *gin.Context) {
	var contactor = &form.Contactor{}
	c.ShouldBindBodyWith(contactor, binding.JSON)
	id := contactor.Id
	if len(t.MainThread.WorkList) == 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程达到上限",
		})
		return
	}
	if _, ok := t.MainThread.WorkList[id]; ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程冲突",
		})
		return
	}
	if _, ok := t.MainThread.AmountList[id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "未识别枪体",
		})
		return
	}

	t.MainLive = true

	t.MainThread.WorkList[id] = t.MainThread.AmountList[id]
	go t.HeartForMain(id, t.MainThread.WorkList[id])
	go t.MainThread.WorkList[id].ReadMsg()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

func (t *ThreadPMMList) buildFatherHttp(c *gin.Context) {
	var contactor = &form.Contactor{}
	c.ShouldBindBodyWith(contactor, binding.JSON)
	id := contactor.Id
	if len(t.FatherThread.WorkList) == 6 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程达到上限",
		})
		return
	}
	if _, ok := t.FatherThread.WorkList[id]; ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程冲突",
		})
		return
	}
	if _, ok := t.FatherThread.AmountList[id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "未识别枪体",
		})
		return
	}
	t.FatherLive = true
	t.FatherThread.WorkList[id] = t.FatherThread.AmountList[id]
	go t.HeartForFather(id, t.FatherThread.WorkList[id])
	go t.FatherThread.WorkList[id].ReadMsg()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

func(t *ThreadPMMList) deleteMainHttp(c *gin.Context) {
	var contactor = &form.Contactor{}
	id := contactor.Id
	c.ShouldBindBodyWith(id, binding.JSON)
	if _, ok := t.MainThread.WorkList[id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未识别枪体",
		})
		return
	}
	//don't close the udp conn
	//t.MainThread.WorkList[VCIGun.Gun].Conn.Close()
	delete(t.MainThread.WorkList, id)
	if len(t.MainThread.WorkList) == 0 {
		t.MainLive = false
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

func(t *ThreadPMMList) deleteFatherHttp(c *gin.Context) {
	var contactor = &form.Contactor{}
	id := contactor.Id
	c.ShouldBindBodyWith(id, binding.JSON)
	if _, ok := t.FatherThread.WorkList[id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未识别枪体",
		})
		return
	}
	//don't close the udp conn
	//t.MainThread.WorkList[VCIGun.Gun].Conn.Close()
	delete(t.FatherThread.WorkList, id)
	if len(t.FatherThread.WorkList) == 0 {
		t.MainLive = false
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

// 中间件 跨域会先发送 option
func middlewarePost1(c *gin.Context) {
	fmt.Println(c.Request.Method)
	if c.Request.Method == "options" {
		c.JSON(http.StatusOK, gin.H{
			"code" : 0,
			"msg" : "next",
		})
	}
	c.Next()
}

func (t *ThreadPMMList) RecvCommand () {
	logfile, err := os.Create("./gin_http.log")
	if err != nil {
		fmt.Println("Could not create log file")
	}
	gin.SetMode(gin.DebugMode)
	gin.DefaultWriter = io.MultiWriter(logfile)

	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		//ping pone
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code":0, "msg":"pong"})
		})
		//v1.GET("/vci/main/ws", t.MainWS)
		//v1.GET("/vci/father/ws", t.FatherWS)
		//v1.GET("/vci/son/ws", t.SonWS)
		//v1.POST("/get/cmd", t.getSqlMessage)
		//v1.GET("/vci/ws", wsocket.WS)
		r.Use(middlewarePost1)
		vci := v1.Group("/pmm")
		{
			main := vci.Group("/main")
			{
				//main.POST("/register", t.registerMain)
				//main.POST("/heart", t.heartMain)
				main.POST("/build", t.buildMainHttp)
				main.POST("/delete", t.deleteMainHttp)
			}
			father := vci.Group("/father")
			{
				father.POST("/build", t.buildFatherHttp)
				father.POST("/delete", t.deleteFatherHttp)
			}
		}
	}
	r.Run(":10002")
}



