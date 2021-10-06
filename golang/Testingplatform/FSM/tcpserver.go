package fsm

import (
	"TestingPlatform/tcp"
	"net"
)

func VCI () {
	server := tcp.NewServer("127.0.0.1", 8000)
	listen, err := server.Listen()
	if err != nil {
		return
	}
	for {
		accept(server, listen)
	}
}

func  accept (server *tcp.Server, listen net.Listener) {
	conn, err := listen.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	go server.ConnectConn(conn)
	go server.ReadMsg(conn)
}
