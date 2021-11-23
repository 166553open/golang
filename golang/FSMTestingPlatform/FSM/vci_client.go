package fsm

import (
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

	conf "FSMTestingPlatform/Conf"
	database "FSMTestingPlatform/Database"
	form "FSMTestingPlatform/Form"
	pool "FSMTestingPlatform/Pool"
	"FSMTestingPlatform/Protoc/fsmvci"
	udp "FSMTestingPlatform/Udp"
	"FSMTestingPlatform/Utils"

	// github 导入
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gorilla/websocket"
)

// Thread
// -----------------------------------------------------------------------
// 线程结构体
// -----------------------------------------------------------------------
type Thread struct {
	RtMessageInterval uint32          // realtime 消息发送间隔
	RtTimer           *time.Timer     // realtime 定时器管道
	RtMessageList     map[uint32]bool // 子线程的每个线程做控制
	pool              *pool.UDPpool   // udp链接池

	WSocket    map[string]*websocket.Conn // Websocket链接 map表
	WorkList   map[uint32]*udp.Client     // 工作线程 map列表
	AmountList map[uint32]*udp.Client     // 所有线程 map列表

}

// ThreadVCIList
// -----------------------------------------------------------------------
// VCI结构体
// -----------------------------------------------------------------------
type ThreadVCIList struct {
	muThread sync.Mutex // 互斥锁
	MainLive bool       // 第一线程在线状态
	SonLive  bool       // 第三线程在线状态

	MainChan chan *udp.Client // 第一线程管道 传输udp client
	SonChan  chan *udp.Client // 第三线程管道 传输udp client

	MainThread *Thread // 第一线程的对象实体
	SonThread  *Thread // 第三线程的对象实体
}

// NewThread
// -----------------------------------------------------------------------
// 构建线程返回线程的指针（VCI、PMM、OHP共用）
// -----------------------------------------------------------------------
func NewThread(addr net.UDPAddr, poolSize int) *Thread {
	newPool := pool.NewUdppool(addr, poolSize)
	List := make(map[uint32]*udp.Client)
	for i := 1; i < poolSize+1; i++ {
		client, _ := udp.NewClient(newPool)
		List[uint32(i)] = client
	}
	return &Thread{
		RtMessageInterval: 2000,
		RtTimer:           time.NewTimer(time.Duration(1000) * time.Millisecond),
		RtMessageList:     make(map[uint32]bool),
		pool:              newPool,
		WSocket:           make(map[string]*websocket.Conn),
		WorkList:          make(map[uint32]*udp.Client),
		AmountList:        List,
	}
}

// NewThreadList
// ------------------------------------------------------------------------
// 构建OHP结构体指针
// ------------------------------------------------------------------------
func NewThreadList(main, son *Thread) *ThreadVCIList {
	return &ThreadVCIList{
		muThread: sync.Mutex{},
		MainLive: false,
		SonLive:  false,

		MainChan:   make(chan *udp.Client),
		SonChan:    make(chan *udp.Client),
		MainThread: main,
		SonThread:  son,
	}
}

// ThreadVCIInit
// ------------------------------------------------------------------------
// 初始化VCI，返回VCI线程指针
// TODO Address 可以放入json
// ------------------------------------------------------------------------
func ThreadVCIInit() *ThreadVCIList {
	addrMain := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9000,
	}
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
// ------------------------------------------------------------------------
// 监听所有的udp 从服务端接收的消息
// ------------------------------------------------------------------------
func (t *ThreadVCIList) ReadMsg() {
	for {
		select {
		case client := <-t.MainChan:
			client.ReadMsg()
		case client := <-t.SonChan:
			client.ReadMsg()
		}
	}
}

