package fsm

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	conf "FSMTestingPlatform/Conf"
	form "FSMTestingPlatform/Form"
	"FSMTestingPlatform/Protoc/fsmpmm"
	udp "FSMTestingPlatform/Udp"
	"FSMTestingPlatform/Utils"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gorilla/websocket"
)

// ThreadPMMList
// ----------------------------------------------------------------
// PMM结构体
// ----------------------------------------------------------------
type ThreadPMMList struct {
	muThread   sync.Mutex // 互斥锁
	MainLive   bool       // 第一线程在在线状态
	FatherLive bool       // 第二线程在线状态

	ContactorState       []uint32 // 主接触器状态
	ADModuleOnOffState   []uint32 // 模块状态
	MatrixContactorState []uint32 // 阵列接触器状态

	Interval uint32

	MainThread   *Thread // 第一线程对象实体
	FatherThread *Thread // 第二线程对象实体
}

// NewThreadPMMList
// ----------------------------------------------------------------
//	构建PMM结构体指针
// ----------------------------------------------------------------
func NewThreadPMMList(main, father *Thread) *ThreadPMMList {

	contactorState := conf.PMMContactorState

	// 随机从模块中取数据
	adModuleOnOffState := conf.PMMADModuleOnOffState[:Utils.RandValue(len(conf.PMMADModuleOnOffState))]
	// 随机从阵列接触器中取数据
	matrixContactorState := conf.PMMMatrixContactorState[:Utils.RandValue(len(conf.PMMMatrixContactorState))]

	return &ThreadPMMList{
		muThread:             sync.Mutex{},
		MainLive:             false,
		FatherLive:           false,
		ContactorState:       contactorState,
		ADModuleOnOffState:   adModuleOnOffState,
		MatrixContactorState: matrixContactorState,
		Interval:             200,
		MainThread:           main,
		FatherThread:         father,
	}
}

// ThreadPMMInit
// ----------------------------------------------------------------
// 初始化PMM结构体，返回其指针
// ----------------------------------------------------------------
func ThreadPMMInit() *ThreadPMMList {
	addrMain := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9030,
	}
	addrFather := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9040,
	}
	mainThread := NewThread(addrMain, 5)
	fatherThread := NewThread(addrFather, 5)

	return NewThreadPMMList(mainThread, fatherThread)
}

// HeartBeatForMatrix
// ----------------------------------------------------------------
// 检测工作线程、定时发送PMM主线程的心跳
// ----------------------------------------------------------------
func (t *ThreadPMMList) HeartBeatForMatrix(Id uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			// timer.C 管道类型数据。定时器功能，到指定时间会发出数据（time）。
			// 触发消息之后，要重新赋值下次出发的时间
			client.Timer = time.NewTimer(time.Duration(client.Interval) * time.Millisecond)
			if _, ok := t.MainThread.WorkList[Id]; !ok {
				// 工作列表中不存在线程信息，心跳中止
				goto HeartDie
			}
			// 发送心跳数据
			client.WriteMsg(0x04, t.PMMHeartbeatReqInit(Id, client))
		}
	}
HeartDie:
	fmt.Printf("主接触器断开 %d 心跳被中止 \n", Id)
}

// HeartBeatForContactor
// --------------------------------------------------------------
// 定时发送PMM父线程的心跳
// --------------------------------------------------------------
func (t *ThreadPMMList) HeartBeatForContactor(Id uint32, client *udp.Client) {
	for {
		select {
		case <-client.Timer.C:
			// timer.C 管道类型数据。定时器功能，到指定时间会发出数据（time）。
			// 触发消息之后，要重新赋值下次出发的时间
			client.Timer = time.NewTimer(time.Duration(client.Interval) * time.Millisecond)
			if _, ok := t.FatherThread.WorkList[Id]; !ok {
				// 工作列表中不存在线程信息，心跳中止
				goto HeartDie
			}
			// 发送心跳数据
			client.WriteMsg(0x06, t.MainContactorHeartbeatReqInit(Id, client, []int{}))
		}
	}
HeartDie:
	fmt.Printf("主接触器断开 %d 心跳被中止 \n", Id)
}

