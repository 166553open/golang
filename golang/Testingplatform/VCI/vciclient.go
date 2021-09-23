package vci

import (
	"TestingPlatform/Utils"
	"TestingPlatform/database"
	"TestingPlatform/form"
	"TestingPlatform/pool"
	"TestingPlatform/protoc/fsmvci"
	"TestingPlatform/udp"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Thread struct {
	DefaultInterval uint16
	pool *pool.UDPpool
	States interface{}
	WSocket map[string]*websocket.Conn
	WorkList map[uint32]*udp.Client
	AmountList map[uint32]*udp.Client
}

type ThreadVCIList struct {
	muThread sync.Mutex
	MainLive bool
	FatherLive bool
	SonLive bool

	MainChan chan *udp.Client
	FatherChan chan *udp.Client
	SonChan chan *udp.Client

	MainThread *Thread
	FatherThread *Thread
	SonThread *Thread

}

func NewThread (addr net.UDPAddr, poolSize int) *Thread {
	newPool := pool.NewUdppool(addr, poolSize)
	List := make(map[uint32]*udp.Client)
	for i:=1; i<poolSize+1; i++  {
		client, _ :=udp.NewClient(newPool)
		List[uint32(i)] = client
	}
	return &Thread{
		DefaultInterval: 200,
		pool:            newPool,
		WSocket:         make(map[string]*websocket.Conn),
		WorkList:        make(map[uint32]*udp.Client),
		AmountList:      List,
	}
}

func NewThreadList (main, father, son *Thread) *ThreadVCIList {
	return &ThreadVCIList{
		muThread:     sync.Mutex{},
		MainLive:     false,
		FatherLive:   false,
		SonLive:      false,

		MainChan:	  make(chan *udp.Client),
		FatherChan:	  make(chan *udp.Client),
		SonChan:	  make(chan *udp.Client),
		MainThread:   main,
		FatherThread: father,
		SonThread:    son,
	}
}


func ThreadVCIInit () *ThreadVCIList {
	addrMain := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9000,
	}
	addrFather := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9010,
	}
	addrSon := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9020,
	}
	mainThread := NewThread(addrMain, 1)
	fatherThread := NewThread(addrFather, 6)
	sonThread := NewThread(addrSon, 6)

	thread := NewThreadList(mainThread, fatherThread, sonThread)
	return thread
}

/*公共区方法*/
//fsm cvi 消息体初始化

// VCIMainLoginInit
// 0x00
func (t *ThreadVCIList) VCIMainLoginInit() *fsmvci.VehicleChargingInterfaceLogin  {
	protoc, err := Utils.ReadFile("./conf/protoc.json", "")
	if err != nil {
		return &fsmvci.VehicleChargingInterfaceLogin{
			VehicleChargingProtoVersion: "",
			VehicleChargingVendor:       "",
			SelfCheckRul:                0,
		}
	}
	protocMap := protoc.(map[string]string)
	return &fsmvci.VehicleChargingInterfaceLogin{
		VehicleChargingProtoVersion: protocMap["VehicleChargingProtoVersion"],
		VehicleChargingVendor:       protocMap["MainStateMachineVendor"],
		SelfCheckRul:                0,
	}
}

// VCImainHeartInit
// 0x02
func (t *ThreadVCIList) VCImainHeartInit (client *udp.Client) *fsmvci.VCImainHeartbeatReq {
	gunBaseList := make([]*fsmvci.GunBaseStatus, 0)
	for k, _ := range t.MainThread.WorkList {
		gunBaseStatus := Utils.GunBaseStatusInit(uint32(k), true, false)
		gunBaseList = append(gunBaseList, gunBaseStatus)
	}
	client.Mu.Lock()
	client.HeartbeatCtr++
	client.Mu.Unlock()

	return &fsmvci.VCImainHeartbeatReq{
		HeartbeatCtr: &fsmvci.Uint32Value{Value: uint32(client.HeartbeatCtr)},
		GunBaseList:  gunBaseList,
		CurrentTime:  &fsmvci.DateTimeLong{Time: uint64(time.Now().UnixMilli()) },
		Interval:     &fsmvci.Uint32Value{Value: client.Interval},
	}
}

