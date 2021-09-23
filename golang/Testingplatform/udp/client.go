package udp

import (
	"TestingPlatform/Utils"
	"TestingPlatform/pool"
	"TestingPlatform/protoc/fsmvci"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"net"
	"sync"
	"time"
)

type Client struct {
	Mu sync.Mutex
	Interval uint32				//数据间隔时长
	HeartbeatCtr int64			//心跳计数
	CurrentTime uint64			//当前时间戳
	MessageChan chan []byte		//消息管道

	Address net.UDPAddr			//UDPAddr
	WSocket *Wsocket			//WS对象
	Conn *net.UDPConn			//UDP链接
	Timer *time.Timer			//定时计数器
}

type Wsocket struct {
	Wsmu sync.Mutex
	WsRoom map[int][]string		//WebSocket 分组
	WsConnList map[string]*websocket.Conn	//WebSocket链接
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
	return &Client{
		Address: pool.GetAddress(),
		Conn : udpConn,
		Interval:1000,
		HeartbeatCtr: 0,
		MessageChan: make(chan []byte, 1000),
		WSocket: Wsocket,
		CurrentTime: uint64(time.Now().UnixMilli()),
		Timer: time.NewTimer(1000*time.Millisecond),
	}, nil
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
		protoMsg, err := Utils.DeCodeByProto(bytes)
		if err != nil {
			return
		}
		msgTypeCode := Utils.ByteToUint8(bytes[3:4])
		switch msgTypeCode {
		case 0x80:
			//vCILoginAns := protoMsg.(*fsmvci.VehicleChargingInterfaceLoginAns)
			fmt.Printf("this is server return message 0x80 %+v \n", protoMsg)
			c.sendWSChannel(protoMsg)
		case 0x82:
			vCIHeartAns := protoMsg.(*fsmvci.VCImainHeartbeatAns)
			if c.HeartbeatCtr + 3  < int64(vCIHeartAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(vCIHeartAns.HeartbeatCtr.Value) {
				fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
				return
			}
			c.Interval = vCIHeartAns.Interval.Value
			fmt.Printf("this is server return message 0x82 %v \n", vCIHeartAns)
			c.sendWSChannel(protoMsg)
		case 0x84:
			pluggedHeartbeatAns := protoMsg.(*fsmvci.VCIPluggedHeartbeatAns)
			if c.HeartbeatCtr + 3  < int64(pluggedHeartbeatAns.HeartbeatCtr.Value)  || c.HeartbeatCtr -3 > int64(pluggedHeartbeatAns.HeartbeatCtr.Value) {
				fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
				return
			}
			c.Interval = pluggedHeartbeatAns.Interval.Value
			fmt.Printf("this is server return message 0x84 %v \n", pluggedHeartbeatAns)
			c.sendWSChannel(protoMsg)
		case 0x86:
			vCIChargingHeartbeatAns := protoMsg.(*fsmvci.VCIChargingHeartbeatAns)
			c.Interval = vCIChargingHeartbeatAns.Interval.Value
			fmt.Printf("this is server return message 0x86 %v \n", vCIChargingHeartbeatAns)
			c.sendWSChannel(protoMsg)
		case 0x88:
			vCIChargingRTpull := protoMsg.(*fsmvci.VCIChargingRTpull)
			fmt.Printf("this is server return message 0x88 %v \n", vCIChargingRTpull)
			c.sendWSChannel(protoMsg)
		default:

		}
	}
}

// WriteMsg
// 客户端向服务端接收数据
func (c *Client) WriteMsg (msgType uint, protoMsg proto.Message) {
	bytes, err := Utils.EnCodeByProto(msgType, protoMsg)
	if err != nil {
		return
	}
	_, err = c.Conn.Write(bytes)
	if err != nil {
		return
	}
	//fmt.Printf("%0x \n", bytes)
}

func (c *Client) writeToMySQL (msg proto.Message) {

}

// 返回的数据发送给WS管道，返回给前端
func (c *Client) sendWSChannel (msg proto.Message) {
	messageChan, _ := json.Marshal(msg)
	c.MessageChan<-messageChan
}

func testclient () {

}
