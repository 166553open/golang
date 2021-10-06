package pool

import (
	"errors"
	"net"
	"sync"
)

/*
* Conn Pool
*	MaxConn  int
*	NumConn  int
*	NumFree  int
*	address  string
*	mu sync.Mutex
*	MaxChan  chan int
*	CapSlice []net.Conn
*
*	Connection pool for network connections
*	The capacity of the connection pool should be entered when it is initialized, and it is recommended not to exceed 10,000.
*	The connection pool can reuse network connections for concurrency.
*	When the network connection exceeds 10,000 goroutines at a time, there may be a queue full error.
*	To protect the safe operation of the code, batch concurrency is recommended.
*
*/

type Pool struct {
	maxConn int
	numConn  int
	numFree  int
	address  string
	mu sync.Mutex
	maxChan  chan int
	capSlice []net.Conn
}

// NewPool
// return connection pool pointer that have address and size(pool cap)
func NewPool (address string, size int) *Pool {
	capSlice := make([]net.Conn, 0)
	channel := make(chan int, size)
	for i:=0; i < size; i++ {
		conn, _ := net.Dial("tcp", address)
		capSlice = append(capSlice, conn)
	}

	return &Pool{
		maxConn:size,
		numConn: 0,
		numFree: size,
		address: address,
		maxChan: channel,
		capSlice: capSlice,
	}
}

func (p *Pool) Open () (net.Conn, error) {
	return p.open()
}

func (p *Pool) Close(conn net.Conn) error {
	return p.close(conn)
}

func (p *Pool) GetMaxConn () int {
	return p.maxConn
}

func (p *Pool) GetNumConn () int {
	return p.numConn
}

func (p *Pool) GetNumFree () int {
	return p.numFree
}

func (p *Pool) GetAddress () string {
	return p.address
}

func (p *Pool) open () (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.numFree > 0 {
		conn := p.capSlice[len(p.capSlice)-1]
		p.capSlice = p.capSlice[:len(p.capSlice)-1]
		p.numConn++
		p.numFree--
		p.maxChan<-1

		return conn, nil
	}else{
		return nil, errors.New("pool was over")
	}
}

func (p *Pool) close (conn net.Conn) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.capSlice = append(p.capSlice, conn)
	<-p.maxChan
	p.numConn--
	p.numFree++
	return nil
}

func (p *Pool) factory() (net.Conn, error) {
	conn, err := net.Dial("tcp", p.address)
	return conn, err
}
