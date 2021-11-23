package udp

import (
	bytes2 "bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"FSMTestingPlatform/Protoc/fsmohp"
	"FSMTestingPlatform/Protoc/fsmpmm"
	"FSMTestingPlatform/Protoc/fsmvci"
	"FSMTestingPlatform/Utils"

	"github.com/golang/protobuf/proto"
)

// Server
// ----------------------------------------------------------------------------------
// server结构体
// ----------------------------------------------------------------------------------
type Server struct {
	MaxTime      int
	Ip           net.IP
	Prot         int
	OnlineList   map[string]*ConnWithUDP
	muOnlineList sync.Mutex
}

// ConnWithUDP
// ----------------------------------------------------------------------------------
// ConnWithUDP结构体
// ----------------------------------------------------------------------------------
type ConnWithUDP struct {
	Lived      bool
	TimeOutNum int8
	LastTime   int64
	Addr       string
}

// NewServer
// ----------------------------------------------------------------------------------
// 构建Server，返回其指针
// ----------------------------------------------------------------------------------
func NewServer(ip net.IP, port int) *Server {
	return &Server{
		MaxTime:    1000,
		Ip:         ip,
		Prot:       port,
		OnlineList: make(map[string]*ConnWithUDP),
	}
}

// Online
// ----------------------------------------------------------------------------------
// 调用connectConn函数
// ----------------------------------------------------------------------------------
func (s *Server) Online(addr string) {
	s.connectConn(addr)
}

