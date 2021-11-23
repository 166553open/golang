package fsm

import (
	conf "FSMTestingPlatform/Conf"
	database "FSMTestingPlatform/Database"
	"FSMTestingPlatform/Utils"
	"FSMTestingPlatform/form"
	"FSMTestingPlatform/pool"
	"FSMTestingPlatform/protoc/fsmvci"
	"FSMTestingPlatform/udp"

	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	// github 导入
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gorilla/websocket"
)

type Thread struct {
	//RtMessageStatus bool 				// realtime 消息是否发送
	RtMessageInterval uint32			// realtime 消息发送间隔
	RtTimer	*time.Timer					// realtime 定时器管道
	RtMessageList map[uint32]bool		// 子线程的每个线程做控制

	//DefaultInterval uint16				// 默认值 预期收到回复时长
	pool *pool.UDPpool					// udp链接池
	//States interface{}				// 状态接口
	WSocket map[string]*websocket.Conn	// Websocket链接 map表
	WorkList map[uint32]*udp.Client		// 工作线程 map列表
	AmountList map[uint32]*udp.Client	// 所有线程 map列表

}

type ThreadVCIList struct {
	muThread sync.Mutex					// 互斥锁
	MainLive bool						// 第一线程在线状态
	//FatherLive bool						// 第二线程在线状态
	SonLive bool						// 第三线程在线状态

	MainChan chan *udp.Client			// 第一线程管道 传输udp client
	//FatherChan chan *udp.Client			// 第二线程管道 传输udp client
	SonChan chan *udp.Client			// 第三线程管道 传输udp client

	MainThread *Thread					// 第一线程的对象实体
	//FatherThread *Thread				// 第二线程的对象实体
	SonThread *Thread					// 第三线程的对象实体
}

// NewThread
// 共用的方法
func NewThread (addr net.UDPAddr, poolSize int) *Thread {
	newPool := pool.NewUdppool(addr, poolSize)
	List := make(map[uint32]*udp.Client)
	for i:=1; i<poolSize+1; i++  {
		client, _ :=udp.NewClient(newPool)
		List[uint32(i)] = client
	}
	return &Thread{
		//DefaultInterval: 200,
		//RtMessageStatus: false,
		RtMessageInterval: 2000,
		RtTimer: 		 time.NewTimer(time.Duration(1000)*time.Millisecond),
		RtMessageList: 	 make(map[uint32]bool),
		pool:            newPool,
		WSocket:         make(map[string]*websocket.Conn),
		WorkList:        make(map[uint32]*udp.Client),
		AmountList:      List,
	}
}

func NewThreadList (main, son *Thread) *ThreadVCIList {
	return &ThreadVCIList{
		muThread:     sync.Mutex{},
		MainLive:     false,
		//FatherLive:   false,
		SonLive:      false,

		MainChan:	  make(chan *udp.Client),
		//FatherChan:	  make(chan *udp.Client),
		SonChan:	  make(chan *udp.Client),
		MainThread:   main,
		//FatherThread: father,
		SonThread:    son,
	}
}


func ThreadVCIInit () *ThreadVCIList {
	addrMain := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9000,
	}
	//addrFather := net.UDPAddr{
	//	IP:   net.IPv4(0, 0, 0, 0),
	//	Port: 9010,
	//}
	addrSon := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9020,
	}
	mainThread := NewThread(addrMain, 1)
	//fatherThread := NewThread(addrFather, 6)
	sonThread := NewThread(addrSon, 6)

	thread := NewThreadList(mainThread, sonThread)
	return thread
}
/*功能区*/

// ReadMsg
// 监听所有的udp 从服务端接收的消息
// TODO 整改暂时未使用 运行一段时间崩溃 内存无限泄露
// 已修改 TODO 协程无法共用控制台
func (t *ThreadVCIList) ReadMsg () {
	for {
		select {
		case client := <-t.MainChan:
			client.ReadMsg()
		//case client := <- t.FatherChan:
		//	client.ReadMsg()
		case client := <- t.SonChan:
			client.ReadMsg()
		}
	}
}

