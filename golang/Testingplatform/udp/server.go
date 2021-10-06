package udp

import (
	"TestingPlatform/Utils"
	"TestingPlatform/protoc/fsmohp"
	"TestingPlatform/protoc/fsmpmm"
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
		protoMsg, err := Utils.DeCodeByProto(s.Prot, bytes)
		if err != nil {
			return
		}
		fmt.Printf("read msg from %s and this context %v \n", addr.String(), protoMsg)
		s.Online(addr.String())
		msgTypeCode := Utils.ByteToUint8(bytes[3:4])
		binary.Read(bytes2.NewBuffer(bytes[3:4]), binary.BigEndian, &msgTypeCode)
		s.messageTypeAssertion(conn, addr, msgTypeCode, protoMsg)
	}
}

// messageTypeAssertion
// 消息类型断言
// 不通Server消息体的Command 都是80、82、84、86、88
func (s *Server) messageTypeAssertion (conn *net.UDPConn, addr net.Addr, msgTypeCode uint8, protoMsg proto.Message) {
	if s.Prot == 9000 || s.Prot == 9010 || s.Prot == 9020 {
		s.vciMessage(conn, addr, msgTypeCode, protoMsg)
	}
	if s.Prot == 9030 || s.Prot == 9040 {
		s.pmmMessage(conn, addr, msgTypeCode, protoMsg)
	}
	if s.Prot == 9050 {
		s.ohpMessage(conn, addr, msgTypeCode, protoMsg)
	}
}

