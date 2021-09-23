package main

import vci "TestingPlatform/VCI"

func main () {
	vciClient()
	for{

	}
}

func vciClient () {
	thread := vci.ThreadVCIInit()
	for _, v := range thread.MainThread.WorkList {
		go v.WriteMsg(uint(0x00), thread.VCIMainLoginInit())
	}
	//go thread.ReadMsg()
	go thread.RecvCommand()
	for{

	}
}

func pmmClient () {
	thread := vci.ThreadPMMInit()
	for _, v := range thread.MainThread.WorkList {
		go v.WriteMsg(uint(0x00), thread.PowerMatrixLoginInit())
		go v.WriteMsg(uint(0x02), thread.ADModuleLoginInit())
	}
	go thread.RecvCommand()
}
