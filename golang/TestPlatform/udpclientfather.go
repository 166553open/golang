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

type FatherUdp struct {
	GunAmount int8
	GunFree int8
	GunWork int8
	DefaultInterval uint32
	GunList map[uint32]*udp.Client
	mu sync.Mutex
}



func NewFatherUdp() *FatherUdp {
	return &FatherUdp{
		GunAmount: 6,
		GunFree:   6,
		GunWork:   0,
		DefaultInterval: uint32(200),
		GunList:   make(map[uint32]*udp.Client),
		mu:        sync.Mutex{},
	}
}

func (f *FatherUdp) BuildFatherClient(GunId uint32) *udp.Client {
	if GunId > 6 || GunId < 1 {
		fmt.Printf("GunId is out of range 1, 6 %d\n", GunId)
		return nil
	}
	if _, ok := f.GunList[GunId]; !ok {

		addr := net.UDPAddr{
			IP: net.IPv4(0, 0, 0, 0),
			Port: 9010,
		}

		client, err :=udp.NewClient(addr)
		if err != nil {
			return nil
		}

		f.mu.Lock()
		defer f.mu.Unlock()
		f.GunList[GunId] = client
		return client
	}else{
		fmt.Printf("This Gun is working\n")
		return f.GunList[GunId]
	}
}

func (f *FatherUdp) DeleteFatherClient(GunId uint32) {
	if GunId > 6 || GunId < 1 {
		fmt.Printf("GunId is out of range 1, 6 %d\n", GunId)
		return
	}
	if _, ok := f.GunList[GunId]; ok {
		f.mu.Lock()
		delete(f.GunList, GunId)
		f.mu.Unlock()
	}else{
		fmt.Printf("This Gun is not work\n")
		return
	}
}

func (f *FatherUdp) VCIPluggedHeartbeatReqInit (client *udp.Client) *fsmvci.VCIPluggedHeartbeatReq {
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

func (f *FatherUdp) ReadConsole() {
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
			if strList[0] == "father_build" {
				gunId, _ := strconv.Atoi(strList[1])
				if gunId > 6 {
					gunId = 6
				}
				if gunId < 1 {
					gunId = 1
				}
				go f.BuildRun(uint32(gunId))
			}else if strList[0] == "father_delete" {
				gunId, _ := strconv.Atoi(strList[1])
				if gunId > 6 {
					gunId = 6
				}
				if gunId < 1 {
					gunId = 1
				}
				f.DeleteFatherClient(uint32(gunId))
			}
		}else{
			fmt.Printf("Input is :\n", str)
		}
	}
}

func (f *FatherUdp) BuildRun(GunId uint32) {
	client := f.BuildFatherClient(GunId)
	go func() {
		newTimer := time.NewTimer(time.Duration(f.DefaultInterval)*time.Microsecond)
		for{
			select {
			case <-newTimer.C:
				for _, v := range f.GunList{
					newTimer = time.NewTimer(time.Duration(v.GunInterval)*time.Millisecond)
					client.WriteMsg(uint(0x04), f.VCIPluggedHeartbeatReqInit(client))
				}
			}
		}
	}()
	go client.ReadMsg()
	for{

	}
}

func main () {
	fatherUdp := NewFatherUdp()
	go fatherUdp.ReadConsole()
	for {
	}
}