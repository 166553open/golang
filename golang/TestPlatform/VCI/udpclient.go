package main

import (
	"bufio"
	"fmt"
	"goWork/Utils"
	"goWork/protoc/fsmvci"
	"goWork/udp"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type MainThread struct {
	MainGunAmount uint8
	MainGunFree uint8
	MainGunWork uint8

	muMain sync.Mutex
	MainGunClient *udp.Client
	FatherClient *FatherThread
	SonClient *SonThread

	MainGunList map[uint32]bool
	MainStates *ThreadStates
}

type ThreadStates struct {
	muThread sync.Mutex
	FatherLive bool
	SonLive bool

	FatherStates map[uint32]bool
	SonStates map[uint32]bool
}

type FatherThread struct {
	FatherGunAmount uint8
	FatherGunFree uint8
	FatherGunWork uint8
	DefaultInterval uint16
	muFather sync.Mutex
	FatherStates *ThreadStates
	FatherGunList map[uint32]*udp.Client
}

type SonThread struct {
	SonGunAmount uint8
	SonGunFree uint8
	SonGunWork uint8
	DefaultInterval uint16
	muSon sync.Mutex
	SonStates *ThreadStates
	SonGunList map[uint32]*udp.Client
}

func NewMainThread () *MainThread {
	threadStates := NewThreadStates()
	fatherThread := NewFatherThread(threadStates)
	sonThread := NewSonThread(threadStates)

	addr := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9000,
	}
	client, _ :=udp.NewClient(addr)

	return &MainThread{
		MainGunAmount: 6,
		MainGunFree:   6,
		MainGunWork:   0,
		MainStates:	   threadStates,
		MainGunClient: client,
		FatherClient:  fatherThread,
		SonClient:     sonThread,
		MainGunList:   make(map[uint32]bool),
	}
}

func NewFatherThread (states *ThreadStates) *FatherThread {
	return &FatherThread{
		FatherGunAmount: 6,
		FatherGunFree:   6,
		FatherGunWork:   0,
		DefaultInterval: 200,
		FatherStates: states,
		FatherGunList:   make(map[uint32]*udp.Client),
	}
}

func NewSonThread (states *ThreadStates) *SonThread {
	return &SonThread{
		SonGunAmount: 6,
		SonGunFree:   6,
		SonGunWork:   0,
		DefaultInterval: 200,
		SonStates: 	states,
		SonGunList:   make(map[uint32]*udp.Client),
	}
}

func NewThreadStates () *ThreadStates {
	return &ThreadStates{
		FatherLive:   false,
		SonLive:      false,
		FatherStates: make(map[uint32]bool),
		SonStates:    make(map[uint32]bool),
	}
}

func (m *MainThread) BuildMainClient () (*udp.Client, error) {
	if m.MainGunClient == nil {
		addr := net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 9000,
		}

		client, err :=udp.NewClient(addr)
		if err != nil {
			return nil, err
		}
		m.MainGunClient = client
		return client, nil
	}else{
		return m.MainGunClient, nil
	}
}

func (m *MainThread) VCIMainLoginInit(client *udp.Client) *fsmvci.VehicleChargingInterfaceLogin {
	return &fsmvci.VehicleChargingInterfaceLogin{
		VehicleChargingProtoVersion: Utils.ConfProtoc["VehicleChargingProtoVersion"].(string),
		VehicleChargingVendor:       Utils.ConfProtoc["VehicleChargingProtoVersion"].(string),
		SelfCheckRul:                1,
	}
}

func (m *MainThread) VCImainHeartInit (client *udp.Client) *fsmvci.VCImainHeartbeatReq {
	gunBaseList := make([]*fsmvci.GunBaseStatus, 0)
	for k, _ := range m.MainGunList {
		gunBaseStatus := Utils.GunBaseStatusInit(uint32(k), true, false)
		gunBaseList = append(gunBaseList, gunBaseStatus)
	}
	client.Mu.Lock()
	client.GunHeartbeatCtr++
	client.Mu.Unlock()

	return &fsmvci.VCImainHeartbeatReq{
		HeartbeatCtr: &fsmvci.Uint32Value{Value: uint32(client.GunHeartbeatCtr)},
		GunBaseList:  gunBaseList,
		CurrentTime:  &fsmvci.DateTimeLong{Time: uint64(time.Now().Unix()) },
		Interval:     &fsmvci.Uint32Value{Value: 200},
	}
}

