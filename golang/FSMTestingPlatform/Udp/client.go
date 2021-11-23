package udp

import (
	database "FSMTestingPlatform/Database"
	"FSMTestingPlatform/Utils"
	"FSMTestingPlatform/pool"
	"FSMTestingPlatform/protoc/fsmohp"
	"FSMTestingPlatform/protoc/fsmpmm"
	"FSMTestingPlatform/protoc/fsmvci"

	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

type Client struct {
	Mu sync.Mutex
	ConnRemotePort int			// 远端（服务端）端口号
	ConnType string				// 链接类型 VCI PMM OHP
	Interval uint32				// 数据间隔时长
	HeartbeatCtr int64			// 心跳计数
	RTpushCtr	uint32			// 推送计数
	//CurrentTime uint64			// 当前时间戳
	WsMessageChan chan []byte	// websocket 消息管道

	RegisterMessageChan		map[uint]*RegisterMessageChan  // 注册 消息管道 消息暂存地址
	HeartBeatMessageChan	map[uint]*HeartBeatMessageChan // 心跳 消息管道 消息暂存地址
	RealTimeMessageChan 	map[uint]*RealTimeMessageChan  // realtime 消息管道 消息暂存地址

	WSocket *Wsocket			// WS对象
	Conn *net.UDPConn			// UDP链接
	Timer *time.Timer			// 定时计数器
}

// RegisterMessageChan 注册消息 暂存地址
type RegisterMessageChan struct {
	SendRegisterChan proto.Message		// 发送的数据内容
	SendRegisterTime int64				// 存储时间
}

// HeartBeatMessageChan 注册消息 暂存地址
type HeartBeatMessageChan struct {
	SendHeartBeatChan proto.Message		// 发送的数据内容
	SendHeartBeatTime int64				// 存储时间
}

// RealTimeMessageChan 注册消息 暂存地址
type RealTimeMessageChan struct {
	SendRealTimeChan proto.Message		// 发送的数据内容
	SendRealTimeTime int64				// 存储时间
}

// Wsocket web socket 数据接收
type Wsocket struct {
	Wsmu sync.Mutex
	WsRoom map[int][]string					//WebSocket 分组
	WsConnList map[string]*websocket.Conn	//WebSocket链接
}

// NewRegisterMessageRam
// ----------------------------------------------------------------
// 初始化注册消息的暂存地址
// ----------------------------------------------------------------
func NewRegisterMessageRam () *RegisterMessageChan {
	var protoMsg proto.Message
	return &RegisterMessageChan{
		SendRegisterChan: protoMsg,
		SendRegisterTime: time.Now().UnixMilli(),
	}
}

// NewHeartBeatMessageRam
// ----------------------------------------------------------------
// 初始化心跳消息的暂存地址
// ----------------------------------------------------------------
func NewHeartBeatMessageRam () *HeartBeatMessageChan {
	var protoMsg proto.Message
	return &HeartBeatMessageChan{
		SendHeartBeatChan: protoMsg,
		SendHeartBeatTime: time.Now().UnixMilli(),
	}
}

// NewRealTimeMessageRam
// ----------------------------------------------------------------
// 初始化RT消息的暂存地址
// ----------------------------------------------------------------
func NewRealTimeMessageRam () *RealTimeMessageChan {
	var protoMsg proto.Message
	return &RealTimeMessageChan{
		SendRealTimeChan: protoMsg,
		SendRealTimeTime: time.Now().UnixMilli(),
	}
}

// NewClient
// ----------------------------------------------------------------
// 返回Client指针
// ----------------------------------------------------------------
func NewClient (pool *pool.UDPpool) (*Client, error) {
	Wsocket := &Wsocket{
		WsRoom: make(map[int][]string, 0),
		WsConnList: make(map[string]*websocket.Conn),
	}
	udpConn, err := pool.Open()
	if err != nil {
		return nil, err
	}
	// serverPortInt 当前链接 服务器端port， 后续代码需要用port区分
	serverIpAndPorts := udpConn.RemoteAddr().String()
	serverIpAndPort := strings.Split(serverIpAndPorts, ":")
	serverPort := serverIpAndPort[len(serverIpAndPort)-1]
	serverPortInt, _ := strconv.Atoi(serverPort)

	connType := ""
	vciPort := make(map[int]int)
	pmmPort := make(map[int]int)
	ohpPort := make(map[int]int)
	portProtoc, err := Utils.ReadFile("./Conf/protoc.json", "PORT")

	if err == nil {
		vciPort = portFromJson(portProtoc, "VCI")
		pmmPort = portFromJson(portProtoc, "PMM")
		ohpPort = portFromJson(portProtoc, "OHP")
	}
	if _, ok := vciPort[serverPortInt]; ok {
		connType = "VCI"
	}
	if _, ok := pmmPort[serverPortInt]; ok {
		connType = "PMM"
	}
	if _, ok := ohpPort[serverPortInt]; ok {
		connType = "OHP"
	}

	client := &Client{
		ConnRemotePort: serverPortInt,
		ConnType: connType,
		Conn : udpConn,
		Interval:1000,
		HeartbeatCtr: 0,
		RTpushCtr: 0,
		WsMessageChan: make(chan []byte),
		RegisterMessageChan: make(map[uint]*RegisterMessageChan),
		HeartBeatMessageChan: make(map[uint]*HeartBeatMessageChan),
		RealTimeMessageChan: make(map[uint]*RealTimeMessageChan),
		WSocket: Wsocket,
		//CurrentTime: uint64(time.Now().UnixMilli()),
		Timer: time.NewTimer(1000*time.Millisecond),
	}

	if  _, ok := vciPort[serverPortInt]; ok {

		// 注册信息帧
		registerMessageChan := NewRegisterMessageRam()

		client.RegisterMessageChan[0x00] = registerMessageChan

		// 心跳信息帧 // 不同线程中的 只存在一个初始化内存地址
		heartBeatMessageChan := NewHeartBeatMessageRam()

		client.HeartBeatMessageChan[0x02] = heartBeatMessageChan

		client.HeartBeatMessageChan[0x04] = heartBeatMessageChan

		client.HeartBeatMessageChan[0x06] = heartBeatMessageChan

		// realtime 信息帧
		realTimeMessageChan := NewRealTimeMessageRam()
		client.RealTimeMessageChan[0x08] = realTimeMessageChan
	}

	if _, ok := pmmPort[serverPortInt]; ok {

		// 注册信息帧 //同一个线程定义了两个注册，要成指向不同的内存地址
		registerMessageChan := NewRegisterMessageRam()

		client.RegisterMessageChan[0x00] = registerMessageChan

		registerMessageChan1 := NewRegisterMessageRam()
		client.RegisterMessageChan[0x02] = registerMessageChan1

		// 心跳信息帧 // 不同线程中的 只存在一个初始化内存地址
		heartBeatMessageChan := NewHeartBeatMessageRam()
		client.HeartBeatMessageChan[0x04] = heartBeatMessageChan

		client.HeartBeatMessageChan[0x06] = heartBeatMessageChan

		// realtime 信息帧
		realTimeMessageChan := NewRealTimeMessageRam()
		client.RealTimeMessageChan[0x08] = realTimeMessageChan
	}

	if _, ok := ohpPort[serverPortInt]; ok {

		// 注册信息帧 //同一个线程定义了两个注册，要成指向不同的内存地址
		registerMessageChan := NewRegisterMessageRam()

		client.RegisterMessageChan[0x00] = registerMessageChan

		// 心跳信息帧 // 不同线程中的 只存在一个初始化内存地址
		heartBeatMessageChan := NewHeartBeatMessageRam()
		client.HeartBeatMessageChan[0x04] = heartBeatMessageChan

		// realtime 信息帧
		realTimeMessageChan := NewRealTimeMessageRam()
		client.RealTimeMessageChan[0x08] = realTimeMessageChan
	}
	return client, nil
}

// ReadMsg  Client
// ----------------------------------------------------------------
// 客户端从服务端接收数据
// ----------------------------------------------------------------
func (c *Client) ReadMsg () {
	for {
		bytes := make([]byte, 4096)
		_, _, err := c.Conn.ReadFrom(bytes)
		if err != nil {
			fmt.Println("client read error:", err.Error())
			return
		}
		// 共用同一个服务器，且不同客户端通讯的COMMAND一致
		// 后续代码需要以端口号区分
		protoMsg, err := Utils.DeCodeByProto(c.ConnRemotePort, bytes)
		if err != nil {
			return
		}
		msgTypeCode := Utils.ByteToUint8(bytes[3:4])
		if c.ConnType == "VCI" {
			c.readMsgFromVCI(msgTypeCode, protoMsg)
		}
		if c.ConnType == "PMM" {
			c.readMsgFromPMM(msgTypeCode, protoMsg)
		}
		if c.ConnType == "OHP" {
			c.readMsgFromOHP(msgTypeCode, protoMsg)
		}
	}
}

func (c *Client) readMsgFromVCI (msgTypeCode uint8, protoMsg proto.Message) {
	switch msgTypeCode {
	case 0x80:
		//vCILoginAns := protoMsg.(*fsmvci.VCIManagerRegisterReponse)
		fmt.Printf("this is server return message 0x80 %+v \n", protoMsg)
		//go c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckRegisterMessageRam(0x00, protoMsg)
	case 0x82:

		vCIHeartAns := protoMsg.(*fsmvci.VCIManagerHeartBeatReponse)
		if c.HeartbeatCtr + 3  < int64(vCIHeartAns.ReHeartbeatCnt)  || c.HeartbeatCtr -3 > int64(vCIHeartAns.ReHeartbeatCnt) {
			fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
			return
		}
		c.HeartbeatCtr = int64(vCIHeartAns.ReHeartbeatCnt)
		c.Interval = vCIHeartAns.ReHeartBeatPeriod
		fmt.Printf("this is server return message 0x82 %v \n", vCIHeartAns)
		//go c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckHeartMessageRam(0x02, protoMsg)

	//case 0x84:
	//	pluggedHeartbeatAns := protoMsg.(*fsmvci.VCIPluggedHeartbeatAns)
	//	if c.HeartbeatCtr + 3  < int64(pluggedHeartbeatAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(pluggedHeartbeatAns.HeartbeatCtr.Value) {
	//		fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
	//		return
	//	}
	//	c.HeartbeatCtr = int64(pluggedHeartbeatAns.HeartbeatCtr.GetValue())
	//	c.Interval = pluggedHeartbeatAns.Interval.GetValue()
	//	fmt.Printf("this is server return message 0x84 %v \n", pluggedHeartbeatAns)
	//	go c.sendWSChannel(c.WsMessageChan, protoMsg)
	//	c.CheckHeartMessageRam(0x04, protoMsg)

	case 0x86:
		vCIChargingHeartbeatAns := protoMsg.(*fsmvci.VCIChargerHeartBeatResponse)
		c.HeartbeatCtr = int64(vCIChargingHeartbeatAns.ReHeartBeatCnt)
		c.Interval = vCIChargingHeartbeatAns.ReHeartBeatPeriod
		fmt.Printf("this is server return message 0x86 %v \n", vCIChargingHeartbeatAns)
		//go c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckHeartMessageRam(0x06, protoMsg)
	case 0x88:
		vCIChargingRTpull := protoMsg.(*fsmvci.VCIChargerRTRsponse)
		fmt.Printf("this is server return message 0x88 %v \n", vCIChargingRTpull)
		//go c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckRealTimeMessageRam(0x08, protoMsg)
	}
}

func (c *Client) readMsgFromPMM (msgTypeCode uint8, protoMsg proto.Message) {
	switch msgTypeCode {

	case 0x80:
		powerMatrixLoginAns := protoMsg.(*fsmpmm.PowerMatrixLoginAns)
		fmt.Printf("this is server return message 0x80 %+v \n", powerMatrixLoginAns)
		go c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckRegisterMessageRam(0x00, protoMsg)
	case 0x82:
		adModuleLoginAns := protoMsg.(*fsmpmm.ADModuleLoginAns)
		//if c.HeartbeatCtr + 3  < int64(vCIHeartAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(vCIHeartAns.HeartbeatCtr.Value) {
		//	fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
		//	return
		//}
		fmt.Printf("this is server return message 0x82 %v \n", adModuleLoginAns)
		go c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckRegisterMessageRam(0x02, protoMsg)
	case 0x84:
		pmmHeartbeatAns := protoMsg.(*fsmpmm.PMMHeartbeatAns)
		c.HeartbeatCtr = int64(pmmHeartbeatAns.HeartbeatCtr)
		//if c.HeartbeatCtr + 3  < int64(pluggedHeartbeatAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(pluggedHeartbeatAns.HeartbeatCtr.Value) {
		//	fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
		//	return
		//}
		c.Interval = pmmHeartbeatAns.Interval
		fmt.Printf("this is server return message 0x84 %v \n", protoMsg)
		go c.sendWSChannel(c.WsMessageChan, protoMsg)

		c.CheckHeartMessageRam(0x04, protoMsg)
	case 0x86:
		mainHeartbeat := protoMsg.(*fsmpmm.MainContactorHeartbeatAns)
		c.Interval = mainHeartbeat.Interval
		fmt.Printf("this is server return message 0x86 %v \n", protoMsg)
		go c.sendWSChannel(c.WsMessageChan, protoMsg)

		c.CheckHeartMessageRam(0x06, protoMsg)
	case 0x88:
		mainRTPull := protoMsg.(*fsmpmm.MainContactorRTpull)
		c.RTpushCtr = mainRTPull.RTpullCtr
		fmt.Printf("this is server return message 0x88 %v \n", protoMsg)
		go c.sendWSChannel(c.WsMessageChan, protoMsg)

		c.CheckRealTimeMessageRam(0x08, protoMsg)

	}
}

func (c *Client) readMsgFromOHP (msgTypeCode uint8, protoMsg proto.Message) {
	switch msgTypeCode {
	case 0x80:
		//vCILoginAns := protoMsg.(*fsmohp.OrderPipelineLoginAns)
		fmt.Printf("this is server return message 0x80 %+v \n", protoMsg)
		//c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckRegisterMessageRam(0x00, protoMsg)
	case 0x82:
		orderHeart := protoMsg.(*fsmohp.OrderPipelineHeartbeatAns)
		if c.HeartbeatCtr + 3  < int64(orderHeart.HeartbeatCtr)  || c.HeartbeatCtr -3 > int64(orderHeart.HeartbeatCtr) {
			fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
			return
		}
		c.HeartbeatCtr = int64(orderHeart.HeartbeatCtr)
		c.Interval = orderHeart.Interval
		fmt.Printf("this is server return message 0x82 %v \n", orderHeart)
		//c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckHeartMessageRam(0x02, protoMsg)
	case 0x84:
		pluggedHeartbeatAns := protoMsg.(*fsmohp.OrderPipelineRTpull)
		if c.HeartbeatCtr + 3  < int64(pluggedHeartbeatAns.RTpullCtr)  || c.HeartbeatCtr -3 > int64(pluggedHeartbeatAns.RTpullCtr) {
			fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
			return
		}
		c.RTpushCtr = pluggedHeartbeatAns.RTpullCtr
		c.Interval = pluggedHeartbeatAns.Interval
		fmt.Printf("this is server return message 0x84 %v \n", pluggedHeartbeatAns)
		//c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckHeartMessageRam(0x04, protoMsg)
	}
}

// WriteMsg
// 客户端向服务端发送数据
func (c *Client) WriteMsg (msgType uint, protoMsg proto.Message) {
	bytes, err := Utils.EnCodeByProto(msgType, protoMsg)
	if err != nil {
		fmt.Printf("client write data error: %s", err.Error())
		return
	}
	_, err = c.Conn.Write(bytes)
	if err != nil {
		fmt.Printf("client write data error: %s", err.Error())
		return
	}
	c.sendDataRam(msgType, protoMsg)
}

// CheckWSMessage
// 检测消息管道内的数据，取出后
// Web Sockert 管道内的数据，并发送到前端
func (c *Client) CheckWSMessage (roomId int) {
	for{
		select {
		case dataByte := <-c.WsMessageChan:
			c.WSocket.Wsmu.Lock()
			connList := c.WSocket.WsRoom[roomId]
			for _, v := range connList {
				conn, _ := c.WSocket.WsConnList[v]
				err := conn.WriteMessage(1, dataByte)
				if err != nil {
					delete(c.WSocket.WsConnList, v)
					delete(c.WSocket.WsRoom, roomId)
					log.Println("goodbye:", v)
					break
				}
			}
			c.WSocket.Wsmu.Unlock()
		}
	}
}

// 返回的数据发送给WS管道，返回给前端
func (c *Client) sendWSChannel (channel chan []byte, msg proto.Message) {
	messageChan, _ := json.Marshal(msg)
	channel<-messageChan
}

// msgType 消息体Command
// protocMsgOld 消息体发送的信息
// protocMsgNew 消息体接收的信息
// costTime 收发过程消耗的时间
// isPass 测试用例是否通过 (PS: 心跳信息只保存未通过用例)
func (c *Client) writeToMySQL (msgType uint, protocMsgOld, protocMsgNew proto.Message, costTime int64, isPass int, noPassReason string) error {
	testCollection := database.TestCollection{}
	switch c.ConnType {
	case "VCI":
		testCollection.ProjectId = 1
		if msgType == 0x00 {
			testCollection.MessageType = 1
		}else if msgType == 0x08 {
			testCollection.MessageType = 3
		}else{
			testCollection.MessageType = 2
		}
	case "PMM":
		testCollection.ProjectId = 2
		if msgType == 0x00 || msgType == 0x02 {
			testCollection.MessageType = 1
		}else if msgType == 0x08 {
			testCollection.MessageType = 3
		}else{
			testCollection.MessageType = 2
		}
	case "OHP":
		testCollection.ProjectId = 3
	default:
		return errors.New("未知类型消息")
	}

	testCollection.TestDate = time.Now().Format("2006-01-02 15:04:05")
	testCollection.CurrentTime = time.Now().UnixMilli()
	testCollection.TimeCost = costTime
	testCollection.IsPass = isPass
	testCollection.NoPassReason = noPassReason

	protocStrOld, err := Utils.PB2Json(protocMsgOld)
	protocStrNew, err := Utils.PB2Json(protocMsgNew)

	if err != nil {
		return err
	}
	testCollection.InputMessage = protocStrOld
	testCollection.OutputMessage = protocStrNew
	_, err  = database.MySQL.Table("test_collection").Insert(&testCollection)
	if err != nil{
		fmt.Println(err)
		return err
	}
	return nil
}

// sendDataRam
// 数据添加到内存
func (c *Client) sendDataRam (msgType uint, protocMsg proto.Message) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.ConnType == "VCI" {
		if msgType == 0x00 {
			c.RegisterMessageChan[msgType] = NewRegisterMessageRam()
			c.RegisterMessageChan[msgType].SendRegisterChan = protocMsg
		}
		if msgType == 0x02 || msgType == 0x04 || msgType == 0x06 {
			c.HeartBeatMessageChan[msgType] = NewHeartBeatMessageRam()
			c.HeartBeatMessageChan[msgType].SendHeartBeatChan = protocMsg
		}
		if msgType == 0x08 {
			c.RealTimeMessageChan[msgType] = NewRealTimeMessageRam()
			c.RealTimeMessageChan[msgType].SendRealTimeChan = protocMsg
		}
	}
	if c.ConnType == "PMM" {
		if msgType == 0x00 {
			c.RegisterMessageChan[msgType] = NewRegisterMessageRam()
			c.RegisterMessageChan[msgType].SendRegisterChan = protocMsg
		}
		if msgType == 0x02 {
			c.RegisterMessageChan[msgType] = NewRegisterMessageRam()
			c.RegisterMessageChan[msgType].SendRegisterChan = protocMsg
		}
		if msgType == 0x04 || msgType == 0x06 {
			c.HeartBeatMessageChan[msgType] = NewHeartBeatMessageRam()
			c.HeartBeatMessageChan[msgType].SendHeartBeatChan = protocMsg
		}

		if msgType == 0x08 {
			c.RealTimeMessageChan[msgType] = NewRealTimeMessageRam()
			c.RealTimeMessageChan[msgType].SendRealTimeChan = protocMsg
		}
	}
}