// HeartBeat （HeartBeatForMain、HeartBeatForFather、HeartBeatForSon）
// 检测工作线程、发送心跳数据

func (t *ThreadVCIList) HeartBeatForRegister (GunId uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			// timer.C 管道类型数据。定时器功能，到指定时间会发出数据（time）。
			// 触发消息之后，要重新赋值下次出发的时间
			client.Timer = time.NewTimer(time.Duration(client.Interval)*time.Millisecond)
			if _, ok := t.MainThread.WorkList[GunId]; !ok {
				// 工作列表中不存在线程信息，心跳中止
				goto HeartDie
			}
			// 发送心跳数据
			client.WriteMsg(0x02, t.VCImainHeartInit(client))
		}
	}
	HeartDie:
		fmt.Printf("主级线程 %d 心跳被中止 \n", GunId)
}

// son线程 注销
/*
func (t *ThreadVCIList) HeartBeatForFather (GunId uint32,client *udp.Client) {
	for{
		select {
		case <-client.Timer.C:
			// timer.C 管道类型数据。定时器功能，到指定时间会发出数据（time）。
			// 触发消息之后，要重新赋值下次出发的时间
			client.Timer = time.NewTimer(time.Duration(t.FatherThread.DefaultInterval)*time.Millisecond)
			if _, ok := t.FatherThread.WorkList[GunId]; !ok {
				// 工作列表中不存在线程信息，心跳中止
				goto HeartDie
			}
			// 发送心跳数据
			client.WriteMsg(0x04, t.VCIPluggedHeartbeatReqInit(GunId, client))
		}
	}
	HeartDie:
		fmt.Printf("父级线程 %d 心跳被中止 \n", GunId)
}
*/

func (t *ThreadVCIList) HeartBeatForCharge (gunId uint32, client *udp.Client) {
	for{
		select {
		case <-client.Timer.C:
			// timer.C 管道类型数据。定时器功能，到指定时间会发出数据（time）。
			// 触发消息之后，要重新赋值下次出发的时间
			client.Timer = time.NewTimer(time.Duration(client.Interval)*time.Millisecond)
			if _, ok := t.SonThread.WorkList[gunId]; !ok {
				// 工作列表中不存在线程信息，心跳中止
				goto HeartDie
			}
			// 发送心跳数据
			client.WriteMsg(0x06, t.VCIChargingHeartbeatReqInit(gunId, client))
		}
	}
	HeartDie:
		fmt.Printf("子级线程 %d 心跳被中止 \n", gunId)
}

// RegisterWS
// 创建WebSocket，给前端发送数据
func (t *ThreadVCIList)  RegisterWS (c *gin.Context) {
	// 升级HTTP链接至WebSocket
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

	// 获取room参数，备后期需要
	roomIdStr :=  c.Query("room")
	roomId, _:= strconv.Atoi(roomIdStr)

	// 发送测试数据
	conn.WriteMessage(1, []byte("{\"成功\":\"hello\"}"))

	//同一级线程() 绑定在一个websocket发送
	if t.MainThread.WorkList[uint32(roomId)] == nil {
		conn.WriteMessage(1, []byte("{\"失败\":\"内部错误, 无法识别枪线程\"}"))
		conn.Close()
		return
	}
	// 启用互斥锁
	//TODO 目前同一级线程数据全由同一个websocket做转发 Do nothing, just display the todo logo
	// roomId 和枪Id相对应
	client := t.MainThread.WorkList[uint32(roomId)]
	client.WSocket.Wsmu.Lock()

	// 所有的 websocket 链接
	client.WSocket.WsConnList[conn.RemoteAddr().String()] = conn
	// 当前组的 websocket 链接
	client.WSocket.WsRoom[roomId] = append(client.WSocket.WsRoom[roomId], conn.RemoteAddr().String())

	client.WSocket.Wsmu.Unlock()
	defer conn.Close()

	for k, v := range t.MainThread.WorkList {
		go v.CheckWSMessage(int(k))
	}
	for{

	}
}