func (m *MainThread) HeartBeat() {
	newTimer := m.MainGunClient.Timer
	for{
		select {
		case <-newTimer.C:
			m.MainGunClient.WriteMsg(uint(0x02), m.VCImainHeartInit(m.MainGunClient))
			newTimer = time.NewTimer(time.Duration(m.MainGunClient.GunInterval)*time.Millisecond)
		}
	}
}

func (f *FatherThread) BuildFatherClient (GunId uint32) (*udp.Client, error) {
	if _, ok := f.FatherGunList[GunId]; !ok {
		addr := net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 9010,
		}

		client, err :=udp.NewClient(addr)
		if err != nil {
			return nil, err
		}
		f.muFather.Lock()
		f.FatherGunFree--
		f.FatherGunWork++
		f.FatherGunList[GunId] = client
		f.muFather.Unlock()
		return client, nil
	}else{
		fmt.Printf("This Gun is working\n")
		return f.FatherGunList[GunId], nil
	}
}

func (f *FatherThread) DeleteFatherClient (GunId uint32) {
	if _, ok := f.FatherGunList[GunId]; ok {
		f.muFather.Lock()
		delete(f.FatherGunList, GunId)
		f.FatherGunFree++
		f.FatherGunWork--
		f.muFather.Unlock()

		f.FatherStates.muThread.Lock()
		delete (f.FatherStates.FatherStates, GunId)
		f.FatherStates.muThread.Unlock()
	}
}

func (f *FatherThread) VCIPluggedHeartbeatReqInit (client *udp.Client) *fsmvci.VCIPluggedHeartbeatReq {
	client.GunHeartbeatCtr++
	return &fsmvci.VCIPluggedHeartbeatReq{
		ID:           &fsmvci.Uint32Value{Value: 1},
		HeartbeatCtr: &fsmvci.Uint32Value{Value: uint32(client.GunHeartbeatCtr)},
		GunStatus:    &fsmvci.GunPluggedStatus{
			AuxPowerDrv: &fsmvci.BoolEnum{Value: true},
			AuxPowerFb:  &fsmvci.BoolEnum{Value: true},
			ElockDrv:    &fsmvci.BoolEnum{Value: true},
			ElockFb:     &fsmvci.BoolEnum{Value: true},
			TPos:        &fsmvci.Int32Value{Value: 50},
			TNeg:        &fsmvci.Int32Value{Value: 60},
		},
		CurrentTime:  &fsmvci.DateTimeLong{Time: uint64(time.Now().Unix())},
		Interval:     &fsmvci.Uint32Value{Value: 200},
	}
}

func (f *FatherThread) HeartBeat (GunId uint32) {
	client, _ := f.BuildFatherClient(GunId)

	newTimer := client.Timer
	for{
		select {
		case <-newTimer.C:
			if client, ok := f.FatherGunList[GunId]; ok {
				client.WriteMsg(uint(0x04), f.VCIPluggedHeartbeatReqInit(client))
				newTimer = time.NewTimer(time.Duration(client.GunInterval)*time.Millisecond)
			}else{
				goto CloseHeart
			}
		}
	}
	CloseHeart:
		fmt.Printf("fatherThread GunId:%d logout, heart beat interrupt\n", GunId)
}

func (s *SonThread) BuildSonClient (GunId uint32) (*udp.Client, error) {
	if _, ok := s.SonGunList[GunId]; !ok {
		addr := net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 9010,
		}

		client, err :=udp.NewClient(addr)
		if err != nil {
			return nil, err
		}
		s.muSon.Lock()
		s.SonGunFree--
		s.SonGunWork++
		s.SonGunList[GunId] = client
		s.muSon.Unlock()
		return client, nil
	}else{
		fmt.Printf("This Gun is working\n")
		return s.SonGunList[GunId], nil
	}
}