// VCI Server
// vciMessage
func (s *Server) vciMessage (conn *net.UDPConn, addr net.Addr, msgTypeCode uint8, protoMsg proto.Message) {
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

// PMM Server
// pmmMessage
func (s *Server) pmmMessage (conn *net.UDPConn, addr net.Addr, msgTypeCode uint8, protoMsg proto.Message) {
	switch msgTypeCode {
	case 0x00:
		s.PowerMatrixLoginAns(conn, addr, protoMsg.(*fsmpmm.PowerMatrixLogin))
	case 0x02:
		s.ADModuleLoginAns(conn, addr, protoMsg.(*fsmpmm.ADModuleLogin))
	case 0x04:
		s.PMMHeartbeatReq(conn, addr, protoMsg.(*fsmpmm.PMMHeartbeatReq))
	case 0x06:
		s.MainContactorHeartbeatAns(conn, addr, protoMsg.(*fsmpmm.MainContactorHeartbeatReq))
	case 0x08:
		s.MainContactorRTpull(conn, addr, protoMsg.(*fsmpmm.MainContactorRTpush))
	}
}


// OHP Server
// ohpMessage
func (s *Server) ohpMessage (conn *net.UDPConn, addr net.Addr, msgTypeCode uint8, protoMsg proto.Message) {
	switch msgTypeCode {
	case 0x00:
		s.OrderPipelineLoginAns(conn, addr, protoMsg.(*fsmohp.OrderPipelineLogin))
	case 0x02:
		s.OrderPipelineHeartbeatAns(conn, addr, protoMsg.(*fsmohp.OrderPipelineHeartbeatReq))
	case 0x04:
		s.OrderPipelineRTpull(conn, addr, protoMsg.(*fsmohp.OrderPipelineRTpush))
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
	heartbeatCtr := vCImainHeartbeatReq.HeartbeatCtr.GetValue()
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
	heartbeatCtr := vCImainHeartbeatReq.HeartbeatCtr.GetValue()
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
	RTpushCtr := vCImainHeartbeatReq.RTpushCtr.GetValue()
	RTpushCtr++
	vCIChargingRTpull := &fsmvci.VCIChargingRTpull{
		ID:        vCImainHeartbeatReq.ID,
		RTpullCtr: &fsmvci.Uint32Value{Value: RTpushCtr},
		SysCtrl:   &fsmvci.SysCtrlChargingStage{StopCmd: &fsmvci.BoolEnum{Value: false}},
		FaultList: nil,
		Interval:  &fsmvci.Uint32Value{Value: 2000},
	}
	s.WriteMsg(conn,0x88, vCIChargingRTpull, addr)
}

// PowerMatrixLoginAns
// 0x80
func (s *Server) PowerMatrixLoginAns (conn *net.UDPConn, addr net.Addr, powerMatrixLogin *fsmpmm.PowerMatrixLogin) {
	powerMatrixLoginAns := &fsmpmm.PowerMatrixLoginAns{
		PowerModuleProtoVersion: powerMatrixLogin.PowerModuleProtoVersion,
		MainStateMachineVendor:  powerMatrixLogin.PowerModuleVendor,
		SelfCheckRul:            powerMatrixLogin.SelfCheckRul,
		EnableServerList:        nil,
		PramList:                nil,
		Interval:                nil,
	}
	s.WriteMsg(conn, 0x80, powerMatrixLoginAns, addr)
}

// ADModuleLoginAns
// 0x82
func (s *Server) ADModuleLoginAns (conn *net.UDPConn, addr net.Addr, ADModuleLogin *fsmpmm.ADModuleLogin) {
	ADModuleLoginAns := &fsmpmm.ADModuleLoginAns{
		MainContactorAmount:   nil,
		MatrixContactorAmount: nil,
		DCModuleAmount:        ADModuleLogin.ADModuleAmount,
		CtrlProtoVersion:      "0.1",
		CtrlVendor:            "0.1",
	}

	s.WriteMsg(conn, 0x82, ADModuleLoginAns, addr)
}

// PMMHeartbeatReq
// 0x84
func (s *Server) PMMHeartbeatReq (conn *net.UDPConn, addr net.Addr, PMMHeartbeatReq *fsmpmm.PMMHeartbeatReq) {
	heartbeatCtr := PMMHeartbeatReq.HeartbeatCtr.GetValue()
	heartbeatCtr++
	PMMHeartbeatAns := &fsmpmm.PMMHeartbeatAns{
		HeartbeatCtr: &fsmpmm.Uint32Value{Value: heartbeatCtr},
		PramList:     nil,
		SysCtrlList:  nil,
		CurrentTime:  &fsmpmm.DateTimeLong{Time: uint64(time.Now().UnixMilli()) },
		Interval:     PMMHeartbeatReq.Interval,
	}
	s.WriteMsg(conn, 0x84, PMMHeartbeatAns, addr)
}

// MainContactorHeartbeatAns
// 0x86
func (s *Server) MainContactorHeartbeatAns (conn *net.UDPConn, addr net.Addr, MainHeartbeatReq *fsmpmm.MainContactorHeartbeatReq) {
	heartbeatCtr := MainHeartbeatReq.HeartbeatCtr.GetValue()
	heartbeatCtr++
	MainHeartbeatAns := &fsmpmm.MainContactorHeartbeatAns{
		ID:            MainHeartbeatReq.ID,
		HeartbeatCtr:  &fsmpmm.Uint32Value{Value: heartbeatCtr},
		SysCtrlList:   nil,
		GunDesireList: nil,
		CurrentTime:   Utils.FsmpmmCurrentTime(0),
		Interval:      Utils.FsmpmmInterval(200),
	}
	s.WriteMsg(conn, 0x86, MainHeartbeatAns, addr)
}

// MainContactorRTpull
// 0x88
func (s *Server) MainContactorRTpull (conn *net.UDPConn, addr net.Addr, MainContactorRTpush *fsmpmm.MainContactorRTpush) {
	heartbeatCtr := MainContactorRTpush.RTpushCtr.GetValue()
	heartbeatCtr++
	MainContactorRTpull := &fsmpmm.MainContactorRTpull{
		ID:            MainContactorRTpush.ID,
		RTpullCtr:     &fsmpmm.Uint32Value{Value: heartbeatCtr},
		SysCtrlList:   nil,
		GunDesireList: nil,
		Interval:      MainContactorRTpush.Interval,
	}
	s.WriteMsg(conn, 0x88, MainContactorRTpull, addr)
}

// OrderPipelineLoginAns
// 0x80
func (s *Server) OrderPipelineLoginAns (conn *net.UDPConn, addr net.Addr, OrderPipelineLogin *fsmohp.OrderPipelineLogin) {
	OrderPipelineLoginAns :=  &fsmohp.OrderPipelineLoginAns{
		OrderPipelineProtoVersion: OrderPipelineLogin.OrderPipelineProtoVersion,
		MainStateMachineVendor:    OrderPipelineLogin.OrderPipelineVendor,
		SelfCheckRul:              OrderPipelineLogin.SelfCheckRul,
		EnableServerList:          &fsmohp.EnableServer{
			VCIServer: &fsmohp.BoolEnum{Value: true},
			PMMServer: &fsmohp.BoolEnum{Value: true},
			DMCServer: &fsmohp.BoolEnum{Value: true},
			OHPServer: &fsmohp.BoolEnum{Value: true},
			LCRServer: &fsmohp.BoolEnum{Value: true},
		},
		AllowList:                 []fsmohp.SettlementModuleEnum{1, 3, 4, 5},
	}
	s.WriteMsg(conn, 0x80, OrderPipelineLoginAns, addr)
}

// OrderPipelineHeartbeatAns
// 0x82
func (s *Server) OrderPipelineHeartbeatAns (conn *net.UDPConn, addr net.Addr, OrderPipelineHeartbeatReq *fsmohp.OrderPipelineHeartbeatReq) {
	heartbeatCtr := OrderPipelineHeartbeatReq.HeartbeatCtr.GetValue()
	heartbeatCtr++
	OrderPipelineHeartbeatAns := &fsmohp.OrderPipelineHeartbeatAns{
		HeartbeatCtr: &fsmohp.Uint32Value{Value: heartbeatCtr},
		PipelineAns:  nil,
		CurrentTime:  &fsmohp.DateTimeLong{Time: uint64(time.Now().UnixMilli())},
		Interval:     &fsmohp.Uint32Value{Value: OrderPipelineHeartbeatReq.Interval.GetValue()},
	}
	s.WriteMsg(conn, 0x82, OrderPipelineHeartbeatAns, addr)
}

// OrderPipelineRTpull
// 0x84
func (s *Server) OrderPipelineRTpull (conn *net.UDPConn, addr net.Addr, OrderPipelineRTpush *fsmohp.OrderPipelineRTpush){
	rTpushCtr := OrderPipelineRTpush.RTpushCtr.GetValue()
	OrderPipelineRTpull := &fsmohp.OrderPipelineRTpull{
		ID:        &fsmohp.Uint32Value{Value: OrderPipelineRTpush.MeterID.GetValue()},
		RTpullCtr: &fsmohp.Uint32Value{Value: rTpushCtr},
		PMMFault:  []fsmohp.PMMFaultStopEnum{},
		VCIFault:  []fsmohp.VCIFaultStopEnum{},
		Interval:  &fsmohp.Uint32Value{Value: OrderPipelineRTpush.Interval.GetValue()},
	}
	s.WriteMsg(conn, 0x84, OrderPipelineRTpull, addr)
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
