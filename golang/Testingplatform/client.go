package main

import (
	fsm "TestingPlatform/FSM"
)

func main () {
	go vciClient()
	go pmmClient()
	go ohpClient()
	for{

	}
}

func vciClient () {
	thread := fsm.ThreadVCIInit()
	for _, v := range thread.MainThread.AmountList {
		go v.WriteMsg(uint(0x00), thread.VCIMainLoginInit())
	}
	//go thread.ReadMsg()
	go thread.RecvCommand()
	for{

	}
}

func pmmClient () {
	thread := fsm.ThreadPMMInit()
	for _, v := range thread.MainThread.AmountList {
		go v.WriteMsg(uint(0x00), thread.PowerMatrixLoginInit())
		go v.WriteMsg(uint(0x02), thread.ADModuleLoginInit())
	}
	go thread.RecvCommand()
}

func ohpClient () {
	thread := fsm.ThreadOHPInit()
	for _, v := range thread.MainThread.AmountList {
		go v.WriteMsg(uint(0x00), thread.OrderPipelineLoginInit(v))
	}
	go thread.RecvCommand()
}