// RecvCommand
// ----------------------------------------------------------------
// 定义Http接口，外部通过API修改线程内部状态
// ----------------------------------------------------------------
func (t *ThreadPMMList) RecvCommand(r *gin.Engine) {
	// 输出gin日志到 gin_PMM 文件
	logfile, err := os.Create("./gin_PMM.log")
	if err != nil {
		fmt.Println("Could not create log file")
	}
	gin.SetMode(gin.DebugMode)
	gin.DefaultWriter = io.MultiWriter(logfile)

	// http 路由分组
	v1 := r.Group("/api/v1")
	{
		v1.GET("/pmm-matrix/ws", t.MatrixWS)
		v1.GET("/pmm-contactor/ws", t.ContactorWS)

		r.Use(middlewarePost)
		matrix := v1.Group("/pmm-matrix")
		{
			matrix.POST("/build", t.buildMatrixHttp)
			matrix.POST("/register-matrix", t.registerMatrixPower)
			matrix.POST("/register-model", t.registerADModel)
			matrix.POST("/heart", t.registerHeartBeat)
			matrix.POST("/change-state", t.changeRegister)
			matrix.POST("/delete", t.deleteMatrixHttp)
		}
		contactor := v1.Group("/pmm-contactor")
		{
			contactor.POST("/build", t.buildContactorHttp)
			contactor.POST("/heart", t.contactorHeartBeat)
			contactor.POST("/rt", t.contactorRT)
			contactor.POST("/change-state", t.changeContact)
			contactor.POST("/change-rtstate", t.changeRTContact)

			contactor.POST("/delete", t.deleteContactorHttp)
		}
	}
}

/*接口区域 START*/

// MatrixWS
// --------------------------------------------------------------
//	主线程的websocket接口
// --------------------------------------------------------------
func (t *ThreadPMMList) MatrixWS(c *gin.Context) {
	// 升级HTTP链接至WebSocket
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade error :", err)
		return
	}

	// 获取room参数，备后期需要
	roomIdStr := c.Query("room")
	roomId, _ := strconv.Atoi(roomIdStr)

	// 发送测试数据
	conn.WriteMessage(1, []byte("{\"成功\":\"hello\"}"))
	if t.MainThread.WorkList[uint32(roomId)] == nil {
		conn.WriteMessage(1, []byte("{\"失败\":\"内部错误, 无法识别枪线程\"}"))
		conn.Close()
		return
	}

	// 启用互斥锁
	client := t.MainThread.WorkList[uint32(roomId)]
	client.WSocket.Wsmu.Lock()

	// 当前client下 所有的 websocket 链接
	client.WSocket.WsConnList[conn.RemoteAddr().String()] = conn

	// 当前client下 当前组的 websocket 链接
	client.WSocket.WsRoom[roomId] = append(client.WSocket.WsRoom[roomId], conn.RemoteAddr().String())
	client.WSocket.Wsmu.Unlock()
	defer conn.Close()

	for k, v := range t.MainThread.WorkList {
		v.CheckWSMessage(int(k))
	}
}

// ContactorWS
// --------------------------------------------------------------
//	父线程的websocket的接口
// --------------------------------------------------------------
func (t *ThreadPMMList) ContactorWS(c *gin.Context) {
	// 升级HTTP链接至WebSocket
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade error :", err)
		return
	}

	// 获取room参数，备后期需要
	roomIdStr := c.Query("room")
	roomId, _ := strconv.Atoi(roomIdStr)

	// 发送测试数据
	conn.WriteMessage(1, []byte("{\"成功\":\"hello\"}"))
	if t.FatherThread.WorkList[uint32(roomId)] == nil {
		conn.WriteMessage(1, []byte("{\"失败\":\"内部错误, 无法识别枪线程\"}"))
		conn.Close()
		return
	}

	// 启用互斥锁
	client := t.MainThread.WorkList[uint32(roomId)]
	client.WSocket.Wsmu.Lock()

	// 当前client下 所有的 websocket 链接
	client.WSocket.WsConnList[conn.RemoteAddr().String()] = conn

	// 当前client下 当前组的 websocket 链接
	client.WSocket.WsRoom[roomId] = append(client.WSocket.WsRoom[roomId], conn.RemoteAddr().String())
	client.WSocket.Wsmu.Unlock()
	defer conn.Close()

	for k, v := range t.MainThread.WorkList {
		v.CheckWSMessage(int(k))
	}
}