// son线程 注销
/*
// FatherWS
// 创建WebSocket，给前端发送数据
func (t *ThreadVCIList)  FatherWS (c *gin.Context) {
	// 升级HTTP链接至WebSocket
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

	// 获取room参数，备后期需要
	roomIdStr :=  c.Query("room")
	roomId, _:= strconv.Atoi(roomIdStr)

	// 发送测试数据
	conn.WriteMessage(1, []byte("{\"成功\":\"hello\"}"))

	//同一级线程() 绑定在一个websocket发送
	if t.FatherThread.WorkList[uint32(roomId)] == nil {
		conn.WriteMessage(1, []byte("{\"失败\":\"内部错误, 无法识别枪线程\"}"))
		conn.Close()
		return
	}

	// 启用互斥锁
	client := t.FatherThread.WorkList[uint32(roomId)]
	client.WSocket.Wsmu.Lock()

	// 当前client下 所有的 websocket 链接
	client.WSocket.WsConnList[conn.RemoteAddr().String()] = conn
	// 当前组的 websocket 链接
	client.WSocket.WsRoom[roomId] = append(client.WSocket.WsRoom[roomId], conn.RemoteAddr().String())

	client.WSocket.Wsmu.Unlock()
	defer conn.Close()

	for k, v := range t.FatherThread.WorkList {
		go v.CheckWSMessage(int(k))
	}
	for{

	}
}
*/

// ChargeWS
// 创建WebSocket，给前端发送数据
func (t *ThreadVCIList)  ChargeWS (c *gin.Context) {
	// 升级HTTP链接至WebSocket
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

	// 获取room参数，备后期需要
	roomIdStr :=  c.Query("room")
	roomId, _:= strconv.Atoi(roomIdStr)

	// 发送测试数据
	conn.WriteMessage(1, []byte("{\"成功\":\"hello\"}"))

	if t.SonThread.WorkList[uint32(roomId)] == nil {
		conn.WriteMessage(1, []byte("{\"失败\":\"内部错误, 无法识别枪线程\"}"))
		conn.Close()
		return
	}

	// 启用互斥锁
	client := t.SonThread.WorkList[uint32(roomId)]
	client.WSocket.Wsmu.Lock()

	// 当前client下 所有的 websocket 链接
	client.WSocket.WsConnList[conn.RemoteAddr().String()] = conn
	// 当前组的 websocket 链接
	client.WSocket.WsRoom[roomId] = append(client.WSocket.WsRoom[roomId], conn.RemoteAddr().String())

	client.WSocket.Wsmu.Unlock()
	defer conn.Close()
	for k, v := range t.SonThread.WorkList {
		go v.CheckWSMessage(int(k))
	}
	for{

	}
}



// 中间件
// axios 跨域请求会先发送 option
// 已经在vue侧做了代理
func middlewarePost(c *gin.Context) {
	if c.Request.Method == "options" {
		c.JSON(http.StatusOK, gin.H{
			"code" : 0,
			"msg" : "next",
		})
	}
	c.Next()
}

func (t *ThreadVCIList) RTStart() {
	index3, index5 := int64(0), int64(0)
	modelFault := conf.VCIRedisListName
	modelIndex := []int64{index3, index5}
	// 指针
	for {
		select {
		case <-t.SonThread.RtTimer.C:
			t.SonThread.RtTimer = time.NewTimer(time.Duration(t.SonThread.RtMessageInterval)*time.Millisecond)
			for k, v := range t.SonThread.WorkList {
				if t.SonThread.RtMessageList[k] {
					modelFaultKey := Utils.RandValue(2)
					RtInfo, err := Utils.Redis2FaultVCI(k, v.RTpushCtr, v.Interval, modelFault[modelFaultKey], int64(modelIndex[modelFaultKey]))
					if err != nil {
						fmt.Printf("real time message create error : %s", err.Error())
						continue
					}
					v.WriteMsg(0x08, RtInfo)
					modelIndex[modelFaultKey]++
					if modelIndex[0] > 76706 {
						modelIndex[0] = 0
					}
					if modelIndex[1] > 21111090 {
						modelIndex[1] = 0
					}
				}
			}
		}
	}
}

