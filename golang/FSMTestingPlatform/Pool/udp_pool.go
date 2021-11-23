package pool

import (
	"fmt"
	"net"
	"sync"
)

type UDPpool struct {
	maxConn int				//最大链接数量
	workConn  int			//工作的链接数量
	freeConn  int			//空闲的链接数量
	address  net.UDPAddr	//服务器的链接
	mu sync.Mutex			//互斥锁
	capConn []*net.UDPConn	//UDP链接切片集合
}

// NewUdppool
// 创造udp链接池
func NewUdppool (address net.UDPAddr, size int) *UDPpool {
	capConn := make([]*net.UDPConn, size)
	for i:=0; i<size; i++ {
		addr, err := net.ResolveUDPAddr("udp", address.String())
		if err != nil {
			fmt.Println("client udp addr is error:", err.Error())
			return nil
		}
		conn, err := net.DialUDP("udp", nil, addr)
		capConn = append(capConn, conn)
	}
	return &UDPpool{
		maxConn:	0,
		workConn:	size,
		freeConn:	size,
		address:	address,
		mu:       	sync.Mutex{},
		capConn:	capConn,
	}
}

// Open
// 从链接池中获取udp链接，open操作
func (p *UDPpool) Open () (*net.UDPConn, error) {
	return p.open()
}

// Close
// udp链接使用结束，归还链接池，close操作
func (p *UDPpool) Close(conn *net.UDPConn) error {
	return p.close(conn)
}

func (p *UDPpool) GetMaxConn () int {
	return p.maxConn
}

func (p *UDPpool) GetNumConn () int {
	return p.workConn
}

func (p *UDPpool) GetNumFree () int {
	return p.freeConn
}

// GetAddress
// 获取当前udp链接的address
func (p *UDPpool) GetAddress () net.UDPAddr {
	return p.address
}

func (p *UDPpool) GetAddressStr () string {
	return p.address.String()
}

func (p *UDPpool) open() (*net.UDPConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.freeConn > 0 {
		conn := p.capConn[len(p.capConn)-1]
		p.capConn = p.capConn[:len(p.capConn)-1]
		p.workConn++
		p.freeConn--
		return conn, nil
	}else{
		return p.factory()
		//return nil, errors.New("pool was over")
	}
}

func (p *UDPpool) close(conn *net.UDPConn) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.capConn = append(p.capConn, conn)
	p.workConn--
	p.freeConn++
	return nil
}

// 工厂函数，制造udp链接
func (p *UDPpool) factory() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", p.address.String())
	if err != nil {
		fmt.Println("client udp addr is error:", err.Error())
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil,  addr)
	return conn, err
}