// HeartBeatForMain
// -----------------------------------------------------------------------
// 检测工作线程、发送心跳数据
// -----------------------------------------------------------------------
func (t *ThreadVCIList) HeartBeatForRegister(GunId uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			// timer.C 管道类型数据。定时器功能，到指定时间会发出数据（time）。
			// 触发消息之后，要重新赋值下次出发的时间
			client.Timer = time.NewTimer(time.Duration(client.Interval) * time.Millisecond)
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

// HeartBeatForSon
// -----------------------------------------------------------------------
// 检测工作线程、发送心跳数据
// -----------------------------------------------------------------------
func (t *ThreadVCIList) HeartBeatForCharge(gunId uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			// timer.C 管道类型数据。定时器功能，到指定时间会发出数据（time）。
			// 触发消息之后，要重新赋值下次出发的时间
			client.Timer = time.NewTimer(time.Duration(client.Interval) * time.Millisecond)
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
// -----------------------------------------------------------------------
// 构建充电线程的WebSocket，给前端发送数据
// -----------------------------------------------------------------------
func (t *ThreadVCIList) RegisterWS(c *gin.Context) {
	// 升级HTTP链接至WebSocket
	var upgrader = websocket.Upgrader{
		// 解决跨域问题
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	// 获取room参数，备后期需要
	roomIdStr := c.Query("room")
	roomId, _ := strconv.Atoi(roomIdStr)

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
	// roomId 和枪id相对应
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
	for {

	}
}

// ChargeWS
// -----------------------------------------------------------------------
// 构建充电线程的WebSocket，给前端发送数据
// -----------------------------------------------------------------------
func (t *ThreadVCIList) ChargeWS(c *gin.Context) {
	// 升级HTTP链接至WebSocket
	var upgrader = websocket.Upgrader{
		// 解决跨域问题
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	// 获取room参数，备后期需要
	roomIdStr := c.Query("room")
	roomId, _ := strconv.Atoi(roomIdStr)

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
	for {

	}
}

// middlewarePost
// -----------------------------------------------------------------------
// 中间件
// axios 跨域请求会先发送 option
// tips 已经在vue侧做了代理
// -----------------------------------------------------------------------
func middlewarePost(c *gin.Context) {
	if c.Request.Method == "options" {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "next",
		})
	}
	c.Next()
}

// RTStart
// -----------------------------------------------------------------------
// 充电线程发送RT数据帧
// -----------------------------------------------------------------------
func (t *ThreadVCIList) RTStart() {
	index3, index5 := int64(0), int64(0)
	modelFault := conf.VCIRedisListName
	modelIndex := []int64{index3, index5}
	// 指针
	for {
		select {
		case <-t.SonThread.RtTimer.C:
			t.SonThread.RtTimer = time.NewTimer(time.Duration(t.SonThread.RtMessageInterval) * time.Millisecond)
			for k, v := range t.SonThread.WorkList {
				if t.SonThread.RtMessageList[k] {
					if modelIndex[0] > 76706 {
						modelIndex[0] = 0
					}
					if modelIndex[1] > 21111090 {
						modelIndex[1] = 0
					}
					modelFaultKey := Utils.RandValue(2)
					RtInfo, err := Utils.Redis2FaultVCI(k, v.RTpushCtr, v.Interval, modelFault[modelFaultKey], int64(modelIndex[modelFaultKey]))
					if err != nil {
						fmt.Printf("real time message create error : %s", err.Error())
						continue
					}
					v.WriteMsg(0x08, RtInfo)
					modelIndex[modelFaultKey]++

				}
			}
		}
	}
}

// RecvCommand
// -----------------------------------------------------------------------
// 定义Http接口，外部通过API修改线程内部状态
// -----------------------------------------------------------------------
func (t *ThreadVCIList) RecvCommand(r *gin.Engine) {
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

		v1.GET("/vci-register/ws", t.RegisterWS)
		v1.GET("/vci-charger/ws", t.ChargeWS)
		r.Use(middlewarePost)
		register := v1.Group("/vci-register")
		{
			register.POST("/build", t.buildRegisterHttp)
			register.POST("/change-state", t.changeIntervalAtRegister)
			register.POST("/register", t.registerRegister)
			register.POST("/heart", t.heartRegister)
			register.DELETE("/delete", t.deleteRegisterHttp)
		}
		charge := v1.Group("/vci-charger")
		{
			charge.POST("/build", t.buildChargeHttp)
			charge.POST("/change-state", t.changeIntervalAtCharge)
			charge.POST("/change-rtstatus", t.changeRTStatus)
			charge.DELETE("/delete", t.deleteChargeHttp)
		}
	}
}

/*接口区 START*/

// getSqlMessage
// -----------------------------------------------------------------------
// 从数据库获取当前模块的消息体 （暂不考虑使用）
// -----------------------------------------------------------------------
func (t *ThreadVCIList) getSqlMessage(c *gin.Context) {
	// 识别参数的数据
	var sqlMsg = database.SqlMessage{}
	c.ShouldBindBodyWith(&sqlMsg, binding.JSON)
	if sqlMsg.MessageType == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 0,
			"msg":  "所属类别不能为空",
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
			"code": 1,
			"data": err,
			"msg":  "错误",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": resqData,
		"msg":  "成功",
	})
}