// VCIPluggedHeartbeatReqInit
// 0x04
func (t *ThreadVCIList) VCIPluggedHeartbeatReqInit (GunId uint32, client *udp.Client) *fsmvci.VCIPluggedHeartbeatReq {
	client.HeartbeatCtr++
	return &fsmvci.VCIPluggedHeartbeatReq{
		ID:           &fsmvci.Uint32Value{Value: GunId},
		HeartbeatCtr: &fsmvci.Uint32Value{Value: uint32(client.HeartbeatCtr)},
		GunStatus:    &fsmvci.GunPluggedStatus{
			AuxPowerDrv: &fsmvci.BoolEnum{Value: true},
			AuxPowerFb:  &fsmvci.BoolEnum{Value: true},
			ElockDrv:    &fsmvci.BoolEnum{Value: true},
			ElockFb:     &fsmvci.BoolEnum{Value: true},
			TPos:        &fsmvci.Int32Value{Value: 50},
			TNeg:        &fsmvci.Int32Value{Value: 60},
		},
		CurrentTime:  &fsmvci.DateTimeLong{Time: uint64(time.Now().UnixMilli())},
		Interval:     &fsmvci.Uint32Value{Value: client.Interval},
	}
}

// VCIChargingHeartbeatReqInit
// 0x06
func (t *ThreadVCIList) VCIChargingHeartbeatReqInit (GunId uint32, client *udp.Client) *fsmvci.VCIChargingHeartbeatReq {
	chargingFaultStateList := make([]*fsmvci.ChargingFaultState, 0)

	chargingFaultStateList = append(chargingFaultStateList, &fsmvci.ChargingFaultState{
		FaultName:     fsmvci.FaultEnum(0),
		FaultType:     fsmvci.FaultState_FaultSustained,
		FaultTime:     nil,
		FaultDownTime: nil,
	})
	client.HeartbeatCtr++
	return &fsmvci.VCIChargingHeartbeatReq{
		ID:                 &fsmvci.Uint32Value{Value: GunId},
		HeartbeatCtr:       &fsmvci.Uint32Value{Value: uint32(client.HeartbeatCtr)},
		GunStatus0:         &fsmvci.BmsShakehands{
			BmsVolMaxAllowed: &fsmvci.DoubleValue{Value: 60.22},
			GBTProtoVersion:  "1.0",
		},
		GunStatus1:         &fsmvci.BmsIdentify{
			BatteryType:     &fsmvci.Int32Value{Value: 1},
			CapacityRated:   &fsmvci.DoubleValue{Value: 60.30},
			VoltageRated:    &fsmvci.DoubleValue{Value: 60.30},
			BatteryVendor:   "WanMa",
			BatterySequence: &fsmvci.Int32Value{Value: 1},
			ProduceDate:     "2022.04.01",
			ChargeCount:     &fsmvci.Int32Value{Value: 0},
			RightIdentifier: &fsmvci.Int32Value{Value: 1},
			BmsVersion:      "1.0",
			BmsAndCarId:     "1",
			BmsVIN:          "yes",
		},
		GunStatus2:         &fsmvci.BmsConfig{
			VIndAllowedMax: &fsmvci.DoubleValue{Value: 60.00},
			IAllowedMax:    &fsmvci.DoubleValue{Value: 50.00},
			EnergyRated:    &fsmvci.DoubleValue{Value: 300.00},
			VAllowedMax:    &fsmvci.DoubleValue{Value: 50.00},
			TAllowedMax:    &fsmvci.DoubleValue{Value: 50.00},
			StartSoc:       &fsmvci.DoubleValue{Value: 50.00},
			VCurrent:       &fsmvci.DoubleValue{Value: 50.00},
			VCOutputMax:    &fsmvci.DoubleValue{Value: 50.00},
			VCOutputMin:    &fsmvci.DoubleValue{Value: 50.00},
			ICOutputMax:    &fsmvci.DoubleValue{Value: 50.00},
			ICOutputMin:    &fsmvci.DoubleValue{Value: 50.00},
		},
		GunStatus3:         &fsmvci.BmsCharging{
			VDemand:         &fsmvci.DoubleValue{Value: 5.00},
			IDemand:         &fsmvci.DoubleValue{Value: 5.00},
			CurrentSoc:      &fsmvci.DoubleValue{Value: 5.00},
			RemainTime:      &fsmvci.DoubleValue{Value: 5.00},
			ChargeMode:      1,
			VMeasure:        &fsmvci.DoubleValue{Value: 5.00},
			IMeasure:        &fsmvci.DoubleValue{Value: 5.00},
			VIndMax:         &fsmvci.DoubleValue{Value: 5.00},
			VIndMaxCode:     &fsmvci.Int32Value{Value: 5},
			VIndMin:         &fsmvci.DoubleValue{Value: 5.00},
			VIndMinCode:     &fsmvci.Int32Value{Value: 5},
			TMax:            &fsmvci.DoubleValue{Value: 5.00},
			TMaxCode:        &fsmvci.Int32Value{Value: 5},
			TMin:            &fsmvci.DoubleValue{Value: 5.00},
			TMinCode:        &fsmvci.Int32Value{Value: 5},
			ChargeAllow:     &fsmvci.BoolEnum{Value: true},
			VIndHigh:        &fsmvci.BoolEnum{Value: false},
			VIndLow:         &fsmvci.BoolEnum{Value: false},
			SoHigh:          &fsmvci.BoolEnum{Value: false},
			SocLow:          &fsmvci.BoolEnum{Value: false},
			IHigh:           &fsmvci.BoolEnum{Value: false},
			THigh:           &fsmvci.BoolEnum{Value: false},
			Insulation:      &fsmvci.BoolEnum{Value: false},
			OutputConnector: &fsmvci.BoolEnum{Value: false},
			VIndMaxGroupNum: &fsmvci.Int32Value{Value: 5},
			HeatingMode:     &fsmvci.Int32Value{Value: 0},
		},
		GunStatus4:         &fsmvci.BmsChargeFinish{
			EndSoc:             &fsmvci.DoubleValue{Value: 6.00},
			VMinIndividal:      &fsmvci.DoubleValue{Value: 4.00},
			VMaxIndividal:      &fsmvci.DoubleValue{Value: 4.00},
			TemperatureMin:     &fsmvci.DoubleValue{Value: 4.00},
			TemperatureMax:     &fsmvci.DoubleValue{Value: 4.00},
			BmsStopReason:      &fsmvci.Int32Value{Value: 4},
			BmsFaultReason:     &fsmvci.Int32Value{Value: 4},
			BmsErrorReason:     &fsmvci.Int32Value{Value: 4},
			ChargerStopReason:  &fsmvci.Int32Value{Value: 4},
			ChargerFaultReason: &fsmvci.Int32Value{Value: 4},
			ChargerErrorReason: &fsmvci.Int32Value{Value: 4},
			BmsEFrame:          &fsmvci.Int32Value{Value: 4},
			ChargerEFrame:      &fsmvci.Int32Value{Value: 4},
		},
		ReConnect:          &fsmvci.BMSReConnectEvent{
			TimeOutState:   &fsmvci.Int32Value{Value: 1},
			BMSTimeoutType: 0,
			ReconnectCnt:   &fsmvci.Int32Value{Value: 0},
			NextState:      &fsmvci.Int32Value{Value: 0},
		},
		FaultList:          chargingFaultStateList,
		ChargingRecoverMsg: nil,
		CurrentTime:        &fsmvci.DateTimeLong{Time: uint64(time.Now().UnixMilli())},
		Interval:           &fsmvci.Uint32Value{Value: client.Interval},
	}
}