// RecvCommand The client gets the command and executes it
func (t *ThreadVCIList) RecvCommand (r *gin.Engine) {
	// 输出gin日志到 gin_http 文件
	logfile, err := os.Create("./gin_VCI.log")
	if err != nil {
		fmt.Println("Could not create log file")
	}
	gin.SetMode(gin.DebugMode)
	gin.DefaultWriter = io.MultiWriter(logfile)

	// http 路由分组
	v1 := r.Group("/api/v1")
	{

		v1.POST("/get/cmd", t.getSqlMessage)
		v1.GET("/vci/fault", t.fault)

		v1.GET("/vci/main/ws", t.RegisterWS)
		v1.GET("/vci/son/ws", t.ChargeWS)

		//v1.GET("/vci/ws", wsocket.WS)
		r.Use(middlewarePost)
		register := v1.Group("/vci-register")
		{
			register.POST("/build", t.buildRegisterHttp)
			register.POST("/change-state", t.changeIntervalAtRegister)
			register.POST("/register", t.registerRegister)
			register.POST("/heart", t.heartRegister)

			//main.POST("/delete-and-change", t.deleteMainHttp)
			register.DELETE("/delete", t.deleteRegisterHttp)


			//father := vci.Group("/father")
			//{
			//	father.POST("/build", t.buildFatherHttp)
			//	father.DELETE("/delete", t.deleteFatherHttp)
			//}
		}
		charge := v1.Group("/vci-charge")
		{
			charge.POST("/build", t.buildChargeHttp)
			charge.POST("/change-state", t.changeIntervalAtCharge)
			charge.POST("/change-rtstatus", t.changeRTStatus)
			charge.DELETE("/delete", t.deleteChargeHttp)
		}
	}
}

/*接口区 START*/
// 接收web传来的数据 创建‘新线程’
func (t *ThreadVCIList) buildRegisterHttp (c *gin.Context) {
	// 识别参数ID
	var vciConfig = form.VCIRegisterConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)

	//if vciConfig.Interval == 0 {
	//	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
	//		"code":1,
	//		"msg":"心跳间隔时间不能为零",
	//	})
	//	return
	//}

	//if len(t.MainThread.WorkList) == 1 {
	//	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
	//		"code":1,
	//		"msg": "线程达到上限",
	//	})
	//	return
	//}
	//
	//// 异常数据场景 信息返回前端
	//if _, ok := t.MainThread.WorkList[vciConfig.GunId]; ok {
	//	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
	//		"code":1,
	//		"msg": "线程冲突",
	//	})
	//	return
	//}
	if _, ok := t.MainThread.AmountList[vciConfig.Id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "未识别枪体Id",
		})
		return
	}

	//启用互斥锁
	t.muThread.Lock()
	if !t.MainLive {
		t.MainLive = true
	}

	// 从全局udp列表 COPY 到 work udp列表中
	t.MainThread.WorkList[vciConfig.Id] = t.MainThread.AmountList[vciConfig.Id]
	t.MainThread.WorkList[vciConfig.Id].Interval = vciConfig.Interval

	t.muThread.Unlock()
	// 启用协程监听、读取udp数据
	go t.MainThread.AmountList[vciConfig.Id].ReadMsg()

	// 启用协程周期性发送 心跳数据
	go t.HeartBeatForRegister(vciConfig.Id, t.MainThread.WorkList[vciConfig.Id])
	//开启接收数据监听
	//go func() {
	//	t.MainChan<-t.MainThread.WorkList[VCIGun.Gun]
	//}()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