func (s *SonThread) DeleteSonClient (GunId uint32) {
	if _, ok := s.SonGunList[GunId]; ok {
		s.muSon.Lock()
		delete(s.SonGunList, GunId)
		s.SonGunFree++
		s.SonGunWork--
		s.muSon.Unlock()

		s.SonStates.muThread.Lock()
		delete(s.SonStates.SonStates, GunId)
		s.SonStates.muThread.Unlock()
	}
}

func (s *SonThread) VCIChargingHeartbeatReqInit(client *udp.Client) *fsmvci.VCIChargingHeartbeatReq {
	chargingFaultStateList := make([]*fsmvci.ChargingFaultState, 0)

	chargingFaultStateList = append(chargingFaultStateList, &fsmvci.ChargingFaultState{
		FaultName:     fsmvci.FaultEnum(0),
		FaultType:     fsmvci.FaultState_FaultSustained,
		FaultTime:     nil,
		FaultDownTime: nil,
	})
	client.GunHeartbeatCtr++
	return &fsmvci.VCIChargingHeartbeatReq{
		ID:                 &fsmvci.Uint32Value{Value: 1},
		HeartbeatCtr:       &fsmvci.Uint32Value{Value: uint32(client.GunHeartbeatCtr)},
		GunStatus0:         &fsmvci.BmsShakehands{
			BmsVolMaxAllowed: &fsmvci.DoubleValue{Value: 60.22},
			GBTProtoVersion:  "1.0",
		},
		GunStatus1:         &fsmvci.BmsIdentify{
			BatteryType:     &fsmvci.Int32Value{Value: 1},
			CapacityRated:   &fsmvci.DoubleValue{Value: 60.30},
			VoltageRated:    &fsmvci.DoubleValue{Value: 60.30},
			BatteryVendor:   "WanMa",
			BatterySequence: &fsmvci.Int32Value{Value: 1},
			ProduceDate:     "2022.04.01",
			ChargeCount:     &fsmvci.Int32Value{Value: 0},
			RightIdentifier: &fsmvci.Int32Value{Value: 1},
			BmsVersion:      "1.0",
			BmsAndCarId:     "1",
			BmsVIN:          "yes",
		},
		GunStatus2:         &fsmvci.BmsConfig{
			VIndAllowedMax: &fsmvci.DoubleValue{Value: 60.00},
			IAllowedMax:    &fsmvci.DoubleValue{Value: 50.00},
			EnergyRated:    &fsmvci.DoubleValue{Value: 300.00},
			VAllowedMax:    &fsmvci.DoubleValue{Value: 50.00},
			TAllowedMax:    &fsmvci.DoubleValue{Value: 50.00},
			StartSoc:       &fsmvci.DoubleValue{Value: 50.00},
			VCurrent:       &fsmvci.DoubleValue{Value: 50.00},
			VCOutputMax:    &fsmvci.DoubleValue{Value: 50.00},
			VCOutputMin:    &fsmvci.DoubleValue{Value: 50.00},
			ICOutputMax:    &fsmvci.DoubleValue{Value: 50.00},
			ICOutputMin:    &fsmvci.DoubleValue{Value: 50.00},
		},
		GunStatus3:         &fsmvci.BmsCharging{
			VDemand:         &fsmvci.DoubleValue{Value: 5.00},
			IDemand:         &fsmvci.DoubleValue{Value: 5.00},
			CurrentSoc:      &fsmvci.DoubleValue{Value: 5.00},
			RemainTime:      &fsmvci.DoubleValue{Value: 5.00},
			ChargeMode:      1,
			VMeasure:        &fsmvci.DoubleValue{Value: 5.00},
			IMeasure:        &fsmvci.DoubleValue{Value: 5.00},
			VIndMax:         &fsmvci.DoubleValue{Value: 5.00},
			VIndMaxCode:     &fsmvci.Int32Value{Value: 5},
			VIndMin:         &fsmvci.DoubleValue{Value: 5.00},
			VIndMinCode:     &fsmvci.Int32Value{Value: 5},
			TMax:            &fsmvci.DoubleValue{Value: 5.00},
			TMaxCode:        &fsmvci.Int32Value{Value: 5},
			TMin:            &fsmvci.DoubleValue{Value: 5.00},
			TMinCode:        &fsmvci.Int32Value{Value: 5},
			ChargeAllow:     &fsmvci.BoolEnum{Value: true},
			VIndHigh:        &fsmvci.BoolEnum{Value: false},
			VIndLow:         &fsmvci.BoolEnum{Value: false},
			SoHigh:          &fsmvci.BoolEnum{Value: false},
			SocLow:          &fsmvci.BoolEnum{Value: false},
			IHigh:           &fsmvci.BoolEnum{Value: false},
			THigh:           &fsmvci.BoolEnum{Value: false},
			Insulation:      &fsmvci.BoolEnum{Value: false},
			OutputConnector: &fsmvci.BoolEnum{Value: false},
			VIndMaxGroupNum: &fsmvci.Int32Value{Value: 5},
			HeatingMode:     &fsmvci.Int32Value{Value: 0},
		},
		GunStatus4:         &fsmvci.BmsChargeFinish{
			EndSoc:             &fsmvci.DoubleValue{Value: 6.00},
			VMinIndividal:      &fsmvci.DoubleValue{Value: 4.00},
			VMaxIndividal:      &fsmvci.DoubleValue{Value: 4.00},
			TemperatureMin:     &fsmvci.DoubleValue{Value: 4.00},
			TemperatureMax:     &fsmvci.DoubleValue{Value: 4.00},
			BmsStopReason:      &fsmvci.Int32Value{Value: 4},
			BmsFaultReason:     &fsmvci.Int32Value{Value: 4},
			BmsErrorReason:     &fsmvci.Int32Value{Value: 4},
			ChargerStopReason:  &fsmvci.Int32Value{Value: 4},
			ChargerFaultReason: &fsmvci.Int32Value{Value: 4},
			ChargerErrorReason: &fsmvci.Int32Value{Value: 4},
			BmsEFrame:          &fsmvci.Int32Value{Value: 4},
			ChargerEFrame:      &fsmvci.Int32Value{Value: 4},
		},
		ReConnect:          &fsmvci.BMSReConnectEvent{
			TimeOutState:   &fsmvci.Int32Value{Value: 1},
			BMSTimeoutType: 0,
			ReconnectCnt:   &fsmvci.Int32Value{Value: 0},
			NextState:      &fsmvci.Int32Value{Value: 0},
		},
		FaultList:          chargingFaultStateList,
		ChargingRecoverMsg: nil,
		CurrentTime:        &fsmvci.DateTimeLong{Time: uint64(time.Now().Unix())},
		Interval:           &fsmvci.Uint32Value{Value: 200},
	}
}

