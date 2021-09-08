package main

import (
	"goWork/protoc/fsmvci"
	"goWork/tcp"
)

func main () {
	client, err := tcp.NewClient("127.0.0.1:9000")
	if err != nil {
		return
	}
	msg := &fsmvci.VehicleChargingInterfaceLogin{
		VehicleChargingProtoVersion: "1.0",
		VehicleChargingVendor:       "1.0",
		SelfCheckRul:                1,
	}

	client.WriteMsg(0x00, msg)
	go client.ReadMsg()
	for {

	}

}