// son线程 注销
/*
func (t *ThreadVCIList) buildFatherHttp (c *gin.Context) {
	//识别参数ID
	var VCIGun = &form.VCIGun{}
	c.ShouldBindBodyWith(VCIGun, binding.JSON)

	// 异常数据场景，返回数据到前端
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

	// 启用互斥锁
	t.muThread.Lock()
	if !t.FatherLive {
		t.FatherLive = true
	}
	// 从全局udp列表 COPY 到 work udp列表中
	t.FatherThread.WorkList[VCIGun.Gun] = t.FatherThread.AmountList[VCIGun.Gun]
	t.muThread.Unlock()
	// 启用协程监听、读取udp数据
	go t.FatherThread.WorkList[VCIGun.Gun].ReadMsg()

	// 启用协程周期性发送 心跳数据
	//go t.HeartBeatForFather(VCIGun.Gun, t.FatherThread.WorkList[VCIGun.Gun])

	//开启接收数据监听
	//go func() {
	//	t.FatherChan<-t.FatherThread.WorkList[VCIGun.Gun]
	//}()

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}
*/

func (t *ThreadVCIList) buildChargeHttp (c *gin.Context) {
	// 识别参数ID
	var vciConfig = form.VCIConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)

	// 异常数据场景 消息发送给前端
	//if len(t.SonThread.WorkList) == 6 {
	//	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
	//		"code":1,
	//		"msg": "线程达到上限",
	//	})
	//	return
	//}

	//if _, ok := t.SonThread.WorkList[vciConfig.GunId]; ok {
	//	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
	//		"code":1,
	//		"msg": "线程冲突",
	//	})
	//	return
	//}
	if _, ok := t.SonThread.AmountList[vciConfig.GunId]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg": "未识别枪体",
		})
		return
	}

	// 启用互斥锁
	t.muThread.Lock()
	if !t.SonLive {
		t.SonLive = true
	}
	// 从全局 udp列表 COPY 到work udp列表中
	t.SonThread.WorkList[vciConfig.GunId] = t.SonThread.AmountList[vciConfig.GunId]
	t.muThread.Unlock()

	//
	rt := t.VehicleChargingInterfaceLoginAnsInit(vciConfig.GunId, t.SonThread.WorkList[vciConfig.GunId])
	t.SonThread.WorkList[vciConfig.GunId].WriteMsg(0x08, rt)
	t.SonThread.RtMessageList[vciConfig.GunId] = false
	// 启用协程监听、读取udp数据
	go t.SonThread.WorkList[vciConfig.GunId].ReadMsg()
	// 启用协程周期性发送 心跳数据
	go t.HeartBeatForCharge(vciConfig.GunId, t.SonThread.WorkList[vciConfig.GunId])

	//开启接收数据监听
	//go func() {
	//	t.SonChan<- t.SonThread.WorkList[VCIGun.Gun]
	//}()

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

//接收web传来的数据 ‘删除’线程
func (t *ThreadVCIList) deleteRegisterHttp (c *gin.Context) {
	// 识别参数ID
	var vciConfig = form.VCIConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)
	if _, ok := t.MainThread.WorkList[vciConfig.Id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未识别枪体",
		})
		return
	}
	// 仅从工作队列中删除，保留Client
	// 启用互斥锁
	t.muThread.Lock()
	// 从工作列表中删除当前的线程
	delete(t.MainThread.WorkList, vciConfig.Id)
	if len(t.MainThread.WorkList) == 0 {
		t.MainLive = false
	}
	t.muThread.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

// son线程 注销
/*
func (t *ThreadVCIList) deleteFatherHttp (c *gin.Context) {
	// 识别参数ID
	var VCIGun = &form.VCIGun{}
	c.ShouldBindBodyWith(VCIGun, binding.JSON)
	if _, ok := t.FatherThread.WorkList[VCIGun.Gun]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未识别枪体",
		})
		return
	}
	// 仅从工作队列中删除，保留Client
	// 启用互斥锁
	t.muThread.Lock()
	// 从工作列表中删除当前的线程
	delete(t.FatherThread.WorkList, VCIGun.Gun)
	if len(t.FatherThread.WorkList) == 0 {
		t.FatherLive = false
	}
	t.muThread.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}
*/