func (s *SonThread) VCIChargingRTpushInit (client *udp.Client) *fsmvci.VCIChargingRTpush {
	chargingFaultStateList := make([]*fsmvci.ChargingFaultState, 0)
	chargingFaultStateList = append(chargingFaultStateList, &fsmvci.ChargingFaultState{
		FaultName:     0,
		FaultType:     0,
		FaultTime:     nil,
		FaultDownTime: nil,
	})

	return &fsmvci.VCIChargingRTpush{
		ID:           &fsmvci.Uint32Value{Value: 1},
		RTpushCtr:    &fsmvci.Uint32Value{Value: 1},
		GunDesireMsg: &fsmvci.GunDesire{
			VDemand:            &fsmvci.FloatValue{Value: 5.00},
			IDemand:            &fsmvci.FloatValue{Value: 5.00},
			VPTPDemand:         &fsmvci.FloatValue{Value: 5.00},
			IPTPDemand:         &fsmvci.FloatValue{Value: 5.00},
			OutConnectorDemand: &fsmvci.BoolEnum{Value: true},
		},
		GunHaltMsg:   &fsmvci.GunHalt{
			StopState: &fsmvci.BoolEnum{Value: false},
			HaltType:  0,
			Faultcode: 0,
		},
		ReConnect:    &fsmvci.BMSReConnectEvent{
			TimeOutState:   &fsmvci.Int32Value{Value: 0},
			BMSTimeoutType: 0,
			ReconnectCnt:   &fsmvci.Int32Value{Value: 0},
			NextState:      &fsmvci.Int32Value{Value: 0},
		},
		FaultList:    chargingFaultStateList,
		Interval:     &fsmvci.Uint32Value{Value: 200},
	}
}

