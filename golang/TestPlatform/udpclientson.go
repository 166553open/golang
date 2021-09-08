package main

import (
	"bufio"
	"fmt"
	"goWork/protoc/fsmvci"
	"goWork/udp"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SonUdp struct {
	GunAmount int8
	GunFree int8
	GunWork int8
	DefaultInterval uint32
	GunList map[uint32]*udp.Client
	mu sync.Mutex
}

type SonClient struct {
	Addr net.Addr
	UdpConn *udp.Client
}

func NewSonUdp () *SonUdp {
	return &SonUdp{
		GunAmount: 6,
		GunFree:   6,
		GunWork:   0,
		DefaultInterval: uint32(200),
		GunList:   make(map[uint32]*udp.Client),
		mu:        sync.Mutex{},
	}
}

func (s *SonUdp) BuildSonClient(GunId uint32) *udp.Client {
	if GunId > 6 || GunId < 1 {
		fmt.Printf("GunId is out of range 1, 6 %d\n", GunId)
		return nil
	}
	if _, ok := s.GunList[GunId]; !ok {

		addr := net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 9020,
		}

		client, err :=udp.NewClient(addr)
		if err != nil {
			return nil
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.GunList[GunId] = client
		return client
	}else{
		fmt.Printf("This Gun is working\n")
		return s.GunList[GunId]
	}
}

func (s *SonUdp) DeleteSonClient(GunId uint32) {
	if GunId > 6 || GunId < 1 {
		fmt.Printf("GunId is out of range 1, 6 %d\n", GunId)
		return
	}
	if _, ok := s.GunList[GunId]; ok {
		s.mu.Lock()
		delete(s.GunList, GunId)
		s.mu.Unlock()
	}else{
		fmt.Printf("This Gun is not work\n")
		return
	}
}


func (s *SonUdp) VCIChargingHeartbeatReqInit(client *udp.Client) *fsmvci.VCIChargingHeartbeatReq {
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

func (s *SonUdp) VCIChargingRTpushInit (client *udp.Client) *fsmvci.VCIChargingRTpush {
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


func (s *SonUdp) ReadConsole() {
	inputReader := bufio.NewReader(os.Stdin)
	for {
		str, err := inputReader.ReadString('\n')
		if err != nil {
			fmt.Printf("Input error :\n", err.Error())
			return
		}
		str = strings.TrimSpace(str)
		if strings.Contains(str, ":") {
			strList := strings.Split(str, ":")
			if strList[0] == "son_build" {
				gunId, _ := strconv.Atoi(strList[1])
				if gunId > 6 {
					gunId = 6
				}
				if gunId < 1 {
					gunId = 1
				}
				go s.BuildRun(uint32(gunId))
			}else if strList[0] == "son_delete" {
				gunId, _ := strconv.Atoi(strList[1])
				if gunId > 6 {
					gunId = 6
				}
				if gunId < 1 {
					gunId = 1
				}
				s.DeleteSonClient(uint32(gunId))
			}
		}else{
			fmt.Printf("Input is :\n", str)
		}
	}
}

func (s *SonUdp) BuildRun(GunId uint32) {
	client := s.BuildSonClient(GunId)
	client.WriteMsg(uint(0x08), s.VCIChargingRTpushInit(client))
	go func() {
		newTimer := time.NewTimer(time.Duration(s.DefaultInterval)*time.Microsecond)
		for{
			select {
			case <-newTimer.C:
				for _, v := range s.GunList{
					newTimer = time.NewTimer(time.Duration(v.GunInterval)*time.Millisecond)
					client.WriteMsg(uint(0x06), s.VCIChargingHeartbeatReqInit(client))
				}
			}
		}
	}()
	go client.ReadMsg()
	for{

	}
}

func main () {
	sonUdp := NewSonUdp()
	go sonUdp.ReadConsole()
	for {
	}
}