package fsm

import (
	"fmt"
	"net"

	udp "FSMTestingPlatform/Udp"
)

func main() {
	go serverStart(9000)
	go serverStart(9010)
	go serverStart(9020)
	for {

	}
}

func serverStart(port int) {
	server := udp.NewServer(net.IPv4(0, 0, 0, 0), port)
	conn, err := server.BuildUdpServer()
	if err != nil {
		fmt.Printf("server listen is error : %s\n", err.Error())
		return
	}
	go server.HeartBeatCheck()
	go server.ReadMsg(conn)
	for {

	}
}
