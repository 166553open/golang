package pool

import (
	"fmt"
	"net"
	"sync"
)

type UDPpool struct {
	maxConn int
	workConn  int
	freeConn  int
	address  net.UDPAddr
	mu sync.Mutex
	maxChan  chan int
	capConn []*net.UDPConn
}

func NewUdppool (address net.UDPAddr, size int) *UDPpool {
	maxChan := make(chan int, size)
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
		maxConn:  size,
		workConn:  size,
		freeConn:  size,
		address:  net.UDPAddr{},
		mu:       sync.Mutex{},
		maxChan:  maxChan,
		capConn: capConn,
	}
}

func (p *UDPpool) Open () (*net.UDPConn, error) {
	return p.open()
}

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
		p.maxChan<-1
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
	<-p.maxChan
	p.workConn--
	p.freeConn++
	return nil
}

func (p *UDPpool) factory() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", p.address.String())
	if err != nil {
		fmt.Println("client udp addr is error:", err.Error())
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil,  addr)
	return conn, err
}
