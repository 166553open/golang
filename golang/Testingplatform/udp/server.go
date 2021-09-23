package udp

import (
	"TestingPlatform/Utils"
	"TestingPlatform/protoc/fsmvci"
	bytes2 "bytes"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"net"
	"sync"
	"time"
)

type Server struct {
	MaxTime int
	Ip net.IP
	Prot int
	OnlineList map[string]*ConnWithUDP
	muOnlineList sync.Mutex
}

type ConnWithUDP struct {
	Lived bool
	TimeOutNum int8
	LastTime int64
	Addr string
}

func NewServer (ip net.IP, port int) *Server {
	return &Server{
		MaxTime:  1000,
		Ip:       ip,
		Prot:     port,
		OnlineList: make(map[string]*ConnWithUDP),
	}
}

func (s *Server) ConnectConn (addr string) {
	s.muOnlineList.Lock()
	s.OnlineList[addr] = &ConnWithUDP{
		Lived:      true,
		TimeOutNum: 0,
		LastTime:   time.Now().UnixMilli(),
		Addr:       addr,
	}
	s.muOnlineList.Unlock()
}

func (s *Server) Online (addr string) {
	s.ConnectConn(addr)
}

func (s *Server) HeartBeat (addr string) {
	if addr == "" {
		return
	}
	s.muOnlineList.Lock()
	conn, ok:= s.OnlineList[addr]
	if ok {
		conn.LastTime = time.Now().UnixMilli()
		conn.TimeOutNum = 0
	}
	s.muOnlineList.Unlock()
}

func (s *Server) UnConnectConn(addr string) {
	s.muOnlineList.Lock()
	delete(s.OnlineList, addr)
	s.muOnlineList.Unlock()
}

func (s *Server) ReadMsg (conn *net.UDPConn) {
	for {
		bytes := make([]byte, 1024)
		n, addr, err := conn.ReadFrom(bytes)
		if err != nil {
			fmt.Println("ReadFrom is error :", err.Error())
			break
		}
		if n == 0 {
			s.UnConnectConn(addr.String())
			fmt.Println("client close :", conn.RemoteAddr().String())
			break
		}
		protoMsg, err := Utils.DeCodeByProto(bytes)
		if err != nil {
			return
		}
		fmt.Printf("read msg from %s and this context %v \n", addr.String(), protoMsg)
		s.Online(addr.String())
		msgTypeCode := Utils.ByteToUint8(bytes[3:4])
		binary.Read(bytes2.NewBuffer(bytes[3:4]), binary.BigEndian, &msgTypeCode)
		switch msgTypeCode {
		case 0x00:
			s.WriteVehicleChargingInterfaceLoginAns(conn, addr, protoMsg.(*fsmvci.VehicleChargingInterfaceLogin))
		case 0x02:
			s.WriteVCImainHeartbeatAns(conn, addr, protoMsg.(*fsmvci.VCImainHeartbeatReq))
		case 0x04:
			s.WriteVCIPluggedHeartbeatAns(conn, addr, protoMsg.(*fsmvci.VCIPluggedHeartbeatReq))
		case 0x06:
			s.WriteVCIChargingHeartbeatAns(conn, addr, protoMsg.(*fsmvci.VCIChargingHeartbeatReq))
		case 0x08:
			s.WriteVCIChargingRTpull(conn, addr, protoMsg.(*fsmvci.VCIChargingRTpush))
		}
	}
}

//Handle connect
//func (s *Server) WriteConnect (conn *net.UDPConn, msg string, addr net.Addr) {
//	s.ConnectConn(addr.String())
//	fmt.Printf("%s is connect \n", addr.String())
//}

