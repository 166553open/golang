package tcp

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"goWork/Utils"
	"net"
)

type client struct {
	conn net.Conn
}

func NewClient (addr string) (*client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("Client dial error : %s \n", err.Error())
		return nil, err
	}
	return &client{conn: conn}, nil
}

func (c *client) ReadMsg () {
	buff := make([]byte, 1024)
	for{
		n, err := c.conn.Read(buff)
		if n == 0 {
			fmt.Println("client leave:", c.conn.RemoteAddr().String())
			c.conn.Close()
			break
		}
		if err != nil {
			fmt.Println("client error:", err)
			c.conn.Close()
			break
		}
		msgProto, err := Utils.DeCodeByProto(buff)
		if err != nil {
			return
		}
		fmt.Printf("The message type %v and message\n", msgProto)
	}
}

func (c *client) WriteMsg (msgType uint, msg proto.Message) {
	bytes, err := Utils.EnCodeByProto(msgType, msg)
	if err != nil {
		return
	}
	fmt.Println(bytes, msgType, msg)
	c.conn.Write(bytes)
}

/*
type message struct {
	client1 chan string
	client2 chan string
}
var msg = message{
	client1: make(chan string),
	client2: make(chan string),
}

func main () {
	pool1 := pool.NewPool("127.0.0.1:5000", 10)
	pool2 := pool.NewPool("127.0.0.1:6000", 10)

	conn1, err := pool1.Open()

	if err != nil {
		fmt.Printf("conn1 is error :%s", err.Error())
		return
	}
	conn2, err := pool2.Open()
	if err != nil {
		fmt.Printf("conn1 is error :%s", err.Error())
		return
	}

	go server(conn1, msg.client1)
	go server(conn2, msg.client2)
	go clientGetContext(msg)
	for {

	}
}

func server (conn net.Conn, channel chan string) {
	defer conn.Close()
	go read(conn)
	go Write(conn, channel)
	for {

	}
}

func read(conn net.Conn) {
	buff := make([]byte, 1024)

	for {
		n, err := conn.Read(buff)
		if n == 0 {
			fmt.Println("server close:", conn.RemoteAddr().String())
			break
		}
		if err != nil {
			fmt.Println("server error:", conn.RemoteAddr().String())
			break
		}
		fmt.Println(string(buff[:n]))
		conn.Write(buff[:n])
	}
}

func Write (conn net.Conn, channel chan string) {
	for {
		select{
		case x := <-channel:
			conn.Write([]byte(x))
		}
	}
}

func clientGetContext (msg1 message) {
	// 读取命令行输入
	inputReader := bufio.NewReader(os.Stdin)
	for {
		string, err := inputReader.ReadString('\n')
		if err != nil {
			fmt.Println("client write error :", err)
			break
		}
		//1
		str := strings.TrimSpace(string)
		if strings.Contains(str, "?:") {
			msgSlice := strings.Split(str, "?:")
			if msgSlice[0] == "server1" {
				msg1.client1 <- msgSlice[1]
			}else{
				msg1.client2 <- msgSlice[1]
			}
		}else{
			msg1.client2 <- str
		}
	}
}
*/