// VehicleChargingInterfaceLoginAnsInit
// 0x08
func (t *ThreadVCIList) VehicleChargingInterfaceLoginAnsInit (GunId uint32, client *udp.Client) *fsmvci.VCIChargingRTpush {
	chargingRecover := make([]*fsmvci.ChargingRecover, 0)
	pluggedRecover := make([]*fsmvci.PluggedRecover, 0)

	for k, _ := range t.SonThread.WorkList {
		chargingRecover = append(chargingRecover, Utils.ChargingRecoverInit(k))
		pluggedRecover = append(pluggedRecover, Utils.PluggedRecoverInit(k, true, false))
	}


	client.HeartbeatCtr++
	return &fsmvci.VCIChargingRTpush{
		ID:           &fsmvci.Uint32Value{Value: GunId},
		RTpushCtr:    &fsmvci.Uint32Value{Value: uint32(client.HeartbeatCtr)},
		GunDesireMsg: nil,
		GunHaltMsg:   nil,
		ReConnect:    nil,
		FaultList:    nil,
		Interval:     &fsmvci.Uint32Value{Value: 1000},
	}

}

/*功能区*/

// ReadMsg
// 监听所有的udp 从服务端接收的消息
// TODO 整改暂时未使用 运行一段时间崩溃 内存无限泄露
// 已修改 TODO 携程无法共用控制台
func (t *ThreadVCIList) ReadMsg () {
	for {
		select {
		case client := <-t.MainChan:
			client.ReadMsg()
		case client := <- t.FatherChan:
			client.ReadMsg()
		case client := <- t.SonChan:
			client.ReadMsg()
		}
		//for _, v := range t.MainThread.AmountList {
		//	go v.ReadMsg()
		//}
		//
		//for _, v := range t.FatherThread.AmountList {
		//	go v.ReadMsg()
		//}
		//
		//for _, v := range t.SonThread.AmountList {
		//	go v.ReadMsg()
		//}
	}
}