// buildRegisterHttp
// -----------------------------------------------------------------------
// 创建VCI Manager线程
// -----------------------------------------------------------------------
func (t *ThreadVCIList) buildRegisterHttp(c *gin.Context) {
	// 识别参数ID
	var vciConfig = form.VCIRegisterConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)

	if _, ok := t.MainThread.AmountList[vciConfig.Id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体Id",
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

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
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

// registerRegister
// -----------------------------------------------------------------------
// 发送Manager线程注册帧
// -----------------------------------------------------------------------
func (t *ThreadVCIList) registerRegister(c *gin.Context) {
	// 识别参数的数据
	register := form.VCIRegister{}
	c.ShouldBindBodyWith(&register, binding.JSON)

	selfCheckState := 0
	selfCheckState = register.SelfCheckState

	if selfCheckState > 3 {
		selfCheckState = 3
	}

	if len(t.MainThread.WorkList) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处在工作中的线程",
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
		"code": 0,
		"msg":  "success",
	})
}

// heartRegister
// -----------------------------------------------------------------------
// 发送Manager线程心跳帧
// -----------------------------------------------------------------------
func (t *ThreadVCIList) heartRegister(c *gin.Context) {
	// 识别参数的数据
	heartBeat := form.VCIRegisterHeartBeat{}
	c.ShouldBindBodyWith(&heartBeat, binding.JSON)

	if _, ok := t.SonThread.WorkList[heartBeat.GunId]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": "1",
			"msg":  "没有处在工作中的线程",
		})
		return
	}
	register := t.MainThread.WorkList[1]

	// 逻辑异常处理
	if heartBeat.GunBasicState.LinkState > 0 && heartBeat.GunBasicState.PositionedState > 0 {
		heartBeat.GunBasicState.PositionedState = 0
	}

	// 心跳计数赋值
	if heartBeat.HeartBeatCnt == 0 {
		heartBeat.HeartBeatCnt = uint32(register.HeartbeatCtr)
	}

	// 心跳间隔赋值
	if heartBeat.HeartBeatPeriod == 0 {
		heartBeat.HeartBeatPeriod = register.RTpushCtr
	}

	// 当前时间戳赋值
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

		} else {
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
		"code": 0,
		"msg":  "success",
	})
}

// changeIntervalAtRegister
// -----------------------------------------------------------------------
// 动态修改Manager线程的心跳状态
// -----------------------------------------------------------------------
func (t *ThreadVCIList) changeIntervalAtRegister(c *gin.Context) {
	// 获取并绑定网络传输的数据
	vciConfig := form.VCIConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)

	if vciConfig.HeartBeatPeriod == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "心跳间隔时间不能为零",
		})
		return
	}
	if len(t.MainThread.WorkList) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处在工作中的线程",
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
		"code": 0,
		"msg":  "修改成功",
	})
}

// deleteRegisterHttp
// -----------------------------------------------------------------------
// 删除VCI Manager线程
// -----------------------------------------------------------------------
func (t *ThreadVCIList) deleteRegisterHttp(c *gin.Context) {
	// 识别参数ID
	var vciConfig = form.VCIConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)
	if _, ok := t.MainThread.WorkList[vciConfig.Id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体",
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
		"code": 0,
		"msg":  "success",
	})
}

// buildChargeHttp
// -----------------------------------------------------------------------
// 创建VCI Charger线程
// -----------------------------------------------------------------------
func (t *ThreadVCIList) buildChargeHttp(c *gin.Context) {
	// 识别参数ID
	var vciConfig = form.VCIConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)

	if _, ok := t.SonThread.AmountList[vciConfig.GunId]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体",
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

	register := Utils.VCIChargingRegister(uint32(t.SonThread.WorkList[vciConfig.GunId].ConnRemotePort))
	t.SonThread.WorkList[vciConfig.GunId].WriteMsg(0x01, register)
	t.SonThread.RtMessageList[vciConfig.GunId] = false

	// 启用协程监听、读取udp数据
	go t.SonThread.WorkList[vciConfig.GunId].ReadMsg()
	// 启用协程周期性发送 心跳数据
	go t.HeartBeatForCharge(vciConfig.GunId, t.SonThread.WorkList[vciConfig.GunId])

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// changeIntervalAtCharge
// -----------------------------------------------------------------------
// 动态修改Charger线程的心跳状态
// -----------------------------------------------------------------------
func (t *ThreadVCIList) changeIntervalAtCharge(c *gin.Context) {
	vcichBeat := form.ChargeHeartBeat{}
	c.ShouldBindBodyWith(&vcichBeat, binding.JSON)

	if vcichBeat.HeartBeatPeriod == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "心跳间隔时间不能为零",
		})
		return
	}
	if len(t.SonThread.WorkList) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处在工作中的线程",
		})
		return
	}

	if _, ok := t.SonThread.WorkList[vcichBeat.GunId]; !ok {
		if vcichBeat.GunId != 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code": 1,
				"msg":  "未识别的线程ID",
			})
			return
		}
		for _, v := range t.SonThread.WorkList {
			v.Mu.Lock()
			v.Interval = vcichBeat.HeartBeatPeriod
			v.Mu.Unlock()
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "修改成功",
		})
		return

	}

	client := t.SonThread.WorkList[vcichBeat.GunId]
	client.Mu.Lock()
	client.Interval = vcichBeat.HeartBeatPeriod
	client.Mu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "修改成功",
	})
	return
}