func (t *ThreadVCIList) deleteChargeHttp (c *gin.Context) {
	// 识别参数ID
	var vciConfig = form.VCIConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)
	if _, ok := t.SonThread.WorkList[vciConfig.GunId]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未识别枪体",
		})
		return
	}
	// 仅从工作队列中删除，保留Client
	// 启用互斥锁
	t.muThread.Lock()
	// 从工作列表中删除当前的线程
	delete(t.SonThread.WorkList, vciConfig.GunId)
	delete(t.SonThread.RtMessageList, vciConfig.GunId)
	if len(t.SonThread.WorkList) == 0 {
		t.SonLive = false
	}
	t.muThread.Unlock()

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


// 从数据库获取当前模块的消息体
func (t *ThreadVCIList) getSqlMessage(c *gin.Context) {
	// 识别参数的数据
	var sqlMsg = database.SqlMessage{}
	c.ShouldBindBodyWith(&sqlMsg, binding.JSON)
	if sqlMsg.MessageType == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code" : 0,
			"msg" : "所属类别不能为空",
		})
		return
	}
	// 查询数据库protocol信息
	where := make(map[string]interface{})
	in := make(map[string][]string)

	where["project_id"] = sqlMsg.MessageType
	if sqlMsg.MessageName != "" {
		where["message_name"] = sqlMsg.MessageName
	}
	if sqlMsg.MessageCode != "" {
		codeSlice := strings.Split(sqlMsg.MessageCode, ",")
		in["message_code"] = codeSlice
	}
	// 数据库返回数据

	resqData, err := Utils.MysqlSql("test_message", where, in, []string{"id", "message_code", "message_name", "message_body", "project_name", "project_id"})

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

// 主线程注册帧
func (t *ThreadVCIList) registerRegister (c *gin.Context) {
	// 识别参数的数据
	register := form.VCIRegister{}
	c.ShouldBindBodyWith(&register, binding.JSON)

	selfCheckState := 0
	selfCheckState = register.SelfCheckState

	if selfCheckState > 3  {
		selfCheckState = 3
	}

	if len(t.MainThread.WorkList) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"没有处在工作中的线程",
		})
	}

	// json字节流数据转化为protocol数据
	//var a fsmvci.VCIManagerRegisterInfo
	//json.Unmarshal([]byte(register.MessageBody), &a)


	// 给所有的工作线程发送数据
	for _, v := range t.MainThread.WorkList {
		v.WriteMsg(0x00, Utils.VCIRegister(selfCheckState))
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

// 主线程心跳帧
func (t *ThreadVCIList) heartRegister (c *gin.Context) {
	// 识别参数的数据
	heartBeat := form.VCIRegisterHeartBeat{}
	c.ShouldBindBodyWith(&heartBeat, binding.JSON)

	if _, ok := t.SonThread.WorkList[heartBeat.GunId]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":"1",
			"msg":"没有处在工作中的线程",
		})
		return
	}
	register := t.MainThread.WorkList[1]

	if heartBeat.GunBasicState.LinkState > 0 && heartBeat.GunBasicState.PositionedState >0 {
		heartBeat.GunBasicState.PositionedState = 0
	}

	if heartBeat.HeartBeatCnt == 0 {
		heartBeat.HeartBeatCnt = uint32(register.HeartbeatCtr)
	}

	if heartBeat.HeartBeatPeriod == 0 {
		heartBeat.HeartBeatPeriod = register.RTpushCtr
	}

	if heartBeat.TimeNow == 0 {
		heartBeat.TimeNow = register.Interval
	}

	gunBaseList := make([]*fsmvci.GunBasicState, 0)
	connectStateList := make([]*fsmvci.GunConnectState, 0)

	var gunBaseStatus = [3][]uint32{}
	gunBaseStatus[0] = []uint32{0, 0}
	gunBaseStatus[1] = []uint32{0, 1}
	gunBaseStatus[2] = []uint32{1, 1}
	for k, _ := range t.SonThread.WorkList {
		statusList := gunBaseStatus[Utils.RandValue(3)]
		if k == heartBeat.GunId {
			gunBaseList = append(gunBaseList, Utils.GunBaseStatusInit(heartBeat.GunId, heartBeat.GunBasicState.LinkState, heartBeat.GunBasicState.PositionedState))

		}else{
			gunBaseList = append(gunBaseList, Utils.GunBaseStatusInit(k, statusList[0], statusList[1]))
		}

		connectStateList = append(connectStateList, Utils.ConnectStateList(1, 1, 1, 1))
	}

	VCIManagerHeartBeatInfo := Utils.VCIRegisterHeart(heartBeat.HeartBeatCnt, heartBeat.HeartBeatPeriod, uint64(heartBeat.TimeNow), gunBaseList, connectStateList)

	// json 字节流转化为 protocol 类型
	//var VCIManagerHeartBeatInfo fsmvci.VCIManagerHeartBeatInfo
	//Utils.Json2PB(register.MessageBody, &VCIManagerHeartBeatInfo)
	for _, v := range t.MainThread.WorkList {
		v.WriteMsg(0x02, VCIManagerHeartBeatInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"success",
	})
}