// HeartBeat
// 检测工作线程、发送心跳数据

func (t *ThreadVCIList) HeartBeatForMain (GunId uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			client.Timer = time.NewTimer(time.Duration(t.MainThread.DefaultInterval)*time.Millisecond)
			if _, ok := t.MainThread.WorkList[GunId]; !ok {
				goto HeartDie
			}
			client.WriteMsg(0x02, t.VCImainHeartInit(client))
		}
	}
	HeartDie:
		fmt.Printf("主级线程 %d 心跳被中止 \n", GunId)
}

func (t *ThreadVCIList) HeartBeatForFather (GunId uint32,client *udp.Client) {
	for{
		select {
		case <-client.Timer.C:
			client.Timer = time.NewTimer(time.Duration(t.FatherThread.DefaultInterval)*time.Millisecond)
			if _, ok := t.FatherThread.WorkList[GunId]; !ok {
				goto HeartDie
			}
			client.WriteMsg(0x04, t.VCIPluggedHeartbeatReqInit(GunId, client))
		}
	}
	HeartDie:
		fmt.Printf("父级线程 %d 心跳被中止 \n", GunId)
}

func (t *ThreadVCIList) HeartBeatForSon (GunId uint32,client *udp.Client) {
	for{
		select {
		case <-client.Timer.C:
			client.Timer = time.NewTimer(time.Duration(t.SonThread.DefaultInterval)*time.Millisecond)
			if _, ok := t.SonThread.WorkList[GunId]; !ok {
				goto HeartDie
			}
			client.WriteMsg(0x06, t.VCIChargingHeartbeatReqInit(GunId, client))
		}
	}
	HeartDie:
		fmt.Printf("子级线程 %d 心跳被中止 \n", GunId)
}

// MainWS
// 创建WebSocket，给前端发送数据
func (t *ThreadVCIList)  MainWS (c *gin.Context) {
	var upgrader = websocket.Upgrader{
		// 解决跨域问题
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	} // use default options
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	roomIdStr :=  c.Query("room")
	roomId, _:= strconv.Atoi(roomIdStr)

	conn.WriteMessage(1, []byte("{\"A\":\"hello\"}"))
	//同一级线程() 绑定在一个websocket发送


	if t.MainThread.WorkList[uint32(roomId)] == nil {
		conn.WriteMessage(1, []byte("{\"失败\":\"内部错误\"}"))
		conn.Close()
		return
	}

	//TODO 目前同一级线程数据全由同一个websocket做转发 Do nothing, just display the todo logo
	client := t.MainThread.WorkList[uint32(roomId)]
	client.WSocket.WsConnList[conn.RemoteAddr().String()] = conn
	client.WSocket.WsRoom[roomId] = append(client.WSocket.WsRoom[roomId], conn.RemoteAddr().String())

	//conn.Close()

	defer conn.Close()
	for k, v := range t.MainThread.WorkList {
		go checkWSMessage(v, int(k))
	}
	for{

	}
}