// buildMatrixHttp
// ----------------------------------------------------------------
// 创建PMM主线程
// ----------------------------------------------------------------
func (t *ThreadPMMList) buildMatrixHttp(c *gin.Context) {
	// 识别参数ID
	var contactor = &form.PMMRegister{}
	c.ShouldBindBodyWith(contactor, binding.JSON)
	id := contactor.Id

	// 异常数据场景 信息返回前端
	if len(t.MainThread.WorkList) == 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "线程达到上限",
		})
		return
	}
	if _, ok := t.MainThread.WorkList[id]; ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "线程冲突",
		})
		return
	}
	if _, ok := t.MainThread.AmountList[id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别接触器Id",
		})
		return
	}

	// 数据值通过验证。启用互斥锁，防止并发
	t.muThread.Lock()
	if !t.MainLive {
		t.MainLive = true
	}
	// 从全局udp列表 COPY 到 work udp列表中
	t.MainThread.WorkList[id] = t.MainThread.AmountList[id]
	t.muThread.Unlock()

	// 启用协程监听、读取udp数据
	go t.MainThread.WorkList[id].ReadMsg()
	// 启用协程周期性发送 心跳数据
	go t.HeartBeatForMatrix(id, t.MainThread.WorkList[id])

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// registerMatrixPower
// ----------------------------------------------------------------
// 发送功率矩阵注册信息帧
// ----------------------------------------------------------------
func (t *ThreadPMMList) registerMatrixPower(c *gin.Context) {
	pmmRegisterPower := form.PMMRegisterPower{}
	err := c.ShouldBindBodyWith(&pmmRegisterPower, binding.JSON)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "数据解析错误：" + err.Error(),
		})
		return
	}

	client, ok := t.MainThread.WorkList[pmmRegisterPower.Id]
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处在工作中的线程",
		})
		return
	}
	powerMatrixLogin := Utils.PMMPowerMatrixLogin(pmmRegisterPower.State)
	client.WriteMsg(0x00, powerMatrixLogin)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// registerADModel
// ----------------------------------------------------------------
// 模块注册信息帧
// ----------------------------------------------------------------
func (t *ThreadPMMList) registerADModel(c *gin.Context) {
	pmmRegisterAdModule := form.PMMRegisterAdModule{}
	err := c.ShouldBindBodyWith(&pmmRegisterAdModule, binding.JSON)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "数据解析错误：" + err.Error(),
		})
		return
	}
	client, ok := t.MainThread.WorkList[pmmRegisterAdModule.Id]
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处在工作中的线程",
		})
		return
	}

	alarmLength := 0
	alarmList := make([]*fsmpmm.ADModuleAlarm, 0)
	alarmDataList := make([]*fsmpmm.AlarmDataType, 0)

	// 随机alarm故障在Redis中的下标
	alarmName := conf.PMMRedisListNameAlarm[Utils.RandValue(2)]
	// 从Redis中随机选取一个缓存值
	alarmKey, err := Utils.Redis2Data(alarmName, -1)

	if err != nil {
		alarmKey = []int{}
	}

	if pmmRegisterAdModule.IsAlarm {
		if pmmRegisterAdModule.AlarmLength > 6 {
			alarmLength = 6
		}
		alarmLength = pmmRegisterAdModule.AlarmLength

		for i := 0; i < alarmLength; i++ {
			alarmList = append(alarmList, Utils.PMMADModuleAlarm(i))
			Utils.PMMAlarmDataType(uint32(Utils.RandValue(6)+1), alarmKey)
		}
	}
	adModuleLogin := Utils.PMMADModuleLogin(alarmList, alarmDataList)
	client.WriteMsg(0x02, adModuleLogin)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// registerHeartBeat