// 动态修改心跳时间
func (t *ThreadVCIList) changeIntervalAtRegister (c *gin.Context) {
	// 获取并绑定网络传输的数据
	vciConfig :=  form.VCIConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)

	if vciConfig.HeartBeatPeriod == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"心跳间隔时间不能为零",
		})
		return
	}
	if len(t.MainThread.WorkList) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"没有处在工作中的线程",
		})
		return
	}

	// 修改所有工作中线程的间隔时长
	for _, v := range t.MainThread.WorkList {
		v.Mu.Lock()
		v.Interval = vciConfig.HeartBeatPeriod
		v.Mu.Unlock()
	}
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"修改成功",
	})
}

// 动态修改心跳时间
func (t *ThreadVCIList) changeIntervalAtCharge (c *gin.Context) {
	vcichBeat := form.ChargeHeartBeat{}
	c.ShouldBindBodyWith(&vcichBeat, binding.JSON)

	if vcichBeat.HeartBeatPeriod == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"心跳间隔时间不能为零",
		})
		return
	}
	if len(t.SonThread.WorkList) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"没有处在工作中的线程",
		})
		return
	}

	if _, ok := t.SonThread.WorkList[vcichBeat.GunId]; !ok {
		if vcichBeat.GunId != 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code":1,
				"msg":"未识别的线程ID",
			})
			return
		}
		for _, v := range t.SonThread.WorkList {
			v.Mu.Lock()
			v.Interval = vcichBeat.HeartBeatPeriod
			v.Mu.Unlock()
		}

		c.JSON(http.StatusOK, gin.H{
			"code":0,
			"msg":"修改成功",
		})
		return

	}

	client := t.SonThread.WorkList[vcichBeat.GunId]
	client.Mu.Lock()
	client.Interval = vcichBeat.HeartBeatPeriod
	client.Mu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"修改成功",
	})
	return
}

// 动态修改 vci RTTime消息类型的状态
func (t *ThreadVCIList) changeRTStatus (c *gin.Context) {
	vciRTInfo :=  form.VCIRTInfo{}
	c.ShouldBindBodyWith(&vciRTInfo, binding.JSON)

	err := vciRTInfo.VerifyVCIRTInfo()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":err.Error(),
		})
		return
	}

	t.muThread.Lock()
	defer t.muThread.Unlock()
	if _, ok := t.SonThread.WorkList[vciRTInfo.GunId]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":"未识别枪体Id",
		})
		return
	}
	if vciRTInfo.HeartBeatPeriod == 0 {
		vciRTInfo.HeartBeatPeriod = t.SonThread.WorkList[vciRTInfo.GunId].Interval
	}
	if vciRTInfo.PushCnt == 0 {
		vciRTInfo.PushCnt = t.SonThread.WorkList[vciRTInfo.GunId].RTpushCtr
	}

	t.SonThread.RtMessageList[vciRTInfo.GunId] = vciRTInfo.Status
	t.SonThread.RtMessageInterval = vciRTInfo.HeartBeatPeriod

	t.SonThread.WorkList[vciRTInfo.GunId].RTpushCtr = vciRTInfo.PushCnt
	t.SonThread.WorkList[vciRTInfo.GunId].Interval = vciRTInfo.HeartBeatPeriod

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"修改成功",
	})
}