func (s *SonThread) HeartBeat(GunId uint32) {
	client, _ := s.BuildSonClient(GunId)
	client.WriteMsg(uint(0x08), s.VCIChargingRTpushInit(client))

	newTimer := client.Timer
	for {
		select {
		case <-newTimer.C:
			if client, ok := s.SonGunList[GunId]; ok {
				s.SonGunList[GunId].WriteMsg(uint(0x06), s.VCIChargingHeartbeatReqInit(s.SonGunList[GunId]))
				newTimer = time.NewTimer(time.Duration(client.GunInterval)*time.Millisecond)
			}else{
				goto CloseHeart
			}
		}
	}
	CloseHeart:
		fmt.Printf("sonThread GunId:%d logout, heart beat interrupt\n", GunId)
}

func (m *MainThread) ReadConsole() {
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
			gunId, err := Utils.StringToUint32AndLimit(target)
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

						if _, ok := m.MainGunList[uint32(i)]; !ok {
							//开启 父级线程 心跳和注册信息
							go m.FatherClient.HeartBeat(uint32(i))
							//开启 子级线程 心跳和注册信息
							go m.SonClient.HeartBeat(uint32(i))

							m.MainStates.FatherStates[uint32(i)] = true
							m.MainStates.SonStates[uint32(i)] = true
							m.MainGunFree--
							m.MainGunWork++
							m.MainGunList[uint32(i)] = true
						}
						fmt.Println(m.MainGunList)
					}
				}else{
					if _, ok := m.MainGunList[gunId]; ok {
						fmt.Printf("Gun %d is working and cannot be opened repeatedly\n", gunId)
					}else{
						if m.MainGunWork == 0 {
							m.MainStates.FatherLive, m.MainStates.SonLive = true, true
						}
						//开启 父级线程 心跳和注册信息
						go m.FatherClient.HeartBeat(gunId)
						//开启 子级线程 心跳和注册信息
						go m.SonClient.HeartBeat(gunId)

						m.MainStates.FatherStates[gunId] = true
						m.MainStates.SonStates[gunId] = true
						m.MainGunFree--
						m.MainGunWork++
						m.MainGunList[gunId] = true
					}
				}
			}
			if action == "delete" {
				if gunId == 0 {
					for k, _ := range m.MainGunList {
						delete(m.MainGunList, k)
						m.FatherClient.DeleteFatherClient(k)
						m.SonClient.DeleteSonClient(k)
						m.MainGunWork--
						m.MainGunFree++
						if m.MainGunWork == 0 {
							m.MainStates.FatherLive, m.MainStates.SonLive = false, false
						}
					}
				}else{
					if _, ok := m.MainGunList[gunId]; !ok {
						fmt.Printf("Gun %d is already dormant\n", gunId)
					}else{
						m.MainGunWork--
						m.MainGunFree++
						if m.MainGunWork == 0 {
							m.MainStates.FatherLive, m.MainStates.SonLive = false, false
						}
						//关闭 线程
						delete(m.MainGunList, gunId)
						m.FatherClient.DeleteFatherClient(gunId)
						m.SonClient.DeleteSonClient(gunId)
					}
				}
			}

			m.muMain.Unlock()
		}
	}
}

func main () {
	mainThread := NewMainThread()
	//go mainThread.BuildMainClient()
	go mainThread.MainGunClient.WriteMsg(uint(0x00), mainThread.VCIMainLoginInit(mainThread.MainGunClient))
	go mainThread.HeartBeat()
	go mainThread.ReadConsole()
	for{

	}
}