// CheckRegisterMessageRam
// 监测注册消息管道数据存入数据库
func (c *Client) CheckRegisterMessageRam (msgType uint,  protocMsg proto.Message) {
	c.Mu.Lock()
	heartMsgOld := c.RegisterMessageChan[msgType].SendRegisterChan
	oldTime := c.RegisterMessageChan[msgType].SendRegisterTime
	c.Mu.Unlock()
	differenceTime := time.Now().UnixMilli() -  oldTime
	err := c.writeToMySQL(msgType, heartMsgOld, protocMsg, differenceTime, 1, "")
	if err != nil {
		fmt.Printf("Mysql error : %s", err.Error())
	}
}

// CheckHeartMessageRam
// 监测心跳消息管道是否合法数据
// 不合法存入数据库
func (c *Client) CheckHeartMessageRam (msgType uint,  protocMsg proto.Message) {
	c.Mu.Lock()
	heartMsg := c.HeartBeatMessageChan[msgType].SendHeartBeatChan
	c.Mu.Unlock()
	isPass, costTime := 0, uint64(0)
	noPassReason := ""
	if c.ConnType == "VCI" {
		if msgType == 0x02 {
			heartMsgOld := heartMsg.(*fsmvci.VCIManagerHeartBeatInfo)
			heartMsgNew := protocMsg.(*fsmvci.VCIManagerHeartBeatReponse)
			costTime = heartMsgNew.ReTimeNow - heartMsgOld.TimeNow

			heartCnt := heartMsgNew.ReHeartbeatCnt + 3
			heartCntReverse := heartMsgOld.HeartBeatCnt  + 3
			if (heartMsgOld.HeartBeatCnt > heartCnt || heartMsgNew.ReHeartbeatCnt > heartCntReverse) ||
				costTime > uint64(heartMsgOld.HeartBeatPeriod) {
				noPassReason = "心跳计数错误或者超时"
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}else{
				isPass = 1
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}
		}else{
			heartMsgOld := heartMsg.(*fsmvci.VCIChargerHeartBeatInfo)
			heartMsgNew := protocMsg.(*fsmvci.VCIChargerHeartBeatResponse)
			costTime = heartMsgNew.ReTimeNow - heartMsgOld.TimeNow

			heartCnt := heartMsgNew.ReHeartBeatCnt + 3
			heartCntReverse := heartMsgOld.HeartbeatCnt  + 3
			if (heartMsgOld.HeartbeatCnt > heartCnt || heartMsgNew.ReHeartBeatCnt > heartCntReverse) ||
				costTime > uint64(heartMsgOld.HeartBeatPeriod) {
				noPassReason = "心跳计数错误或者超时"
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}else{
				isPass = 1
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}

		}
		//else if msgType == 0x04 {
		//	heartMsgOld := heartMsg.(*fsmvci.VCIPluggedHeartbeatReq)
		//	heartMsgNew := protocMsg.(*fsmvci.VCIPluggedHeartbeatAns)
		//	if heartMsgOld.HeartbeatCtr.GetValue() + 3 < heartMsgNew.HeartbeatCtr.GetValue() || heartMsgOld.HeartbeatCtr.GetValue() - 3 > heartMsgNew.HeartbeatCtr.GetValue() ||
		//		heartMsgOld.CurrentTime.Time + uint64(heartMsgOld.Interval.GetValue()) < heartMsgNew.CurrentTime.Time {
		//		costTime := heartMsgNew.CurrentTime.Time - heartMsgOld.CurrentTime.Time
		//		// TODO 存入数据库
		//		c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), 0)
		//	}
		//}
	}
	if c.ConnType == "PMM" {
		if msgType == 0x04 {
			heartMsgOld := heartMsg.(*fsmpmm.PMMHeartbeatReq)
			heartMsgNew := protocMsg.(*fsmpmm.PMMHeartbeatAns)
			costTime := heartMsgNew.CurrentTime - heartMsgOld.CurrentTime

			heartCnt := heartMsgNew.HeartbeatCtr + 3
			heartCntReverse := heartMsgOld.HeartbeatCtr  + 3
			if (heartMsgOld.HeartbeatCtr > heartCnt || heartMsgNew.HeartbeatCtr > heartCntReverse) ||
				costTime > uint64(heartMsgOld.Interval) {
				noPassReason = "心跳计数错误或者超时"
				// TODO 存入数据库
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}else{
				isPass = 1
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}
		}else{
			heartMsgOld := heartMsg.(*fsmpmm.MainContactorHeartbeatReq)
			heartMsgNew := protocMsg.(*fsmpmm.MainContactorHeartbeatAns)
			costTime := heartMsgNew.CurrentTime - heartMsgOld.CurrentTime

			heartCnt := heartMsgNew.HeartbeatCtr + 3
			heartCntReverse := heartMsgOld.HeartbeatCtr  + 3
			if (heartMsgOld.HeartbeatCtr < heartCnt || heartMsgNew.HeartbeatCtr < heartCntReverse) ||
				costTime > uint64(heartMsgOld.Interval) {
				noPassReason = "心跳计数错误或者超时"
				// TODO 存入数据库
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}else{
				isPass = 1
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}
		}

		if c.ConnType == "OHP" {
			heartMsgOld := heartMsg.(*fsmohp.OrderPipelineHeartbeatReq)
			heartMsgNew := protocMsg.(*fsmohp.OrderPipelineHeartbeatAns)
			costTime := heartMsgNew.CurrentTime - heartMsgOld.CurrentTime

			heartCnt := heartMsgNew.HeartbeatCtr + 3
			heartCntReverse := heartMsgOld.HeartbeatCtr  + 3
			if (heartMsgOld.HeartbeatCtr > heartCnt || heartMsgNew.HeartbeatCtr > heartCntReverse) ||
				costTime > uint64(heartMsgOld.Interval) {
				noPassReason = "心跳计数错误或者超时"
				// TODO 存入数据库
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}else{
				isPass = 1
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), isPass, noPassReason)
			}
		}
	}
}