/*测试 redis 取枚举值*/
func (t *ThreadVCIList) fault(c *gin.Context) {

	strName := c.Query("fault_list_name")
	listId := c.Query("fault_list_id")
	listIdInt, _ := strconv.Atoi(listId)

	data, err := Utils.Redis2FaultVCI(1, 10, 1000,  strName, int64(listIdInt))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":1,
			"msg":err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":0,
		"msg":"成功",
		"data": data,
	})
	return

}
/*接口区 END*/

/*功能区 START*/
//fsm cvi 消息体初始化

// VCIMainLoginInit
// 0x00
func (t *ThreadVCIList) VCIMainLoginInit(state int) *fsmvci.VCIManagerRegisterInfo  {
	return Utils.VCIRegister(state)
}

// VCImainHeartInit
// 0x02
func (t *ThreadVCIList) VCImainHeartInit (client *udp.Client) *fsmvci.VCIManagerHeartBeatInfo {
	gunBaseList := make([]*fsmvci.GunBasicState, 0)
	connectStateList := make([]*fsmvci.GunConnectState, 0)
	var gunBaseStatus = [3][]uint32{}
	gunBaseStatus[0] = []uint32{0, 0}
	gunBaseStatus[1] = []uint32{0, 1}
	gunBaseStatus[2] = []uint32{1, 1}
	for k, _ := range t.SonThread.WorkList {
		statusList := gunBaseStatus[Utils.RandValue(3)]
		gunBaseList = append(gunBaseList, Utils.GunBaseStatusInit(k, statusList[0], statusList[1]))
		connectStateList = append(connectStateList, Utils.ConnectStateList(1, 1, 1, 1))
	}
	client.HeartbeatCtr++
	return Utils.VCIRegisterHeart(uint32(client.HeartbeatCtr), client.Interval, uint64(time.Now().UnixMilli()), gunBaseList, connectStateList)

}

/*
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
*/

// VCIChargingHeartbeatReqInit
// 0x06
func (t *ThreadVCIList) VCIChargingHeartbeatReqInit (GunId uint32, client *udp.Client) *fsmvci.VCIChargerHeartBeatInfo {
	client.HeartbeatCtr++

	faultList := make([]*fsmvci.ChargerFaultState, 0)

	chargerFaultState := Utils.FaultStateAtVCI(0, 0)
	chargerFaultState2 := Utils.FaultStateAtVCI(0, 0)
	faultList =  append(faultList, []*fsmvci.ChargerFaultState{chargerFaultState, chargerFaultState2}...)

	return Utils.VCIChargingHeart(GunId, uint32(client.HeartbeatCtr), client.Interval, faultList)
}

// VehicleChargingInterfaceLoginAnsInit
// 0x08
func (t *ThreadVCIList) VehicleChargingInterfaceLoginAnsInit (GunId uint32, client *udp.Client) *fsmvci.VCIChargerRTInfo {
	client.HeartbeatCtr++
	client.RTpushCtr++
	faultList := make([]*fsmvci.ChargerFaultState, 0)

	chargerFaultState := Utils.FaultStateAtVCI(0,0)
	faultList = append(faultList, chargerFaultState)
	return Utils.VCIChargerRTInfo(GunId, client.RTpushCtr, client.Interval, faultList)
}

/*功能区 END*/