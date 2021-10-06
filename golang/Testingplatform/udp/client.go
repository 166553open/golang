package udp

import (
	"TestingPlatform/Utils"
	"TestingPlatform/database"
	"TestingPlatform/form"
	"TestingPlatform/pool"
	"TestingPlatform/protoc/fsmohp"
	"TestingPlatform/protoc/fsmpmm"
	"TestingPlatform/protoc/fsmvci"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client struct {
	Mu sync.Mutex
	ConnRemotePort int			//远端（服务端）端口号
	ConnType string				//链接类型 VCI PMM OHP
	Interval uint32				//数据间隔时长
	HeartbeatCtr int64			//心跳计数
	RTpushCtr	uint32			//推送计数
	CurrentTime uint64			//当前时间戳

	WsMessageChan chan []byte			//websocket 消息管道

	RegisterChan chan []byte	//注册 消息管道
	HeartBeatChan chan []byte   //心跳 消息管道
	RealTimeChan chan []byte	//realtime 消息管道

	RegisterMessageChan		map[uint]*RegisterMessageChan  // 注册 消息管道 消息暂存地址
	HeartBeatMessageChan	map[uint]*HeartBeatMessageChan // 心跳 消息管道 消息暂存地址
	RealTimeMessageChan 	map[uint]*RealTimeMessageChan  // realtime 消息管道 消息暂存地址

	Address net.UDPAddr			//UDPAddr
	WSocket *Wsocket			//WS对象
	Conn *net.UDPConn			//UDP链接
	Timer *time.Timer			//定时计数器
}

// RegisterMessageChan 注册消息 暂存地址
type RegisterMessageChan struct {
	SendRegisterChan proto.Message		// 发送的数据内容
	SendRegisterTime int64				// 时间
	//RecvRegisterChan []byte			// 接收的数据内容
}

// HeartBeatMessageChan 注册消息 暂存地址
type HeartBeatMessageChan struct {
	SendHeartBeatChan proto.Message		// 发送的数据内容
	SendHeartBeatTime int64				// 时间
	//RecvHeartBeatChan []byte			// 接收的数据内容
}

// RealTimeMessageChan 注册消息 暂存地址
type RealTimeMessageChan struct {
	SendRealTimeChan proto.Message		// 发送的数据内容
	SendRealTimeTime int64				// 时间
	//RecvHeartBeatChan []byte			// 接收的数据内容
}

// Wsocket web socket 数据接收
type Wsocket struct {
	Wsmu sync.Mutex
	WsRoom map[int][]string		//WebSocket 分组
	WsConnList map[string]*websocket.Conn	//WebSocket链接
}

// NewRegisterMessageRam 注册消息 暂存地址 初始化
func NewRegisterMessageRam () *RegisterMessageChan {
	var protoMsg proto.Message
	return &RegisterMessageChan{
		SendRegisterChan: protoMsg,
		//RecvRegisterChan: make([]byte, 1024),
	}
}

// NewHeartBeatMessageRam 心跳消息 暂存地址 初始化
func NewHeartBeatMessageRam () *HeartBeatMessageChan {
	var protoMsg proto.Message
	return &HeartBeatMessageChan{
		SendHeartBeatChan: protoMsg,
		//RecvHeartBeatChan: make([]byte, 1024),
	}
}

// NewRealTimeMessageRam realtime消息 暂存地址 初始化
func NewRealTimeMessageRam () *RealTimeMessageChan {
	var protoMsg proto.Message
	return &RealTimeMessageChan{
		SendRealTimeChan:  protoMsg,
		//RecvHeartBeatChan: make([]byte, 1024),
	}
}

// NewClient return *Client
// 返回*Client
func NewClient (pool *pool.UDPpool) (*Client, error) {
	Wsocket := &Wsocket{
		WsRoom: make(map[int][]string, 0),
		WsConnList: make(map[string]*websocket.Conn),
	}
	udpConn, err := pool.Open()
	if err != nil {
		return nil, err
	}
	// 后续代码需要用端口号区分
	serverIpAndPorts := udpConn.RemoteAddr().String()
	serverIpAndPort := strings.Split(serverIpAndPorts, ":")
	serverPort := serverIpAndPort[len(serverIpAndPort)-1]
	serverPortInt, _ := strconv.Atoi(serverPort)

	connType := ""
	if serverPortInt == 9000 || serverPortInt == 9010 || serverPortInt == 9020 {
		connType = "VCI"
	}
	if serverPortInt == 9030 || serverPortInt == 9040 {
		connType = "PMM"
	}
	if serverPortInt == 9050 {
		connType = "OHP"
	}

	client := &Client{
		ConnRemotePort: serverPortInt,
		ConnType: connType,
		Address: pool.GetAddress(),
		Conn : udpConn,
		Interval:1000,
		HeartbeatCtr: 0,
		RTpushCtr: 0,
		RegisterMessageChan: make(map[uint]*RegisterMessageChan),
		HeartBeatMessageChan: make(map[uint]*HeartBeatMessageChan),
		RealTimeMessageChan: make(map[uint]*RealTimeMessageChan),
		RegisterChan: make(chan []byte),
		HeartBeatChan: make(chan []byte),
		RealTimeChan: make(chan []byte),
		WSocket: Wsocket,
		CurrentTime: uint64(time.Now().UnixMilli()),
		Timer: time.NewTimer(1000*time.Millisecond),
	}


	if serverPortInt == 9000 || serverPortInt == 9010 || serverPortInt == 9020 {

		// 注册信息帧
		registerMessageChan := NewRegisterMessageRam()

		client.RegisterMessageChan[0x00] = registerMessageChan
		//client.RegisterMessageChan[0x80] = registerMessageChan

		// 心跳信息帧 // 不同线程中的 只存在一个初始化内存地址
		heartBeatMessageChan := NewHeartBeatMessageRam()

		client.HeartBeatMessageChan[0x02] = heartBeatMessageChan
		//client.HeartBeatMessageChan[0x82] = heartBeatMessageChan

		client.HeartBeatMessageChan[0x04] = heartBeatMessageChan
		//client.HeartBeatMessageChan[0x84] = heartBeatMessageChan

		client.HeartBeatMessageChan[0x06] = heartBeatMessageChan
		//client.HeartBeatMessageChan[0x86] = heartBeatMessageChan

		// realtime 信息帧
		realTimeMessageChan := NewRealTimeMessageRam()
		client.RealTimeMessageChan[0x08] = realTimeMessageChan
		//client.RealTimeMessageChan[0x88] = realTimeMessageChan
	}

	if serverPortInt == 9030 || serverPortInt == 9040 {

		// 注册信息帧 //同一个线程定义了两个注册，要成指向不同的内存地址
		registerMessageChan := NewRegisterMessageRam()

		client.RegisterMessageChan[0x00] = registerMessageChan
		//client.RegisterMessageChan[0x80] = registerMessageChan

		registerMessageChan1 := NewRegisterMessageRam()
		client.RegisterMessageChan[0x02] = registerMessageChan1
		//client.RegisterMessageChan[0x82] = registerMessageChan1

		// 心跳信息帧 // 不同线程中的 只存在一个初始化内存地址
		heartBeatMessageChan := NewHeartBeatMessageRam()
		client.HeartBeatMessageChan[0x04] = heartBeatMessageChan
		//client.HeartBeatMessageChan[0x84] = heartBeatMessageChan

		client.HeartBeatMessageChan[0x06] = heartBeatMessageChan
		//client.HeartBeatMessageChan[0x86] = heartBeatMessageChan

		// realtime 信息帧
		realTimeMessageChan := NewRealTimeMessageRam()
		client.RealTimeMessageChan[0x08] = realTimeMessageChan
		//client.RealTimeMessageChan[0x88] = realTimeMessageChan
	}

	if serverPortInt == 9050 {

		// 注册信息帧 //同一个线程定义了两个注册，要成指向不同的内存地址
		registerMessageChan := NewRegisterMessageRam()

		client.RegisterMessageChan[0x00] = registerMessageChan
		//client.RegisterMessageChan[0x80] = registerMessageChan

		// 心跳信息帧 // 不同线程中的 只存在一个初始化内存地址
		heartBeatMessageChan := NewHeartBeatMessageRam()
		client.HeartBeatMessageChan[0x04] = heartBeatMessageChan
		//client.HeartBeatMessageChan[0x84] = heartBeatMessageChan

		// realtime 信息帧
		realTimeMessageChan := NewRealTimeMessageRam()
		client.RealTimeMessageChan[0x08] = realTimeMessageChan
		//client.RealTimeMessageChan[0x88] = realTimeMessageChan
	}

	return client, nil

}

// ReadMsg  Client
// 客户端从服务端接收数据
func (c *Client) ReadMsg () {
	for {
		bytes := make([]byte, 1024)
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
		//vCILoginAns := protoMsg.(*fsmvci.VehicleChargingInterfaceLoginAns)
		fmt.Printf("this is server return message 0x80 %+v \n", protoMsg)
		//c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckRegisterMessageRam(0x00, protoMsg)
	case 0x82:
		vCIHeartAns := protoMsg.(*fsmvci.VCImainHeartbeatAns)
		if c.HeartbeatCtr + 3  < int64(vCIHeartAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(vCIHeartAns.HeartbeatCtr.Value) {
			fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
			return
		}
		c.HeartbeatCtr = int64(vCIHeartAns.HeartbeatCtr.GetValue())
		c.Interval = vCIHeartAns.Interval.GetValue()
		fmt.Printf("this is server return message 0x82 %v \n", vCIHeartAns)
		//c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckHeartMessageRam(0x02, protoMsg)
	case 0x84:
		pluggedHeartbeatAns := protoMsg.(*fsmvci.VCIPluggedHeartbeatAns)
		if c.HeartbeatCtr + 3  < int64(pluggedHeartbeatAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(pluggedHeartbeatAns.HeartbeatCtr.Value) {
			fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
			return
		}
		c.HeartbeatCtr = int64(pluggedHeartbeatAns.HeartbeatCtr.GetValue())
		c.Interval = pluggedHeartbeatAns.Interval.GetValue()
		fmt.Printf("this is server return message 0x84 %v \n", pluggedHeartbeatAns)
		//c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckHeartMessageRam(0x04, protoMsg)
	case 0x86:
		vCIChargingHeartbeatAns := protoMsg.(*fsmvci.VCIChargingHeartbeatAns)
		c.HeartbeatCtr = int64(vCIChargingHeartbeatAns.HeartbeatCtr.GetValue())
		c.Interval = vCIChargingHeartbeatAns.Interval.GetValue()
		fmt.Printf("this is server return message 0x86 %v \n", vCIChargingHeartbeatAns)
		//c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckHeartMessageRam(0x06, protoMsg)
	case 0x88:
		vCIChargingRTpull := protoMsg.(*fsmvci.VCIChargingRTpull)
		fmt.Printf("this is server return message 0x88 %v \n", vCIChargingRTpull)
		//c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckRealTimeMessageRam(0x08, protoMsg)
	}
}

func (c *Client) readMsgFromPMM (msgTypeCode uint8, protoMsg proto.Message) {
	switch msgTypeCode {

	case 0x80:
		//vCILoginAns := protoMsg.(*fsmvci.VehicleChargingInterfaceLoginAns)
		fmt.Printf("this is server return message 0x80 %+v \n", protoMsg)
		go c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckRegisterMessageRam(0x00, protoMsg)
	case 0x82:
		ADModuleLoginAns := protoMsg.(*fsmpmm.ADModuleLoginAns)
		//if c.HeartbeatCtr + 3  < int64(vCIHeartAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(vCIHeartAns.HeartbeatCtr.Value) {
		//	fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
		//	return
		//}
		//c.Interval = vCIHeartAns.Interval.Value
		fmt.Printf("this is server return message 0x82 %v \n", ADModuleLoginAns)
		go c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckRegisterMessageRam(0x02, protoMsg)
	case 0x84:
		PMMHeartbeatAns := protoMsg.(*fsmpmm.PMMHeartbeatAns)
		c.HeartbeatCtr = int64(PMMHeartbeatAns.HeartbeatCtr.GetValue())
		//if c.HeartbeatCtr + 3  < int64(pluggedHeartbeatAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(pluggedHeartbeatAns.HeartbeatCtr.Value) {
		//	fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
		//	return
		//}
		//c.Interval = pluggedHeartbeatAns.Interval.Value
		fmt.Printf("this is server return message 0x84 %v \n", protoMsg)
		go c.sendWSChannel(c.WsMessageChan, protoMsg)

		c.CheckHeartMessageRam(0x04, protoMsg)
	case 0x86:
		//vCIChargingHeartbeatAns := protoMsg.(*fsmvci.VCIChargingHeartbeatAns)
		//c.Interval = vCIChargingHeartbeatAns.Interval.Value
		fmt.Printf("this is server return message 0x86 %v \n", protoMsg)
		go c.sendWSChannel(c.WsMessageChan, protoMsg)

		c.CheckHeartMessageRam(0x06, protoMsg)
	case 0x88:
		//vCIChargingRTpull := protoMsg.(*fsmvci.VCIChargingRTpull)
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
		if c.HeartbeatCtr + 3  < int64(orderHeart.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(orderHeart.HeartbeatCtr.Value) {
			fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
			return
		}
		c.HeartbeatCtr = int64(orderHeart.HeartbeatCtr.GetValue())
		c.Interval = orderHeart.Interval.GetValue()
		fmt.Printf("this is server return message 0x82 %v \n", orderHeart)
		//c.sendWSChannel(c.WsMessageChan, protoMsg)
		c.CheckHeartMessageRam(0x02, protoMsg)
	case 0x84:
		pluggedHeartbeatAns := protoMsg.(*fsmvci.VCIPluggedHeartbeatAns)
		if c.HeartbeatCtr + 3  < int64(pluggedHeartbeatAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(pluggedHeartbeatAns.HeartbeatCtr.Value) {
			fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
			return
		}
		c.HeartbeatCtr = int64(pluggedHeartbeatAns.HeartbeatCtr.GetValue())
		c.Interval = pluggedHeartbeatAns.Interval.GetValue()
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

	//err = c.writeToMySQL(msgType, protoMsg)
	//if err != nil {
	//	fmt.Printf("mysql insert data error: %s", err.Error())
	//	return
	//}
	//fmt.Printf("%0x \n", bytes)
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
func (c *Client) writeToMySQL (msgType uint, protocMsgOld, protocMsgNew proto.Message, costTime int64, isPass int) error {
	testCollection := form.TestCollection{}
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

	protocStrOld, err := Utils.PB2Json(protocMsgOld)
	protocStrNew, err := Utils.PB2Json(protocMsgNew)

	if err != nil {
		return err
	}
	testCollection.InputMessage = protocStrOld
	testCollection.OutputMessage = protocStrNew
	_, err = database.MySQL.Table("test_collection").Insert(&testCollection)
	if err != nil{
		return err
	}
	return nil
}

// sendDataRam
// 数据添加到内存
func (c *Client) sendDataRam (msgType uint, protocMsg proto.Message) {
	if c.ConnType == "VCI" {
		if msgType == 0x00 {
			c.RegisterMessageChan[msgType] = NewRegisterMessageRam()
			c.RegisterMessageChan[msgType].SendRegisterChan = protocMsg
			c.RegisterMessageChan[msgType].SendRegisterTime = time.Now().UnixMilli()
		}
		if msgType == 0x02 || msgType == 0x04 || msgType == 0x06 {
			c.HeartBeatMessageChan[msgType] = NewHeartBeatMessageRam()
			c.HeartBeatMessageChan[msgType].SendHeartBeatChan = protocMsg
			c.HeartBeatMessageChan[msgType].SendHeartBeatTime = time.Now().UnixMilli()
		}
		if msgType == 0x08 {
			c.RealTimeMessageChan[msgType] = NewRealTimeMessageRam()
			c.RealTimeMessageChan[msgType].SendRealTimeChan = protocMsg
			c.RealTimeMessageChan[msgType].SendRealTimeTime = time.Now().UnixMilli()
		}
	}
	if c.ConnType == "PMM" {
		if msgType == 0x00 {
			c.RegisterMessageChan[msgType] = NewRegisterMessageRam()
			c.RegisterMessageChan[msgType].SendRegisterChan = protocMsg
			c.RegisterMessageChan[msgType].SendRegisterTime = time.Now().UnixMilli()
		}
		if msgType == 0x02 {
			c.RegisterMessageChan[msgType] = NewRegisterMessageRam()
			c.RegisterMessageChan[msgType].SendRegisterChan = protocMsg
			c.RegisterMessageChan[msgType].SendRegisterTime = time.Now().UnixMilli()
		}
		if msgType == 0x04 || msgType == 0x06 {
			c.HeartBeatMessageChan[msgType] = NewHeartBeatMessageRam()
			c.HeartBeatMessageChan[msgType].SendHeartBeatChan = protocMsg
			c.HeartBeatMessageChan[msgType].SendHeartBeatTime = time.Now().UnixMilli()
		}

		if msgType == 0x08 {
			c.RealTimeMessageChan[msgType] = NewRealTimeMessageRam()
			c.RealTimeMessageChan[msgType].SendRealTimeChan = protocMsg
			c.RealTimeMessageChan[msgType].SendRealTimeTime = time.Now().UnixMilli()
		}
	}
}

// CheckRegisterMessageRam
// 监测注册消息管道数据存入数据库
func (c *Client) CheckRegisterMessageRam (msgType uint,  protocMsg proto.Message) {
	heartMsgOld := c.RegisterMessageChan[msgType].SendRegisterChan
	oldTime := c.RegisterMessageChan[msgType].SendRegisterTime
	differenceTime := time.Now().UnixMilli() -  oldTime
	err := c.writeToMySQL(msgType, heartMsgOld, protocMsg, differenceTime, 1)
	if err != nil {
		fmt.Printf("Mysql error : %s", err.Error())
	}
}

// CheckHeartMessageRam
// 监测心跳消息管道是否合法数据
// 不合法存入数据库
func (c *Client) CheckHeartMessageRam (msgType uint,  protocMsg proto.Message) {
	heartMsg := c.HeartBeatMessageChan[msgType].SendHeartBeatChan
	if c.ConnType == "VCI" {
		if msgType == 0x02 {
			heartMsgOld := heartMsg.(*fsmvci.VCImainHeartbeatReq)
			heartMsgNew := protocMsg.(*fsmvci.VCImainHeartbeatAns)
			//
			// heartMsgOld.CurrentTime.Time + uint64(heartMsgOld.Interval.GetValue()) < heartMsgNew.CurrentTime.Time
			// heartMsgOld.CurrentTime.Time + uint64(heartMsgOld.Interval.GetValue()) < heartMsgNew.CurrentTime.Time
			if heartMsgOld.HeartbeatCtr.GetValue() + 3 < heartMsgNew.HeartbeatCtr.GetValue() || heartMsgOld.HeartbeatCtr.GetValue() - 3 > heartMsgNew.HeartbeatCtr.GetValue() ||
				heartMsgOld.CurrentTime.Time + uint64(heartMsgOld.Interval.GetValue()) < heartMsgNew.CurrentTime.Time {
				costTime := heartMsgNew.CurrentTime.Time - heartMsgOld.CurrentTime.Time

				// TODO 存入数据库
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), 0)
			}
		}else if msgType == 0x04 {
			heartMsgOld := heartMsg.(*fsmvci.VCIPluggedHeartbeatReq)
			heartMsgNew := protocMsg.(*fsmvci.VCIPluggedHeartbeatAns)
			if heartMsgOld.HeartbeatCtr.GetValue() + 3 < heartMsgNew.HeartbeatCtr.GetValue() || heartMsgOld.HeartbeatCtr.GetValue() - 3 > heartMsgNew.HeartbeatCtr.GetValue() ||
				heartMsgOld.CurrentTime.Time + uint64(heartMsgOld.Interval.GetValue()) < heartMsgNew.CurrentTime.Time {
				costTime := heartMsgNew.CurrentTime.Time - heartMsgOld.CurrentTime.Time
				// TODO 存入数据库
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), 0)
			}
		}else{
			heartMsgOld := heartMsg.(*fsmvci.VCIChargingHeartbeatReq)
			heartMsgNew := protocMsg.(*fsmvci.VCIChargingHeartbeatAns)
			if heartMsgOld.HeartbeatCtr.GetValue() + 3 < heartMsgNew.HeartbeatCtr.GetValue() || heartMsgOld.HeartbeatCtr.GetValue() - 3 > heartMsgNew.HeartbeatCtr.GetValue() ||
				heartMsgOld.CurrentTime.Time + uint64(heartMsgOld.Interval.GetValue()) < heartMsgNew.CurrentTime.Time {
				costTime := heartMsgNew.CurrentTime.Time - heartMsgOld.CurrentTime.Time
				// TODO 存入数据库
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), 0)
			}
		}
	}
	if c.ConnType == "PMM" {
		if msgType == 0x04 {
			heartMsgOld := heartMsg.(*fsmpmm.PMMHeartbeatReq)
			heartMsgNew := protocMsg.(*fsmpmm.PMMHeartbeatAns)
			if heartMsgOld.HeartbeatCtr.GetValue() + 3 < heartMsgNew.HeartbeatCtr.GetValue() || heartMsgOld.HeartbeatCtr.GetValue() - 3 > heartMsgNew.HeartbeatCtr.GetValue() ||
				heartMsgOld.CurrentTime.Time + uint64(heartMsgOld.Interval.GetValue()) < heartMsgNew.CurrentTime.Time {
				costTime := heartMsgNew.CurrentTime.Time - heartMsgOld.CurrentTime.Time

				// TODO 存入数据库
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), 0)
			}
		}else{
			heartMsgOld := heartMsg.(*fsmpmm.MainContactorHeartbeatReq)
			heartMsgNew := protocMsg.(*fsmpmm.MainContactorHeartbeatAns)
			if heartMsgOld.HeartbeatCtr.GetValue() + 3 < heartMsgNew.HeartbeatCtr.GetValue() || heartMsgOld.HeartbeatCtr.GetValue() - 3 > heartMsgNew.HeartbeatCtr.GetValue() ||
				heartMsgOld.CurrentTime.Time + uint64(heartMsgOld.Interval.GetValue()) < heartMsgNew.CurrentTime.Time {
				costTime := heartMsgNew.CurrentTime.Time - heartMsgOld.CurrentTime.Time
				// TODO 存入数据库
				c.writeToMySQL(msgType, heartMsgOld, heartMsgNew, int64(costTime), 0)
			}
		}
	}
}

// CheckRealTimeMessageRam
// 监测realtime消息管道数据存入数据库
func (c *Client) CheckRealTimeMessageRam (msgType uint, protocMsg proto.Message) {
	isPass := 1
	heartMsgOld := c.RealTimeMessageChan[msgType].SendRealTimeChan
	oldTime := c.RealTimeMessageChan[msgType].SendRealTimeTime
	differenceTime := time.Now().UnixMilli() -  oldTime
	if c.ConnType == "VCI" {
		rTpush := heartMsgOld.(*fsmvci.VCIChargingRTpush)
		//rTpull := protocMsg.(*fsmvci.VCIChargingRTpull)
		if differenceTime > int64(rTpush.Interval.GetValue()) {
			isPass = 0
		}
	}
	if c.ConnType == "PMM" {
		rTpush := heartMsgOld.(*fsmpmm.MainContactorRTpush)
		//rTpull := protocMsg.(*fsmvci.VCIChargingRTpull)
		if differenceTime > int64(rTpush.Interval.GetValue()) {
			isPass = 0
		}
	}

	c.writeToMySQL(msgType, heartMsgOld, protocMsg, differenceTime, isPass)
}

func testclient () {

}
