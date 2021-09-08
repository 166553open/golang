package main

import (
	"bufio"
	"fmt"
	"goWork/Utils"
	"goWork/protoc/fsmvci"
	"goWork/udp"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MainUdp struct {
	mu sync.Mutex
	GunAmount int8
	GunFree int8
	GunWork int8
	GunList map[uint32]string
}

func NewMainUdp () *MainUdp {
	return &MainUdp{
		GunAmount: 6,
		GunFree:   6,
		GunWork:   0,
		GunList:   make(map[uint32]string),
	}
}

func (m *MainUdp) BuildMainClient(GunId uint32) {
	if m.GunFree == 0 || len(m.GunList) == 6 {
		return
	}else{
		m.mu.Lock()
		defer m.mu.Unlock()
		m.GunList[GunId] = "1"
		m.GunFree = m.GunFree - 1
		m.GunWork = m.GunWork + 1
	}
}

func (m *MainUdp) DeleteMainClient(GunId uint32) {
	if m.GunFree == 6 || len(m.GunList) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.GunList, GunId)
	m.GunFree = m.GunFree + 1
	m.GunWork = m.GunWork - 1
}

func (m *MainUdp) VCIMainLoginInit(client *udp.Client) *fsmvci.VehicleChargingInterfaceLogin {
	return &fsmvci.VehicleChargingInterfaceLogin{
		VehicleChargingProtoVersion: Utils.ConfProtoc["VehicleChargingProtoVersion"].(string),
		VehicleChargingVendor:       Utils.ConfProtoc["VehicleChargingProtoVersion"].(string),
		SelfCheckRul:                1,
	}
}

func (m *MainUdp) VCImainHeartInit (client *udp.Client) *fsmvci.VCImainHeartbeatReq {
	gunBaseList := make([]*fsmvci.GunBaseStatus, 0)
	for k, _ := range m.GunList {
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

func (m *MainUdp) ReadConsole() {
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
			if strList[0] == "main_build" {
				gunId, _ := strconv.Atoi(strList[1])
				if gunId > 6 {
					gunId = 6
				}
				if gunId < 1 {
					gunId = 1
				}
				m.BuildMainClient(uint32(gunId))
			}else if strList[0] == "main_delete" {
				gunId, _ := strconv.Atoi(strList[1])
				if gunId > 6 {
					gunId = 6
				}
				if gunId < 1 {
					gunId = 1
				}
				m.DeleteMainClient(uint32(gunId))
			}
		}else{
			fmt.Printf("Input is :\n", str)
		}

		if str == "" {

		}
		if str == "" {

		}

	}
}

func main () {
	addr := net.UDPAddr{
		IP: net.IPv4(0, 0, 0, 0),
		Port: 9000,
	}
	client, err :=udp.NewClient(addr)
	if err != nil {
		return
	}
	mainConnect := NewMainUdp()
	go mainConnect.ReadConsole()
	go client.WriteMsg(uint(0x00), mainConnect.VCIMainLoginInit(client))
	go func() {
		newTimer := time.NewTimer(time.Duration(client.GunInterval)*time.Microsecond)
		for{
			select {
			case <-newTimer.C:
				newTimer = time.NewTimer(time.Duration(client.GunInterval)*time.Millisecond)
				vCImainHeart := mainConnect.VCImainHeartInit(client)
				client.WriteMsg(uint(0x02), vCImainHeart)
			}
		}
	}()
	go client.ReadMsg()
	for {
	}
}