// WriteVehicleChargingInterfaceLoginAns 0x80
func (s *Server) WriteVehicleChargingInterfaceLoginAns (conn *net.UDPConn, addr net.Addr, login *fsmvci.VehicleChargingInterfaceLogin) {
	vCIPramList := make([]*fsmvci.VCIPram, 0)
	pluggedList := make([]*fsmvci.PluggedRecover, 0)
	chargingList := make([]*fsmvci.ChargingRecover, 0)

	vCIPram := Utils.VCIPramInit(1, 1, true, false)
	vCIPramList = append(vCIPramList, vCIPram)

	sysParameterList := Utils.SysParameterInit()

	PluggedRecover := Utils.PluggedRecoverInit(1, true, false)
	pluggedList = append(pluggedList, PluggedRecover)

	chargingRecover := Utils.ChargingRecoverInit(1)
	chargingList = append(chargingList, chargingRecover)
	vCILoginAns := &fsmvci.VehicleChargingInterfaceLoginAns{
		VehicleChargingProtoVersion: Utils.ConfProtoc["VehicleChargingProtoVersion"].(string),
		MainStateMachineVendor:      Utils.ConfProtoc["MainStateMachineVendor"].(string),
		SelfCheckRul:                1,
		EnableServerList:            &fsmvci.EnableServer{
			VCIServer: &fsmvci.BoolEnum{Value: true},
			PMMServer: &fsmvci.BoolEnum{Value: false},
			DMCServer: &fsmvci.BoolEnum{Value: false},
			OHPServer: &fsmvci.BoolEnum{Value: false},
			LCRServer: &fsmvci.BoolEnum{Value: false},
		},
		VCIPramList:                 vCIPramList,
		SysParameterList:            sysParameterList,
		PluggedList:                 pluggedList,
		ChargingList:                chargingList,
	}
	s.WriteMsg(conn, 0x80, vCILoginAns, addr)
}

// WriteVCImainHeartbeatAns 0x82
func (s *Server) WriteVCImainHeartbeatAns (conn *net.UDPConn, addr net.Addr, vCImainHeartbeatReq *fsmvci.VCImainHeartbeatReq) {
	heartbeatCtr := vCImainHeartbeatReq.GetHeartbeatCtr().GetValue()
	heartbeatCtr++
	vCImainHeartbeatAns := &fsmvci.VCImainHeartbeatAns{
		HeartbeatCtr:     &fsmvci.Uint32Value{Value: heartbeatCtr},
		VCIPramList:      nil,
		SysParameterList: Utils.SysParameterInit(),
		CurrentTime:      &fsmvci.DateTimeLong{Time: uint64(time.Now().UnixMilli())},
		Interval:         &fsmvci.Uint32Value{Value: 2000},
	}
	s.WriteMsg(conn, 0x82, vCImainHeartbeatAns, addr)
}

// WriteVCIPluggedHeartbeatAns 0x84
func (s *Server) WriteVCIPluggedHeartbeatAns (conn *net.UDPConn, addr net.Addr, vCImainHeartbeatReq *fsmvci.VCIPluggedHeartbeatReq)  {
	heartbeatCtr := vCImainHeartbeatReq.HeartbeatCtr.Value
	heartbeatCtr++
	vCIPluggedHeartbeatAns := &fsmvci.VCIPluggedHeartbeatAns{
		ID:           vCImainHeartbeatReq.ID,
		HeartbeatCtr: &fsmvci.Uint32Value{Value: heartbeatCtr},
		VCIPramList:  nil,
		SysCtrl:      &fsmvci.SysCtrlPluggedStage{
			ElockCmd:  &fsmvci.BoolEnum{Value: true},
			StartCmd:  &fsmvci.BoolEnum{Value: false},
			StartType: &fsmvci.BoolEnum{Value: false},
		},
		CurrentTime:  &fsmvci.DateTimeLong{Time: uint64(time.Now().UnixMilli())},
		Interval:     &fsmvci.Uint32Value{Value: 2000},
	}
	s.WriteMsg(conn,0x84, vCIPluggedHeartbeatAns, addr)
}

