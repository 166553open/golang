package udp

import (
	bytes2 "bytes"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"goWork/Utils"
	"goWork/protoc/fsmvci"
	"net"
	"sync"
	"time"
)

type Client struct {
	Mu sync.Mutex

	GunInterval uint32
	GunHeartbeatCtr int64
	GunCurrentTime uint64
	Address net.UDPAddr
	GunStateList map[string]string
	Conn *net.UDPConn
	Timer *time.Timer
}

//127.0.0.1:9000
func NewClient (address net.UDPAddr) (*Client, error) {
	addr, err := net.ResolveUDPAddr("udp", address.String())
	if err != nil {
		fmt.Println("client udp addr is error:", err.Error())
		return nil, err
	}
	udpConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("client udp connection is error:", err.Error())
		return nil, err
	}
	return &Client{
		Address: address,
		Conn : udpConn,
		GunInterval:1000,
		GunHeartbeatCtr: 0,
		GunCurrentTime: uint64(time.Now().Unix()),
		Timer: time.NewTimer(1000*time.Millisecond),
	}, nil
}

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

		var msgTypeCode uint8
		binary.Read(bytes2.NewBuffer(bytes[3:4]), binary.BigEndian, &msgTypeCode)

		switch msgTypeCode {
		case 0x80:
			vCILoginAns := protoMsg.(*fsmvci.VehicleChargingInterfaceLoginAns)
			fmt.Printf("this is server return message 0x80 %v \n", vCILoginAns)
		case 0x82:
			vCILoginAns := protoMsg.(*fsmvci.VCImainHeartbeatAns)
			if c.GunHeartbeatCtr + 3  < int64(vCILoginAns.HeartbeatCtr.Value)  || c.GunHeartbeatCtr -3 > int64(vCILoginAns.HeartbeatCtr.Value) {
				fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
				return
			}
			c.Mu.Lock()
			c.GunHeartbeatCtr++
			c.Mu.Unlock()
			fmt.Printf("this is server return message 0x82 %v \n", vCILoginAns)
		case 0x84:
			pluggedHeartbeatAns := protoMsg.(*fsmvci.VCIPluggedHeartbeatAns)
			if c.GunHeartbeatCtr + 3  < int64(pluggedHeartbeatAns.HeartbeatCtr.Value)  || c.GunHeartbeatCtr -3 > int64(pluggedHeartbeatAns.HeartbeatCtr.Value) {
				fmt.Printf("HeartbeatCtr error :client heartbeat less server heartbeat")
				return
			}
			c.Mu.Lock()
			c.GunHeartbeatCtr++
			c.Mu.Unlock()
			fmt.Printf("this is server return message 0x84 %v \n", pluggedHeartbeatAns)
		case 0x86:
			c.Mu.Lock()
			c.GunHeartbeatCtr++
			c.Mu.Unlock()
			vCIChargingHeartbeatAns := protoMsg.(*fsmvci.VCIChargingHeartbeatAns)
			fmt.Printf("this is server return message 0x86 %v \n", vCIChargingHeartbeatAns)
		case 0x88:
			vCIChargingRTpull := protoMsg.(*fsmvci.VCIChargingRTpull)
			fmt.Printf("this is server return message 0x86 %v \n", vCIChargingRTpull)
		default:

		}
	}
}



func (c *Client) WriteMsg (msgType uint, protoMsg proto.Message) {
	bytes, err := Utils.EnCodeByProto(msgType, protoMsg)
	if err != nil {
		return
	}
	_, err = c.Conn.Write(bytes)
	if err != nil {
		return
	}
}

//func (c *client) WriteHeartBeat () {
//	vCIHeartbeatReq := &fsmvci.VCImainHeartbeatReq{
//		HeartbeatCtr: &fsmvci.Uint32Value{Value: c.HeartBeatNum},
//		GunBaseList:  gunBaseList,
//		CurrentTime:  &fsmvci.DateTimeLong{Time: uint64(time.Now().Unix()) },
//		Interval:     &fsmvci.Uint32Value{Value: 200},
//	}
//}

func testclient () {

}