// ----------------------------------------------------------------
// 发送主线程心跳帧
// ----------------------------------------------------------------
func (t *ThreadPMMList) registerHeartBeat(c *gin.Context) {
	heatBeat := form.PMMRegisterHeartBeat{}
	err := c.ShouldBindBodyWith(&heatBeat, binding.JSON)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "数据解析错误:" + err.Error(),
		})
		return
	}
	t.muThread.Lock()
	defer t.muThread.Unlock()
	client, ok := t.MainThread.WorkList[heatBeat.Id]
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处于工作中的线程",
		})
		return
	}

	if heatBeat.Interval > 0 {
		client.Interval = heatBeat.Interval
	}
	client.HeartbeatCtr = heatBeat.HeartbeatCtr

	client.WriteMsg(0x04, t.PMMHeartbeatReqInit(heatBeat.Id, client))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// changeRegister
// ----------------------------------------------------------------
// 修改PMM主线程的心跳状态
// ----------------------------------------------------------------
func (t *ThreadPMMList) changeRegister(c *gin.Context) {
	// 获取并绑定网络传输的数据
	pmmConfig := form.PMMConfig{}
	c.ShouldBindBodyWith(&pmmConfig, binding.JSON)

	if pmmConfig.Interval == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "心跳间隔时间不能为零",
		})
		return
	}
	t.muThread.Lock()
	defer t.muThread.Unlock()
	client, ok := t.MainThread.WorkList[pmmConfig.Id]
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处在工作中的线程",
		})
		return
	}
	// 修改所有工作中线程的间隔时长
	client.Interval = pmmConfig.Interval
	if pmmConfig.HeartbeatCtr > 0 {
		client.HeartbeatCtr = pmmConfig.HeartbeatCtr
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "修改成功",
	})
}

// deleteMatrixHttp
// ----------------------------------------------------------------
// 删除PMM主线程
// ----------------------------------------------------------------
func (t *ThreadPMMList) deleteMatrixHttp(c *gin.Context) {
	// 识别参数ID
	var contactor = &form.PMMRegister{}
	c.ShouldBindBodyWith(contactor, binding.JSON)
	id := contactor.Id

	// 异常数据场景 信息返回前端
	if _, ok := t.MainThread.WorkList[id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体",
		})
		return
	}
	//don't close the udp conn
	//t.MainThread.WorkList[VCIGun.Gun].Conn.Close()
	// 数据值通过验证。启用互斥锁，防止并发
	t.muThread.Lock()
	//从工作列表中删除当前的线程
	delete(t.MainThread.WorkList, id)
	if len(t.MainThread.WorkList) == 0 {
		t.MainLive = false
	}
	t.muThread.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// buildContactorHttp
// ----------------------------------------------------------------
// 创建第二线程的工作线程，最多可创建六个工作线程。由前端参数决定将要建立的线程ID
// ----------------------------------------------------------------
func (t *ThreadPMMList) buildContactorHttp(c *gin.Context) {
	// 识别参数ID
	var pmmRegister = &form.PMMRegister{}
	c.ShouldBindBodyWith(pmmRegister, binding.JSON)
	id := pmmRegister.Id

	// 异常数据场景 信息返回前端
	if len(t.FatherThread.WorkList) == 6 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "线程达到上限",
		})
		return
	}
	if _, ok := t.FatherThread.WorkList[id]; ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "线程冲突",
		})
		return
	}
	if _, ok := t.FatherThread.AmountList[id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体",
		})
		return
	}

	// 数据值通过验证。启用互斥锁，防止并发
	t.muThread.Lock()
	if !t.FatherLive {
		t.FatherLive = true
	}
	// 从全局udp列表 COPY 到 work udp列表中
	t.FatherThread.WorkList[id] = t.FatherThread.AmountList[id]
	t.muThread.Unlock()

	// 启用协程监听、读取udp数据
	go t.FatherThread.WorkList[id].ReadMsg()
	// 启用协程周期性发送 心跳数据
	go t.HeartBeatForContactor(id, t.FatherThread.WorkList[id])

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// contactorHeartBeat
// ----------------------------------------------------------------
// 发送PMM第二线程的心跳数据
// ----------------------------------------------------------------
func (t *ThreadPMMList) contactorHeartBeat(c *gin.Context) {
	hbeart := form.PMMContactorHeartBeat{}
	err := c.ShouldBindBodyWith(&hbeart, binding.JSON)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "数据解析错误：" + err.Error(),
		})
		return
	}
	client, ok := t.FatherThread.WorkList[hbeart.Id]
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处于工作中的线程",
		})
		return
	}
	alarmList := make([]int, 0)
	t.muThread.Lock()
	if hbeart.Interval > 0 {
		client.Interval = hbeart.Interval
	}

	if hbeart.HeartbeatCtr > 0 {
		client.HeartbeatCtr = hbeart.HeartbeatCtr
	}

	if len(hbeart.AlarmState) > 0 {
		alarmList = append(alarmList, hbeart.AlarmState[0])
	}
	t.muThread.Unlock()
	client.WriteMsg(0x06, t.MainContactorHeartbeatReqInit(hbeart.Id, client, alarmList))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// contactorRT
