package main

import (
	"fmt"
	"goWork/pool"
	"sync"
	"time"
)


func main() {
	address := "127.0.0.1:6000"
	size := 1
	t := pool.NewPool(address, size)
	var wg = sync.WaitGroup{}

	for i := 0; i < 1; i++ {
		wg.Add(size)
		for j := 0; j < size; j ++ {
			go func() {
				defer wg.Done()
				test(t)
			}()
		}
		wg.Wait()
	}

	fmt.Println(time.Now())
	for {}
}
func test(p *pool.Pool) {
	conn, err := p.Open()
	if err != nil {
		fmt.Println("pool open is error:", err)
		return
	}
	a := "test"
	for i := 0; i < 1500; i++ {
		a += "a"
	}
	a += "\n"
	conn.Write([]byte(a))
	p.Close(conn)
}


func addHead (msg string) {
	//br := []byte(msg)
	//msglen := len(br)
	//head := make([]byte, 4)
	//a := 002276
	//
	//head[0] = []byte(002276)
	//
	//bufio.NewReader(msg)
	//bufio.NewReader()
}