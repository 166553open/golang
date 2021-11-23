package main

import (
	conf "FSMTestingPlatform/Conf"
	fsm "FSMTestingPlatform/FSM"
	"FSMTestingPlatform/Utils"
)

func main() {
	go vciClient()
	go pmmClient()
	go ohpClient()
	conf.GinEngine.Run(":10001")
	for {

	}
}

// VCI客户端
func vciClient() {
	thread := fsm.ThreadVCIInit()
	for _, v := range thread.MainThread.AmountList {
		go v.WriteMsg(uint(0x00), thread.VCIMainLoginInit(1))
	}
	go thread.RTStart()
	go thread.RecvCommand(conf.GinEngine)
}

// PMM客户端
func pmmClient() {
	thread := fsm.ThreadPMMInit()
	for _, v := range thread.MainThread.AmountList {
		go v.WriteMsg(uint(0x00), thread.PowerMatrixLoginInit(1))
		go v.WriteMsg(uint(0x02), thread.ADModuleLoginInit())
	}
	go thread.RtStart()
	go thread.RecvCommand(conf.GinEngine)
}

// OHP客户端
func ohpClient() {
	thread := fsm.ThreadOHPInit()
	for _, v := range thread.MainThread.AmountList {
		go v.WriteMsg(uint(0x00), thread.OHPOrderPipelineLogin(Utils.RandValue(4)))
	}
	go thread.RtStart()
	go thread.RecvCommand(conf.GinEngine)
}