// WriteVCIChargingHeartbeatAns 0x86
func (s *Server) WriteVCIChargingHeartbeatAns (conn *net.UDPConn, addr net.Addr, vCImainHeartbeatReq *fsmvci.VCIChargingHeartbeatReq)  {
	heartbeatCtr := vCImainHeartbeatReq.HeartbeatCtr.Value
	heartbeatCtr++
	vCIChargingHeartbeatAns := &fsmvci.VCIChargingHeartbeatAns{
		ID:           vCImainHeartbeatReq.ID,
		HeartbeatCtr: &fsmvci.Uint32Value{Value: heartbeatCtr},
		GunCaredMsg:  &fsmvci.GunCared{
			AllocaOK:       &fsmvci.BoolEnum{Value: true},
			OutConnectorFb: &fsmvci.BoolEnum{Value: false},
			MeterVol:       &fsmvci.DoubleValue{Value: 5.00},
			MeterCurr:      &fsmvci.DoubleValue{Value: 5.00},
			BatVol:         &fsmvci.DoubleValue{Value: 5.00},
			ModVol:         &fsmvci.DoubleValue{Value: 5.00},
		},
		CurrentTime:  &fsmvci.DateTimeLong{Time: uint64(time.Now().UnixMilli())},
		Interval:     &fsmvci.Uint32Value{Value: 2000},
	}

	s.WriteMsg(conn,0x86, vCIChargingHeartbeatAns, addr)
}

// WriteVCIChargingRTpull 0x88
func (s *Server) WriteVCIChargingRTpull (conn *net.UDPConn, addr net.Addr, vCImainHeartbeatReq *fsmvci.VCIChargingRTpush)  {
	vCIChargingRTpull := &fsmvci.VCIChargingRTpull{
		ID:        vCImainHeartbeatReq.ID,
		RTpullCtr: &fsmvci.Uint32Value{Value: 5},
		SysCtrl:   &fsmvci.SysCtrlChargingStage{StopCmd: &fsmvci.BoolEnum{Value: false}},
		FaultList: nil,
		Interval:  &fsmvci.Uint32Value{Value: 2000},
	}
	s.WriteMsg(conn,0x88, vCIChargingRTpull, addr)
}

// WriteHeartBeat Return heartBeat message
func  (s *Server) WriteHeartBeat (conn *net.UDPConn,msgTypeCode uint, protoMsg proto.Message, addr net.Addr) {
	s.HeartBeat(addr.String())
	s.WriteMsg(conn, msgTypeCode, protoMsg, addr)
}

// HeartBeatCheck HeartBeat check
// 心跳检测
func (s *Server) HeartBeatCheck () {
	for{
		sleepTime := time.NewTimer(time.Duration(s.MaxTime)*time.Millisecond)
		select {
		case <- sleepTime.C:
			for _, v := range s.OnlineList {
				nowTime := time.Now().UnixMilli()
				if nowTime > int64(s.MaxTime) + v.LastTime {
					if v.TimeOutNum == 2 {
						fmt.Println("leave", v.Addr)
						s.UnConnectConn(v.Addr)
					}else {
						v.TimeOutNum ++
						v.LastTime = nowTime
						fmt.Printf("this time out %d \n", v.TimeOutNum)
					}
				}
			}
		}
	}
}

// WriteMsg
// 数据发送
func (s *Server) WriteMsg(conn *net.UDPConn, msgTypeCode uint, protoMsg proto.Message, addr net.Addr) {
	bytes, err := Utils.EnCodeByProto(msgTypeCode, protoMsg)
	if err != nil {
		return
	}
	_, err = conn.WriteTo(bytes, addr)
	if err != nil {
		fmt.Printf("server send to %v error : %s", conn.RemoteAddr(), err.Error())
		return
	}
}

// BuildUdpServer listen and build udp server
func (s *Server) BuildUdpServer() (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   s.Ip,
		Port: s.Prot,
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func testserver() {

	UdpServer := NewServer(net.IPv4(0,0,0,0), 9000)
	udpConn, err := UdpServer.BuildUdpServer()
	if err != nil {
		fmt.Printf("udp connect error : %s", err.Error())
		return
	}
	go UdpServer.HeartBeatCheck()
	go UdpServer.ReadMsg(udpConn)
	for{

	}
}