// ----------------------------------------------------------------
// 修改PMM第二线程的RT状态
// ----------------------------------------------------------------
func (t *ThreadPMMList) contactorRT(c *gin.Context) {
	pmmContactorRT := form.PMMContactorRT{}
	err := c.ShouldBindBodyWith(&pmmContactorRT, binding.JSON)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "数据解析错误：" + err.Error(),
		})
	}
	client, ok := t.FatherThread.WorkList[pmmContactorRT.Id]
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处于工作中的线程",
		})
		return
	}

	client.WriteMsg(0x08, t.MainContactorRTpushInit(pmmContactorRT.Id))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

// changeContact
// ----------------------------------------------------------------
// 修改PMM第二线程的心跳状态
// ----------------------------------------------------------------
func (t *ThreadPMMList) changeContact(c *gin.Context) {
	pmmBeat := form.PMMContactorHeartBeat{}
	c.ShouldBindBodyWith(&pmmBeat, binding.JSON)

	if pmmBeat.Interval == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "心跳间隔时间不能为零",
		})
		return
	}
	if len(t.FatherThread.WorkList) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "没有处在工作中的线程",
		})
		return
	}
	client, ok := t.FatherThread.WorkList[pmmBeat.Id]
	if !ok {
		if pmmBeat.Id != 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code": 1,
				"msg":  "未识别的线程ID",
			})
			return
		}
		for _, v := range t.FatherThread.WorkList {
			v.Mu.Lock()
			v.Interval = pmmBeat.Interval
			if pmmBeat.HeartbeatCtr > 0 {
				v.HeartbeatCtr = pmmBeat.HeartbeatCtr
			}
			v.Mu.Unlock()
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "修改成功",
		})
		return

	}
	client.Mu.Lock()
	client.Interval = pmmBeat.Interval
	if pmmBeat.HeartbeatCtr > 0 {
		client.HeartbeatCtr = pmmBeat.HeartbeatCtr
	}
	client.Mu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "修改成功",
	})
	return
}

// changeContact
// ----------------------------------------------------------------
// 修改PMM第二线程的RT状态
// ----------------------------------------------------------------
func (t *ThreadPMMList) changeRTContact(c *gin.Context) {
	pmmRTInfo := form.PMMRTInfo{}
	c.ShouldBindBodyWith(&pmmRTInfo, binding.JSON)

	err := pmmRTInfo.VerifyVCIRTInfo()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  err.Error(),
		})
		return
	}

	t.muThread.Lock()
	defer t.muThread.Unlock()
	if _, ok := t.FatherThread.WorkList[pmmRTInfo.Id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体Id",
		})
		return
	}
	if pmmRTInfo.Interval == 0 {
		pmmRTInfo.Interval = t.FatherThread.WorkList[pmmRTInfo.Id].Interval
	}
	if pmmRTInfo.PushCnt == 0 {
		pmmRTInfo.PushCnt = t.FatherThread.WorkList[pmmRTInfo.Id].RTpushCtr
	}

	t.FatherThread.RtMessageList[pmmRTInfo.Id] = pmmRTInfo.Status
	t.FatherThread.RtMessageInterval = pmmRTInfo.Interval

	t.FatherThread.WorkList[pmmRTInfo.Id].RTpushCtr = pmmRTInfo.PushCnt
	t.FatherThread.WorkList[pmmRTInfo.Id].Interval = pmmRTInfo.Interval

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "修改成功",
	})
}

