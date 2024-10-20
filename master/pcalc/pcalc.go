package pcalc

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

// PCalc provides methods to communicate over TCP sending commands
// that are related to the PI Calculation System.
type PCalc struct {
	ip       net.TCPAddr
	listener *net.TCPListener
}

func NewPCalc(ip string, port int) PCalc {
	return PCalc{
		ip: net.TCPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		},
	}
}

func (p *PCalc) Start() {
	service := new(Service)
	rpc.Register(service)
	rpc.HandleHTTP()

	listener, err := net.ListenTCP("tcp", &p.ip)
	if err != nil {
		panic(fmt.Errorf("unable to listen on IP %s for TCP: %s", p.ip.String(), err))
	}

	p.listener = listener
	go http.Serve(p.listener, nil)
	log.Printf("RPC server started at %s", p.ip.String())
}

func (p *PCalc) Stop() {
	if p.listener != nil {
		p.listener.Close()
		p.listener = nil
	}
}
