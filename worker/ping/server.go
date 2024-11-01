package ping

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type PingServer struct {
	ip       *net.TCPAddr
	listener *net.TCPListener
}

func NewPingServer(ip string, port int) PingServer {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		panic(err)
	}
	return PingServer{
		ip: addr,
	}
}

func (p *PingServer) Start() {
	service := new(PingService)
	rpc.Register(service)
	rpc.HandleHTTP()

	listener, err := net.ListenTCP("tcp", p.ip)
	if err != nil {
		panic(fmt.Errorf("unable to listen on IP %s for TCP: %s", p.ip.String(), err))
	}

	p.listener = listener
	go http.Serve(p.listener, nil)
	log.Printf("RPC ping server started at %s", p.ip.String())
}

func (p *PingServer) Stop() {
	if p.listener != nil {
		p.listener.Close()
		p.listener = nil
	}
}