// deleteContactorHttp
// ----------------------------------------------------------------
// 删除PMM第二线程
// ----------------------------------------------------------------
func (t *ThreadPMMList) deleteContactorHttp(c *gin.Context) {
	// 识别参数ID
	var pmmRegister = &form.PMMRegister{}
	c.ShouldBindBodyWith(pmmRegister, binding.JSON)
	id := pmmRegister.Id

	// 异常数据场景 信息返回前端
	if _, ok := t.FatherThread.WorkList[id]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": 1,
			"msg":  "未识别枪体",
		})
		return
	}

	// 数据值通过验证。启用互斥锁，防止并发
	t.muThread.Lock()
	//从工作列表中删除当前的线程
	delete(t.FatherThread.WorkList, id)
	if len(t.FatherThread.WorkList) == 0 {
		t.FatherLive = false
	}
	t.muThread.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

/*接口区 END*/

/*功能区 START*/

// PowerMatrixLoginInit
// ----------------------------------------------------------------
// 初始化功率矩阵注册信息帧(0x00)
// ----------------------------------------------------------------
func (t *ThreadPMMList) PowerMatrixLoginInit(state int) *fsmpmm.PowerMatrixLogin {
	return Utils.PMMPowerMatrixLogin(state)
}

// ADModuleLoginInit
// ----------------------------------------------------------------
// 模块注册信息(0x02)
// ----------------------------------------------------------------
func (t *ThreadPMMList) ADModuleLoginInit() *fsmpmm.ADModuleLogin {

	adModuleAlarmList := make([]*fsmpmm.ADModuleAlarm, 0)

	// 随机alarm故障在Redis中的下标
	alarmName := conf.PMMRedisListNameAlarm[Utils.RandValue(2)]
	// 从Redis中随机选取一个缓存值
	alarmKey, err := Utils.Redis2Data(alarmName, -1)

	if err != nil {
		fmt.Printf("Redis to data is err : %s", err.Error())
		return nil
	}
	for _, v := range alarmKey {
		adModuleAlarmList = append(adModuleAlarmList, Utils.PMMADModuleAlarm(v))
	}

	alarmDataTypeList := make([]*fsmpmm.AlarmDataType, 0)
	alarmDataTypeList = append(alarmDataTypeList, Utils.PMMAlarmDataType(uint32(Utils.RandValue(6)+1), alarmKey))
	return Utils.PMMADModuleLogin(adModuleAlarmList, alarmDataTypeList)
}

