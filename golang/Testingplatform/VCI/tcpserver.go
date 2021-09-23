package vci

import "TestingPlatform/tcp"

func VCI () {
	server := tcp.NewServer("127.0.0.1", 8000)
	listen, err := server.Listen()
	if err != nil {
		return
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		go server.ConnectConn(conn)
		go server.ReadMsg(conn)
	}
}