// FatherWS
// 创建WebSocket，给前端发送数据
func (t *ThreadVCIList)  FatherWS (c *gin.Context) {
	var upgrader = websocket.Upgrader{
		// 解决跨域问题
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	} // use default options
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	roomIdStr :=  c.Query("room")
	roomId, _:= strconv.Atoi(roomIdStr)

	//conn.WriteMessage(1, []byte("{\"A\":\"hello\"}"))
	//同一级线程() 绑定在一个websocket发送

	if t.FatherThread.WorkList[uint32(roomId)] == nil {
		conn.WriteMessage(1, []byte("{\"失败\":\"内部错误\"}"))
		conn.Close()
		return
	}

	client := t.FatherThread.WorkList[uint32(roomId)]
	client.WSocket.WsConnList[conn.RemoteAddr().String()] = conn
	client.WSocket.WsRoom[roomId] = append(client.WSocket.WsRoom[roomId], conn.RemoteAddr().String())

	defer conn.Close()
	for k, v := range t.FatherThread.WorkList {
		go checkWSMessage(v, int(k))
	}
	for{

	}
}

// SonWS
// 创建WebSocket，给前端发送数据
func (t *ThreadVCIList)  SonWS (c *gin.Context) {
	var upgrader = websocket.Upgrader{
		// 解决跨域问题
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	} // use default options
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	roomIdStr :=  c.Query("room")
	roomId, _:= strconv.Atoi(roomIdStr)
	if t.SonThread.WorkList[uint32(roomId)] == nil {
		conn.WriteMessage(1, []byte("{\"失败\":\"内部错误\"}"))
		conn.Close()
		return
	}

	client := t.SonThread.WorkList[uint32(roomId)]
	client.WSocket.WsConnList[conn.RemoteAddr().String()] = conn
	client.WSocket.WsRoom[roomId] = append(client.WSocket.WsRoom[roomId], conn.RemoteAddr().String())

	defer conn.Close()
	for k, v := range t.SonThread.WorkList {
		go checkWSMessage(v, int(k))
	}
	for{

	}
}

// 接收web传来的数据 创建‘新线程’
func (t *ThreadVCIList) buildMainHttp (c *gin.Context) {
	var VCIGun = &form.VCIGun{}
	c.ShouldBindBodyWith(VCIGun, binding.JSON)

	t.MainLive = true
	if len(t.MainThread.WorkList) == 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程达到上限",
		})
		return
	}
	if _, ok := t.MainThread.WorkList[VCIGun.Gun]; ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程冲突",
		})
		return
	}
	if _, ok := t.MainThread.AmountList[VCIGun.Gun]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "未识别枪体",
		})
		return
	}
	t.MainThread.WorkList[VCIGun.Gun] = t.MainThread.AmountList[VCIGun.Gun]
	//开启心跳
	go t.HeartBeatForMain(VCIGun.Gun, t.MainThread.WorkList[VCIGun.Gun])

	//开启接收数据监听
	//go func() {
	//	t.MainChan<-t.MainThread.WorkList[VCIGun.Gun]
	//}()
	go t.MainThread.AmountList[VCIGun.Gun].ReadMsg()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

func (t *ThreadVCIList) buildFatherHttp (c *gin.Context) {
	var VCIGun = &form.VCIGun{}
	c.ShouldBindBodyWith(VCIGun, binding.JSON)

	t.FatherLive = true
	if len(t.FatherThread.WorkList) == 6 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程达到上限",
		})
		return
	}
	if _, ok := t.FatherThread.WorkList[VCIGun.Gun]; ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程冲突",
		})
		return
	}
	if _, ok := t.FatherThread.AmountList[VCIGun.Gun]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "未识别枪体",
		})
		return
	}

	t.FatherThread.WorkList[VCIGun.Gun] = t.FatherThread.AmountList[VCIGun.Gun]

	//开启心跳
	go t.HeartBeatForFather(VCIGun.Gun, t.FatherThread.WorkList[VCIGun.Gun])

	//开启接收数据监听
	//go func() {
	//	t.FatherChan<-t.FatherThread.WorkList[VCIGun.Gun]
	//}()
	go t.FatherThread.WorkList[VCIGun.Gun].ReadMsg()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

func (t *ThreadVCIList) buildSonHttp (c *gin.Context) {
	var VCIGun = &form.VCIGun{}
	c.ShouldBindBodyWith(VCIGun, binding.JSON)

	t.SonLive = true
	if len(t.SonThread.WorkList) == 6 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程达到上限",
		})
		return
	}

	if _, ok := t.SonThread.WorkList[VCIGun.Gun]; ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "线程冲突",
		})
		return
	}
	if _, ok := t.SonThread.AmountList[VCIGun.Gun]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "未识别枪体",
		})
		return
	}

	t.SonThread.WorkList[VCIGun.Gun] = t.SonThread.AmountList[VCIGun.Gun]
	//开启心跳
	go t.HeartBeatForSon(VCIGun.Gun, t.SonThread.WorkList[VCIGun.Gun])

	//开启接收数据监听
	//go func() {
	//	t.SonChan<- t.SonThread.WorkList[VCIGun.Gun]
	//}()
	go t.SonThread.WorkList[VCIGun.Gun].ReadMsg()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

