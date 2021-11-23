package tcp

import (
	"TestingPlatform/Utils"
	"bufio"
	"fmt"
	"github.com/golang/protobuf/proto"
	"net"
	"os"
	"strings"
	"sync"
)

type Server struct {
	mu sync.Mutex
	Ip string
	Port int
	clients map[string]net.Conn
}

func NewServer (ip string, prot int) *Server {
	return &Server{
		Ip:      ip,
		Port:    prot,
		clients: make(map[string]net.Conn),
	}
}

func (s *Server) Listen () (net.Listener, error) {
	lister , err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		fmt.Println("this server start listen error:", err)
		return nil, err
	}
	return lister, nil
}

func (s *Server) ConnectConn (conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[conn.RemoteAddr().String()] = conn
//	conn.Write(Utils.EnCode("aa", "hello"))
}

func (s *Server) UnConnectConn (conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, conn.RemoteAddr().String())
	conn.Close()
}

func (s *Server) ReadMsg (conn  net.Conn) {
	buff := make([]byte, 1024)
	for{
		n, err := conn.Read(buff)
		if n == 0 {
			fmt.Println("client leave:", conn.RemoteAddr().String())
			s.UnConnectConn(conn)
			break
		}
		if err != nil {
			fmt.Println("client error:", err)
			s.UnConnectConn(conn)
			break
		}

		msgProto , err := Utils.DeCodeByProto(s.Port, buff)
		if err != nil {
			return
		}
		fmt.Printf("The message type %v and message \n", msgProto)
	}
}

func (s *Server) WriteMsg (conn net.Conn, msgTypeCode uint, msg proto.Message) {
	bytes, err := Utils.EnCodeByProto(msgTypeCode, msg)
	if err != nil {
		fmt.Printf("Protoc.Marshal is error : %s\n", err.Error())
		return
	}
	_, err = conn.Write(bytes)
	if err != nil {
		fmt.Printf("server write error : %s", err.Error())
		return
	}
}

func (s *Server) ConnWrite () {
	// 读取命令行输入
	inputReader := bufio.NewReader(os.Stdin)
	for {
		string, err := inputReader.ReadString('\n')

		string = strings.TrimSpace(string)
		if err != nil {
			fmt.Println("client write error :", err)
			break
		}
		for _, v := range s.clients {
			v.Write([]byte(string))
		}
	}
}

func test () {
	server := NewServer("127.0.0.1", 5000)
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