// PMMHeartbeatReqInit
// -----------------------------------------------------------------
// 初始化心跳状态同步(0x04)
// -----------------------------------------------------------------
func (t *ThreadPMMList) PMMHeartbeatReqInit(id uint32, client *udp.Client) *fsmpmm.PMMHeartbeatReq {
	client.HeartbeatCtr++
	heartBeat := client.HeartbeatCtr
	//
	mainStatusList := make([]*fsmpmm.MainStatus, 0)
	matrixStatusList := make([]*fsmpmm.MatrixStatus, 0)

	adModuleAList := make([]*fsmpmm.ADModuleAttr, 0)
	adModulePList := make([]*fsmpmm.ADModuleParam, 0)
	adModuleAlarm := make([]*fsmpmm.ADModuleAlarm, 0)
	adModuleOnOffState := make([]fsmpmm.ADModuleOnOffStateEnum, 0)

	alarmFault := make([]fsmpmm.FaultStopEnum, 0)
	alarmArray := make([]fsmpmm.ArrayContactorAlarm, 0)

	t.muThread.Lock()

	// 随机从Redis下标
	alarmName := conf.PMMRedisListNameAlarm[Utils.RandValue(2)]
	// 随机从Redis中选取一个缓存值
	alarmKey, _ := Utils.Redis2Data(alarmName, -1)

	alarmArray = append(alarmArray, fsmpmm.ArrayContactorAlarm(1), fsmpmm.ArrayContactorAlarm(0))
	for _, v := range alarmKey {
		alarmFault = append(alarmFault, fsmpmm.FaultStopEnum(v))
		adModuleAlarm = append(adModuleAlarm, Utils.PMMADModuleAlarm(v))
	}
	if _, ok := t.MainThread.WorkList[id]; !ok {
		mainStatus := Utils.PMMMainStatus(id, t.MatrixContactorState, t.ADModuleOnOffState, alarmFault)
		mainStatusList = append(mainStatusList, mainStatus)
	}
	t.muThread.Unlock()

	//
	matrixStatusList = append(matrixStatusList, Utils.PMMMatrixStatus(1, alarmArray))
	//
	adModuleAList = append(adModuleAList, Utils.PMMADModuleAttr(id))
	//
	adModulePList = append(adModulePList, Utils.PMMADModuleParam(id))
	//
	for i := 0; i < len(mainStatusList); i++ {
		adModuleOnOffState = append(adModuleOnOffState, Utils.PMMADModuleOnOffState())
	}

	cTime := uint64(time.Now().UnixMilli())
	return Utils.PMMHeartbeatReq(uint32(heartBeat), client.Interval, cTime, mainStatusList, matrixStatusList,
		adModuleAList, adModulePList, adModuleAlarm)
}

// MainContactorHeartbeatReqInit
// -----------------------------------------------------------------
// 主接触器线程心跳周期信息帧(0x06)
// -----------------------------------------------------------------
func (t *ThreadPMMList) MainContactorHeartbeatReqInit(id uint32, client *udp.Client, alarmAns []int) *fsmpmm.MainContactorHeartbeatReq {
	t.FatherThread.WorkList[id].HeartbeatCtr++
	hbeatCtr := t.FatherThread.WorkList[id].HeartbeatCtr
	cTime := uint64(time.Now().UnixMilli())

	mainMode := Utils.RandValue(8)
	mainContactorhbeatReq := Utils.PMMMainContactorHeartbeatReq(id, uint32(hbeatCtr), client.Interval, cTime, fsmpmm.ContactorStateEnum(mainMode), t.ADModuleOnOffState, t.MatrixContactorState)
	if len(alarmAns) > 0 {
		mainContactorhbeatReq.AlarmAnsList = fsmpmm.FaultStopEnum(alarmAns[0])
	}
	return mainContactorhbeatReq
}

// MainContactorRTpushInit
// -----------------------------------------------------------------
// 主接触器线程突发上传信息帧(0x08)
// -----------------------------------------------------------------
func (t *ThreadPMMList) MainContactorRTpushInit(id uint32) *fsmpmm.MainContactorRTpush {
	client := t.FatherThread.WorkList[id]
	client.RTpushCtr++

	state := Utils.RandValue(8)
	fault := Utils.RandValue(15)

	return Utils.PMMMainContactorRTpush(id, client.RTpushCtr, t.Interval, fsmpmm.ContactorStateEnum(state), fsmpmm.FaultStopEnum(fault))
}

// RtStart
// ----------------------------------------------------------------
// 修改第二线程的RT状态
// -----------------------------------------------------------------
func (t *ThreadPMMList) RtStart() {
	// 指针
	for {
		select {
		case <-t.FatherThread.RtTimer.C:
			t.FatherThread.RtTimer = time.NewTimer(time.Duration(t.FatherThread.RtMessageInterval) * time.Millisecond)
			for k, _ := range t.FatherThread.WorkList {
				if t.FatherThread.RtMessageList[k] {
					t.FatherThread.WorkList[k].WriteMsg(0x08, t.MainContactorRTpushInit(k))
				}
			}
		}
	}

}

/*功能区 END*/
