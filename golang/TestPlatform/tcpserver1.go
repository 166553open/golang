package main

import "goWork/tcp"

func main () {
	server := tcp.NewServer("127.0.0.1", 9000)
	listen, err := server.Listen()
	if err != nil {
		return
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			return
		}
		go server.ConnectConn(conn)
		server.ReadMsg(conn)
	}

}