//接收web传来的数据 ‘删除’线程
func (t *ThreadVCIList) deleteMainHttp (c *gin.Context) {
	var VCIGun = &form.VCIGun{}
	c.ShouldBindBodyWith(VCIGun, binding.JSON)
	if _, ok := t.MainThread.WorkList[VCIGun.Gun]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未识别枪体",
		})
		return
	}
	//don't close the udp conn
	//t.MainThread.WorkList[VCIGun.Gun].Conn.Close()
	delete(t.MainThread.WorkList, VCIGun.Gun)
	if len(t.MainThread.WorkList) == 0 {
		t.MainLive = false
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

func (t *ThreadVCIList) deleteFatherHttp (c *gin.Context) {
	var VCIGun = &form.VCIGun{}
	c.ShouldBindBodyWith(VCIGun, binding.JSON)
	if _, ok := t.FatherThread.WorkList[VCIGun.Gun]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未识别枪体",
		})
		return
	}
	//don't close the udp conn
	//t.FatherThread.WorkList[VCIGun.Gun].Conn.Close()
	delete(t.FatherThread.WorkList, VCIGun.Gun)
	if len(t.FatherThread.WorkList) == 0 {
		t.FatherLive = false
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

func (t *ThreadVCIList) deleteSonHttp (c *gin.Context) {
	var VCIGun = &form.VCIGun{}
	c.ShouldBindBodyWith(VCIGun, binding.JSON)
	if _, ok := t.SonThread.WorkList[VCIGun.Gun]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未识别枪体",
		})
		return
	}
	//don't close the udp conn
	//t.SonThread.WorkList[VCIGun.Gun].Conn.Close()
	delete(t.SonThread.WorkList, VCIGun.Gun)
	if len(t.SonThread.WorkList) == 0 {
		t.SonLive = false
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}


// ReadConsole The client gets the command and executes it
// Deprecated: Use MainThread.RecvCommand instead, http transport data.
// 客户端获取命令并执行
/*
func (m *MainThread) ReadConsole () {
	inputReader := bufio.NewReader(os.Stdin)
	for {
		str, err := inputReader.ReadString('\n')
		if err != nil {
			fmt.Printf("Read console context error : %s \n", err.Error())
			return
		}
		str = strings.TrimSpace(str)
		if str == "exit" {
			m.MainStates.FatherLive, m.MainStates.SonLive = false, false
			return
		}

		if strings.Contains(str, ":") {
			instructionsSet := strings.Split(str, ":")
			if len(instructionsSet) < 3 {
				fmt.Printf("Set of instructions error \n")
				return
			}
			//gun := instructionsSet[0]
			action := instructionsSet[1]
			target := instructionsSet[2]
			gunId, err := Utils.StringToUint32(target)
			if err != nil {
				fmt.Printf("Get uint32 error : %s", err.Error())
			}
			m.muMain.Lock()
			if action == "build" {
				if gunId == 0 {
					if m.MainGunWork == 0 {
						m.MainStates.FatherLive, m.MainStates.SonLive = true, true
					}
					for i:=1; i<7; i++{
						buildGun(uint32(i), m)
					}
				}else{
					if _, ok := m.MainGunList[gunId]; ok {
						fmt.Printf("Gun %d is working and cannot be opened repeatedly\n", gunId)
					}else{
						if m.MainGunWork == 0 {
							m.MainStates.FatherLive, m.MainStates.SonLive = true, true
						}
						buildGun(gunId, m)
					}
				}
			}
			if action == "delete" {
				if gunId == 0 {
					for k, _ := range m.MainGunList {
						removeGun(k, m)
					}
				}else{
					if _, ok := m.MainGunList[gunId]; !ok {
						fmt.Printf("Gun %d is already dormant\n", gunId)
					}else{
						removeGun(gunId, m)
					}
				}
			}
			m.muMain.Unlock()
		}
	}
}
*/