// ReadMsg
// ----------------------------------------------------------------------------------
// Server读取数据
// ----------------------------------------------------------------------------------
func (s *Server) ReadMsg(conn *net.UDPConn) {
	for {
		bytes := make([]byte, 4096)
		n, addr, err := conn.ReadFrom(bytes)
		if err != nil {
			fmt.Println("ReadFrom is error :", err.Error())
			break
		}
		if n == 0 {
			s.unConnectConn(addr.String())
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

// isHeartBeat
// ----------------------------------------------------------------------------------
// 建立心跳，初始化超时次数和当前时间
// ----------------------------------------------------------------------------------
func (s *Server) isHeartBeat(addr string) {
	if addr == "" {
		return
	}
	s.muOnlineList.Lock()
	conn, ok := s.OnlineList[addr]
	if ok {
		conn.LastTime = time.Now().UnixMilli()
		conn.TimeOutNum = 0
	}
	s.muOnlineList.Unlock()
}

// connectConn
// ----------------------------------------------------------------------------------
// 客户端链接成功后存入Server.OnlineList
// ----------------------------------------------------------------------------------
func (s *Server) connectConn(addr string) {
	s.muOnlineList.Lock()
	s.OnlineList[addr] = &ConnWithUDP{
		Lived:      true,
		TimeOutNum: 0,
		LastTime:   time.Now().UnixMilli(),
		Addr:       addr,
	}
	s.muOnlineList.Unlock()
}

// unConnectConn
// ----------------------------------------------------------------------------------
// 客户端链接信息从Server.OnlineList删除
// ----------------------------------------------------------------------------------
func (s *Server) unConnectConn(addr string) {
	s.muOnlineList.Lock()
	delete(s.OnlineList, addr)
	s.muOnlineList.Unlock()
}

// messageTypeAssertion
// ----------------------------------------------------------------------------------
// 消息类型断言
// 不同Server消息体的Command 都是80、82、84、86、88
// ----------------------------------------------------------------------------------
func (s *Server) messageTypeAssertion(conn *net.UDPConn, addr net.Addr, msgTypeCode uint8, protoMsg proto.Message) {
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

// vciMessage
// ----------------------------------------------------------------------------------
// 判断VCI消息类型，返回相应的初始化数据
// ----------------------------------------------------------------------------------
func (s *Server) vciMessage(conn *net.UDPConn, addr net.Addr, msgTypeCode uint8, protoMsg proto.Message) {
	switch msgTypeCode {
	case 0x00:
		s.WriteVehicleChargingInterfaceLoginAns(conn, addr, protoMsg.(*fsmvci.VCIManagerRegisterInfo))
	case 0x02:
		s.WriteVCImainHeartbeatAns(conn, addr, protoMsg.(*fsmvci.VCIManagerHeartBeatInfo))
	case 0x01:
		s.WriteVCIChargingRegisterRep(conn, addr, protoMsg.(*fsmvci.VCIChargerRegisterInfo))

	case 0x06:
		s.WriteVCIChargingHeartbeatAns(conn, addr, protoMsg.(*fsmvci.VCIChargerHeartBeatInfo))
	case 0x08:
		s.WriteVCIChargingRTpull(conn, addr, protoMsg.(*fsmvci.VCIChargerRTInfo))
	}
}

// pmmMessage
// ----------------------------------------------------------------------------------
// 判断PMM消息类型，返回相应的初始化数据
// ----------------------------------------------------------------------------------
func (s *Server) pmmMessage(conn *net.UDPConn, addr net.Addr, msgTypeCode uint8, protoMsg proto.Message) {
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

// ohpMessage
// ----------------------------------------------------------------------------------
// 判断OHP消息类型，返回相应的初始化数据
// ----------------------------------------------------------------------------------
func (s *Server) ohpMessage(conn *net.UDPConn, addr net.Addr, msgTypeCode uint8, protoMsg proto.Message) {
	switch msgTypeCode {
	case 0x00:
		s.OrderPipelineLoginAns(conn, addr, protoMsg.(*fsmohp.OrderPipelineLogin))
	case 0x02:
		s.OrderPipelineHeartbeatAns(conn, addr, protoMsg.(*fsmohp.OrderPipelineHeartbeatReq))
	case 0x04:
		s.OrderPipelineRTpull(conn, addr, protoMsg.(*fsmohp.OrderPipelineRTpush))
	}
}

// WriteVehicleChargingInterfaceLoginAns
// ----------------------------------------------------------------------------------
// VCI 0x80
// ----------------------------------------------------------------------------------
func (s *Server) WriteVehicleChargingInterfaceLoginAns(conn *net.UDPConn, addr net.Addr, register *fsmvci.VCIManagerRegisterInfo) {
	vCIGunPrametersList := make([]*fsmvci.VCIGunPrameters, 0)
	vCISysParametersList := make([]*fsmvci.VCISysParameters, 0)
	chargerReviveList := make([]*fsmvci.ChargerRevive, 0)

	vCIPram := Utils.VCIPramInit(1, 1, 1, 0)
	vCIGunPrametersList = append(vCIGunPrametersList, vCIPram)

	sysParameterList := Utils.SysParameterInit()
	vCISysParametersList = append(vCISysParametersList, sysParameterList)

	chargerRecover := Utils.ChargingRecoverInit(1)
	chargerReviveList = append(chargerReviveList, chargerRecover)

	vCIManagerRegisterReponse := &fsmvci.VCIManagerRegisterReponse{
		ReProtocolVer:             register.ProtocolVer,
		ReFSMEditionNumber:        register.EditionNumber,
		ReVCIGunPrametersListLong: 4,
		ReVCISysParameterListLong: 4,
		ReChargerReviveListLong:   4,
		ReSelfCheckResult:         1,
		ReEnableServer: &fsmvci.EnableServer{
			VCIServer: 1,
			PMMServer: 1,
			DMCServer: 0,
			OHPServer: 1,
			LCRServer: 0,
		},
		ReVCIGunPrametersList: vCIGunPrametersList,
		ReVCISysParameterList: vCISysParametersList,
		ReChargerReviveList:   chargerReviveList,
	}

	s.WriteMsg(conn, 0x80, vCIManagerRegisterReponse, addr)
}

// WriteVCImainHeartbeatAns
// ----------------------------------------------------------------------------------
// VCI 0x82
// ----------------------------------------------------------------------------------
func (s *Server) WriteVCImainHeartbeatAns(conn *net.UDPConn, addr net.Addr, managerHeart *fsmvci.VCIManagerHeartBeatInfo) {
	heartbeatCtr := managerHeart.HeartBeatCnt
	heartbeatCtr++

	vCIGunPrametersList := make([]*fsmvci.VCIGunPrameters, 0)
	vCISysParametersList := make([]*fsmvci.VCISysParameters, 0)
	sysCtrlConnectStageList := make([]*fsmvci.SysCtrlConnectStage, 0)

	vCIPram := Utils.VCIPramInit(1, 1, 1, 0)
	vCIGunPrametersList = append(vCIGunPrametersList, vCIPram)

	sysParameterList := Utils.SysParameterInit()
	vCISysParametersList = append(vCISysParametersList, sysParameterList)

	SysCtrlConnectStage := Utils.SysCtrlConnectStageInit()
	sysCtrlConnectStageList = append(sysCtrlConnectStageList, SysCtrlConnectStage)

	vCImainHeartbeatAns := &fsmvci.VCIManagerHeartBeatReponse{
		ReHeartbeatCnt:             heartbeatCtr,
		ReTimeNow:                  uint64(time.Now().UnixMilli()),
		ReHeartBeatPeriod:          managerHeart.HeartBeatPeriod,
		ReVCIGunPrametersListLong:  6,
		ReVCISysParametersListLong: 6,
		ReSysCtrlStateLong:         6,
		ReVCIGunParametersList:     vCIGunPrametersList,
		ReVCISysParametersList:     vCISysParametersList,
		ReSysCtrlState:             sysCtrlConnectStageList,
	}
	s.WriteMsg(conn, 0x82, vCImainHeartbeatAns, addr)
}

// WriteVCIChargingRegisterRep
// ----------------------------------------------------------------------------------
// VCI 0x81
// ----------------------------------------------------------------------------------
func (s *Server) WriteVCIChargingRegisterRep(conn *net.UDPConn, addr net.Addr, info *fsmvci.VCIChargerRegisterInfo) {
	info.Register++
	vciChargerRegister := Utils.VCIChargerRegisterResponse(info.Register, uint32(s.Prot))

	s.WriteMsg(conn, 0x81, vciChargerRegister, addr)
}

// WriteVCIChargingHeartbeatAns
// ----------------------------------------------------------------------------------
// VCI 0x86
// ----------------------------------------------------------------------------------
func (s *Server) WriteVCIChargingHeartbeatAns(conn *net.UDPConn, addr net.Addr, vCIChargingHeartBeatInfo *fsmvci.VCIChargerHeartBeatInfo) {
	heartbeatCtr := vCIChargingHeartBeatInfo.HeartbeatCnt
	heartbeatCtr++

	vCIChargingHeartBeatResponse := &fsmvci.VCIChargerHeartBeatResponse{
		ReChargingID:      vCIChargingHeartBeatInfo.ChargingID,
		ReHeartBeatCnt:    heartbeatCtr,
		ReTimeNow:         uint64(time.Now().UnixMilli()),
		ReHeartBeatPeriod: vCIChargingHeartBeatInfo.HeartBeatPeriod,
		ReGunCaredMsg: &fsmvci.GunCaredInfo{
			AllocaOK:       1,
			OutConnectorFb: 0,
			MeterVoltage:   150,
			MeterCurrent:   36,
			BatVoltage:     50,
			ModVoltage:     50,
		},
	}

	s.WriteMsg(conn, 0x86, vCIChargingHeartBeatResponse, addr)
}

// WriteVCIChargingRTpull
// ----------------------------------------------------------------------------------
// VCI 0x88
// ----------------------------------------------------------------------------------
func (s *Server) WriteVCIChargingRTpull(conn *net.UDPConn, addr net.Addr, vCImainHeartbeatReq *fsmvci.VCIChargerRTInfo) {
	RTpushCtr := vCImainHeartbeatReq.PushCnt
	RTpushCtr++
	vCIChargerRTRsponse := &fsmvci.VCIChargerRTRsponse{
		EchoChargerID: vCImainHeartbeatReq.ChargerID,
		EchoPushCnt:   RTpushCtr,
		EchoRtPeriod:  vCImainHeartbeatReq.RtPeriod,
		EchoSysCtrl:   nil,
		EchoFaultList: nil,
	}

	s.WriteMsg(conn, 0x88, vCIChargerRTRsponse, addr)
}

// PowerMatrixLoginAns
// ----------------------------------------------------------------------------------
// PMM 0x80
// ----------------------------------------------------------------------------------
func (s *Server) PowerMatrixLoginAns(conn *net.UDPConn, addr net.Addr, powerMatrixLogin *fsmpmm.PowerMatrixLogin) {
	powerMatrixLoginAns := &fsmpmm.PowerMatrixLoginAns{
		PowerModuleProtoVersion: powerMatrixLogin.PowerModuleProtoVersion,
		MainStateMachineVendor:  powerMatrixLogin.PowerModuleVendor,
		SelfCheckRul:            powerMatrixLogin.SelfCheckRul,
		EnableServerList:        nil,
		PramList:                nil,
		Interval:                1000,
	}
	s.WriteMsg(conn, 0x80, powerMatrixLoginAns, addr)
}

// ADModuleLoginAns
// ----------------------------------------------------------------------------------
// PMM 0x82
// ----------------------------------------------------------------------------------
func (s *Server) ADModuleLoginAns(conn *net.UDPConn, addr net.Addr, ADModuleLogin *fsmpmm.ADModuleLogin) {
	ADModuleLoginAns := &fsmpmm.ADModuleLoginAns{
		MainContactorAmount:   6,
		MatrixContactorAmount: 6,
		DCModuleAmount:        ADModuleLogin.ADModuleAmount,
		CtrlProtoVersion:      "0.1",
		CtrlVendor:            "0.1",
	}

	s.WriteMsg(conn, 0x82, ADModuleLoginAns, addr)
}

// PMMHeartbeatReq
// ----------------------------------------------------------------------------------
// PMM 0x84
// ----------------------------------------------------------------------------------
func (s *Server) PMMHeartbeatReq(conn *net.UDPConn, addr net.Addr, PMMHeartbeatReq *fsmpmm.PMMHeartbeatReq) {
	heartbeatCtr := PMMHeartbeatReq.HeartbeatCtr
	heartbeatCtr++
	PMMHeartbeatAns := &fsmpmm.PMMHeartbeatAns{
		HeartbeatCtr: heartbeatCtr,
		PramList:     nil,
		SysCtrlList:  nil,
		CurrentTime:  uint64(time.Now().UnixMilli()),
		Interval:     PMMHeartbeatReq.Interval,
	}
	s.WriteMsg(conn, 0x84, PMMHeartbeatAns, addr)
}

// MainContactorHeartbeatAns
// ----------------------------------------------------------------------------------
// PMM 0x86
// ----------------------------------------------------------------------------------
func (s *Server) MainContactorHeartbeatAns(conn *net.UDPConn, addr net.Addr, MainHeartbeatReq *fsmpmm.MainContactorHeartbeatReq) {
	heartbeatCtr := MainHeartbeatReq.HeartbeatCtr
	heartbeatCtr++
	MainHeartbeatAns := &fsmpmm.MainContactorHeartbeatAns{
		ID:            MainHeartbeatReq.ID,
		HeartbeatCtr:  heartbeatCtr,
		SysCtrlList:   nil,
		GunDesireList: nil,
		CurrentTime:   uint64(time.Now().UnixMilli()),
		Interval:      1000,
	}
	s.WriteMsg(conn, 0x86, MainHeartbeatAns, addr)
}

// MainContactorRTpull
// ----------------------------------------------------------------------------------
// PMM 0x88
// ----------------------------------------------------------------------------------
func (s *Server) MainContactorRTpull(conn *net.UDPConn, addr net.Addr, MainContactorRTpush *fsmpmm.MainContactorRTpush) {
	heartbeatCtr := MainContactorRTpush.RTpushCtr
	heartbeatCtr++
	MainContactorRTpull := &fsmpmm.MainContactorRTpull{
		ID:            MainContactorRTpush.ID,
		RTpullCtr:     heartbeatCtr,
		SysCtrlList:   nil,
		GunDesireList: nil,
		Interval:      MainContactorRTpush.Interval,
	}
	s.WriteMsg(conn, 0x88, MainContactorRTpull, addr)
}

// OrderPipelineLoginAns
// ----------------------------------------------------------------------------------
// OHP 0x80
// ----------------------------------------------------------------------------------
func (s *Server) OrderPipelineLoginAns(conn *net.UDPConn, addr net.Addr, OrderPipelineLogin *fsmohp.OrderPipelineLogin) {
	OrderPipelineLoginAns := &fsmohp.OrderPipelineLoginAns{
		OrderPipelineProtoVersion: OrderPipelineLogin.OrderPipelineProtoVersion,
		MainStateMachineVendor:    OrderPipelineLogin.OrderPipelineVendor,
		SelfCheckRul:              OrderPipelineLogin.SelfCheckRul,
		EnableServerList: &fsmohp.EnableServer{
			VCIServer: true,
			PMMServer: true,
			DMCServer: true,
			OHPServer: true,
			LCRServer: true,
		},
		AllowList: []fsmohp.SettlementModuleEnum{1, 3, 4, 5},
	}
	s.WriteMsg(conn, 0x80, OrderPipelineLoginAns, addr)
}

// OrderPipelineHeartbeatAns
// ----------------------------------------------------------------------------------
// OHP 0x82
// ----------------------------------------------------------------------------------
func (s *Server) OrderPipelineHeartbeatAns(conn *net.UDPConn, addr net.Addr, OrderPipelineHeartbeatReq *fsmohp.OrderPipelineHeartbeatReq) {
	heartbeatCtr := OrderPipelineHeartbeatReq.HeartbeatCtr
	heartbeatCtr++
	OrderPipelineHeartbeatAns := &fsmohp.OrderPipelineHeartbeatAns{
		HeartbeatCtr: heartbeatCtr,
		PipelineAns:  nil,
		CurrentTime:  uint64(time.Now().UnixMilli()),
		Interval:     OrderPipelineHeartbeatReq.Interval,
	}
	s.WriteMsg(conn, 0x82, OrderPipelineHeartbeatAns, addr)
}

// OrderPipelineRTpull
// ----------------------------------------------------------------------------------
// OHP 0x84
// ----------------------------------------------------------------------------------
func (s *Server) OrderPipelineRTpull(conn *net.UDPConn, addr net.Addr, OrderPipelineRTpush *fsmohp.OrderPipelineRTpush) {
	rTpushCtr := OrderPipelineRTpush.RTpushCtr
	OrderPipelineRTpull := &fsmohp.OrderPipelineRTpull{
		ID:        OrderPipelineRTpush.MeterID,
		RTpullCtr: rTpushCtr,
		PMMFault:  []fsmohp.PMMFaultStopEnum{},
		VCIFault:  []fsmohp.VCIFaultStopEnum{},
		Interval:  OrderPipelineRTpush.Interval,
	}
	s.WriteMsg(conn, 0x84, OrderPipelineRTpull, addr)
}

// WriteHeartBeat
// ----------------------------------------------------------------------------------
// 发送心跳数据，初始化心跳数据
// ----------------------------------------------------------------------------------
func (s *Server) WriteHeartBeat(conn *net.UDPConn, msgTypeCode uint, protoMsg proto.Message, addr net.Addr) {
	s.isHeartBeat(addr.String())
	s.WriteMsg(conn, msgTypeCode, protoMsg, addr)
}

// HeartBeatCheck
// ----------------------------------------------------------------------------------
// 心跳检测
// ----------------------------------------------------------------------------------
func (s *Server) HeartBeatCheck() {
	for {
		sleepTime := time.NewTimer(time.Duration(s.MaxTime) * time.Millisecond)
		select {
		case <-sleepTime.C:
			for _, v := range s.OnlineList {
				nowTime := time.Now().UnixMilli()
				if nowTime > int64(s.MaxTime)+v.LastTime {
					if v.TimeOutNum == 2 {
						fmt.Println("leave", v.Addr)
						s.unConnectConn(v.Addr)
					} else {
						v.TimeOutNum++
						v.LastTime = nowTime
						fmt.Printf("this time out %d \n", v.TimeOutNum)
					}
				}
			}
		}
	}
}

// WriteMsg
// ----------------------------------------------------------------------------------
// 数据发送
// ----------------------------------------------------------------------------------
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

// BuildUdpServer
// ----------------------------------------------------------------------------------
// 创建UDP，并开启监听模式
// ----------------------------------------------------------------------------------
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
	UdpServer := NewServer(net.IPv4(0, 0, 0, 0), 9000)
	udpConn, err := UdpServer.BuildUdpServer()
	if err != nil {
		fmt.Printf("udp connect error : %s", err.Error())
		return
	}
	go UdpServer.HeartBeatCheck()
	go UdpServer.ReadMsg(udpConn)
	for {

	}
}