// changeRTStatus
// -----------------------------------------------------------------------
// 动态修改Charger线程的RT状态
// -----------------------------------------------------------------------
func (t *ThreadVCIList) changeRTStatus(c *gin.Context) {
	vciRTInfo := form.VCIRTInfo{}
	c.ShouldBindBodyWith(&vciRTInfo, binding.JSON)

	err := vciRTInfo.VerifyVCIRTInfo()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  err.Error(),
		})
		return
	}

	t.muThread.Lock()
	defer t.muThread.Unlock()
	if _, ok := t.SonThread.WorkList[vciRTInfo.GunId]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体Id",
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
		"code": 0,
		"msg":  "修改成功",
	})
}

// deleteChargeHttp
// -----------------------------------------------------------------------
// 删除VCI Charger线程
// -----------------------------------------------------------------------
func (t *ThreadVCIList) deleteChargeHttp(c *gin.Context) {
	// 识别参数ID
	var vciConfig = form.VCIConfig{}
	c.ShouldBindBodyWith(&vciConfig, binding.JSON)
	if _, ok := t.SonThread.WorkList[vciConfig.GunId]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体",
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
		"code": 0,
		"msg":  "success",
	})
}

// fault
// -----------------------------------------------------------------------
// 测试 redis 取枚举值
// -----------------------------------------------------------------------
func (t *ThreadVCIList) fault(c *gin.Context) {
	strName := c.Query("fault_list_name")
	listId := c.Query("fault_list_id")
	listIdInt, _ := strconv.Atoi(listId)

	data, err := Utils.Redis2FaultVCI(1, 10, 1000, strName, int64(listIdInt))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "成功",
		"data": data,
	})
	return
}

/*接口区 END*/

/*功能区 START*/
//fsm cvi 消息体初始化

// VCIMainLoginInit
// ------------------------------------------------------------------------
// 初始化Manager的注册数据帧 0x00
// ------------------------------------------------------------------------
func (t *ThreadVCIList) VCIMainLoginInit(state int) *fsmvci.VCIManagerRegisterInfo {
	return Utils.VCIRegister(state)
}

// VCImainHeartInit
// ------------------------------------------------------------------------
// 初始化Manager的心跳数据帧 0x02
// ------------------------------------------------------------------------
func (t *ThreadVCIList) VCImainHeartInit(client *udp.Client) *fsmvci.VCIManagerHeartBeatInfo {
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

// VCIChargingHeartbeatReqInit
// ------------------------------------------------------------------------
// 初始化Charger的心跳数据帧 0x06
// ------------------------------------------------------------------------
func (t *ThreadVCIList) VCIChargingHeartbeatReqInit(GunId uint32, client *udp.Client) *fsmvci.VCIChargerHeartBeatInfo {
	client.HeartbeatCtr++

	faultList := make([]*fsmvci.ChargerFaultState, 0)

	chargerFaultState := Utils.FaultStateAtVCI(0, 0)
	chargerFaultState2 := Utils.FaultStateAtVCI(0, 0)
	faultList = append(faultList, []*fsmvci.ChargerFaultState{chargerFaultState, chargerFaultState2}...)

	return Utils.VCIChargingHeart(GunId, uint32(client.HeartbeatCtr), client.Interval, faultList)
}

// VehicleChargingInterfaceLoginAnsInit
// ------------------------------------------------------------------------
// 初始化Charger的RT数据帧 0x08
// ------------------------------------------------------------------------
func (t *ThreadVCIList) VehicleChargingInterfaceLoginAnsInit(GunId uint32, client *udp.Client) *fsmvci.VCIChargerRTInfo {
	client.HeartbeatCtr++
	client.RTpushCtr++
	faultList := make([]*fsmvci.ChargerFaultState, 0)

	chargerFaultState := Utils.FaultStateAtVCI(0, 0)
	faultList = append(faultList, chargerFaultState)
	return Utils.VCIChargerRTInfo(GunId, client.RTpushCtr, client.Interval, faultList)
}

/*功能区 END*/