// CheckRealTimeMessageRam
// 监测realtime消息管道数据存入数据库
func (c *Client) CheckRealTimeMessageRam (msgType uint, protocMsg proto.Message) {
	isPass := 1
	noPassReason := ""
	c.Mu.Lock()
	heartMsgOld := c.RealTimeMessageChan[msgType].SendRealTimeChan
	oldTime := c.RealTimeMessageChan[msgType].SendRealTimeTime
	c.Mu.Unlock()
	// differenceTime 两帧实时消息的时间差
	differenceTime := time.Now().UnixMilli() -  oldTime

	if c.ConnType == "VCI" {
		vciRTpush, ok := heartMsgOld.(*fsmvci.VCIChargerRTRsponse)
		if ok {
			if differenceTime > int64(vciRTpush.EchoRtPeriod) {
				isPass = 0
				noPassReason = "超时"
			}
		}
		pmmRTpush, ok := heartMsgOld.(*fsmpmm.MainContactorRTpush)
		if ok {
			if differenceTime > int64(pmmRTpush.Interval) {
				isPass = 0
				noPassReason = "超时"
			}
		}
		ohpRTpush, ok := heartMsgOld.(*fsmohp.OrderPipelineRTpush)
		if ok {
			if differenceTime > int64(ohpRTpush.Interval) {
				isPass = 0
				noPassReason = "超时"
			}
		}
	}
	if c.ConnType == "PMM" {
		vciRTpush, ok := heartMsgOld.(*fsmvci.VCIChargerRTRsponse)
		if ok {
			if differenceTime > int64(vciRTpush.EchoRtPeriod) {
				isPass = 0
				noPassReason = "超时"
			}
		}
		pmmRTpush, ok := heartMsgOld.(*fsmpmm.MainContactorRTpush)
		if ok {
			if differenceTime > int64(pmmRTpush.Interval) {
				isPass = 0
				noPassReason = "超时"
			}
		}
		ohpRTpush, ok := heartMsgOld.(*fsmohp.OrderPipelineRTpush)
		if ok {
			if differenceTime > int64(ohpRTpush.Interval) {
				isPass = 0
				noPassReason = "超时"
			}
		}
	}

	c.writeToMySQL(msgType, heartMsgOld, protocMsg, differenceTime, isPass, noPassReason)
}

// 从json文件中获取port
func portFromJson (protocMap map[string]interface{}, moduleName string) map[int]int {
	port := make(map[int]int)
	portList := protocMap[moduleName].([]interface{})
	for _, v := range portList {
		val := int(v.(float64))
		port[val] = val
	}
	return port
}

func testclient () {

}