// 主线程注册帧
func (t *ThreadVCIList) registerMain (c *gin.Context) {
	register := &form.VCIMessage{}
	c.ShouldBindBodyWith(register, binding.JSON)
	err := register.VerifyMainRegister()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":"1",
			"msg":err.Error(),
		})
		return
	}
	var a fsmvci.VehicleChargingInterfaceLogin
	json.Unmarshal([]byte(register.MessageBody), &a)
	fmt.Println(t.MainThread.WorkList)
	for _, v := range t.MainThread.WorkList {
		v.WriteMsg(0x00, &a)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

// 主线程心跳帧
func (t *ThreadVCIList) heartMain (c *gin.Context) {
	register := &form.VCIMessage{}
	c.ShouldBindBodyWith(register, binding.JSON)
	err := register.VerifyMainHeart()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":"1",
			"msg":err.Error(),
		})
		return
	}
	var a fsmvci.VCImainHeartbeatReq
	json.Unmarshal([]byte(register.MessageBody), &a)
	for _, v := range t.MainThread.WorkList {
		a.HeartbeatCtr = &fsmvci.Uint32Value{Value: uint32(v.HeartbeatCtr)}
		a.CurrentTime = &fsmvci.DateTimeLong{Time: uint64(time.Now().UnixMilli())}
		v.WriteMsg(0x02, &a)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

// 获取当前模块的消息体
func (t *ThreadVCIList) getSqlMessage(c *gin.Context) {
	var sqlMsg = form.SqlMessage{}
	c.ShouldBindBodyWith(&sqlMsg, binding.JSON)
	if sqlMsg.MessageType == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code" : 0,
			"msg" : "所属类别不能为空",
		})
		return
	}
	session := database.MySQL.Table("test_message").Where("project_type = ?", sqlMsg.MessageType)
	if sqlMsg.MessageName != "" {
		session.Where("message_name = ?", sqlMsg.MessageName)
	}
	if sqlMsg.MessageCode != "" {
		codeSlice := strings.Split(sqlMsg.MessageCode, ",")
		session.In("message_code", codeSlice)
	}
	resqData, err := session.Cols([]string{"id", "message_code", "message_name", "message_body", "project_name", "project_type"}...).QueryString()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"data":err,
			"msg":"错误",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"data":resqData,
		"msg":"成功",
	})
}

// 检测Web Sockert 管道内的数据，并发送到前端
func checkWSMessage(client *udp.Client, roomId int) {
	for{
		select {
		case dataByte := <-client.MessageChan:
			client.WSocket.Wsmu.Lock()
			connList := client.WSocket.WsRoom[roomId]
			for _, v := range connList {
				conn, _ := client.WSocket.WsConnList[v]
				err := conn.WriteMessage(1, dataByte)
				if err != nil {
					delete(client.WSocket.WsConnList, v)
					delete(client.WSocket.WsRoom, roomId)
					log.Println("goodbye:", v)
					break
				}
			}
			client.WSocket.Wsmu.Unlock()
		}
	}
}

// 中间件 跨域会先发送 option
func middlewarePost(c *gin.Context) {
	fmt.Println(c.Request.Method)
	if c.Request.Method == "options" {
		c.JSON(http.StatusOK, gin.H{
			"code" : 0,
			"msg" : "next",
		})
	}
	c.Next()
}

// RecvCommand The client gets the command and executes it
func (t *ThreadVCIList) RecvCommand () {
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
		v1.GET("/vci/main/ws", t.MainWS)
		v1.GET("/vci/father/ws", t.FatherWS)
		v1.GET("/vci/son/ws", t.SonWS)
		v1.POST("/get/cmd", t.getSqlMessage)
		//v1.GET("/vci/ws", wsocket.WS)
		r.Use(middlewarePost)
		vci := v1.Group("/vci")
		{
			main := vci.Group("/main")
			{
				main.POST("/register", t.registerMain)
				main.POST("/heart", t.heartMain)
				main.POST("/build", t.buildMainHttp)
				main.POST("/delete", t.deleteMainHttp)
			}
			father := vci.Group("/father")
			{
				father.POST("/build", t.buildFatherHttp)
				father.POST("/delete", t.deleteFatherHttp)
			}
			son := vci.Group("/son")
			{
				son.POST("/build", t.buildSonHttp)
				son.POST("/delete", t.deleteSonHttp)
			}
		}
	}
	r.Run(":10001")
}
