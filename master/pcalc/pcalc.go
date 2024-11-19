package pcalc

import (
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"

	"ds-pi.com/master/config"
	"ds-pi.com/master/registry"
)

// PCalc provides methods to communicate over TCP sending commands
// that are related to the PI Calculation System.
type PCalc struct {
	ip       net.TCPAddr
	listener *net.TCPListener
	calc     *Calc
	registry *registry.WorkerRegistry
}

func (p *PCalc) GetPI() *big.Float {
	return p.calc.PI
}

func NewPCalc(ip string, port int, wr *registry.WorkerRegistry) PCalc {
	calc := NewCalc()
	if config.Reset {
		calc.delete()
	}

	calc.Restore()

	return PCalc{
		ip: net.TCPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		},
		calc:     calc,
		registry: wr,
	}
}

func (p *PCalc) Start() {
	calcService := new(CalcRPC)
	calcService.calc = p.calc
	calcService.reg = p.registry

	rpc.Register(calcService)
	rpc.HandleHTTP()

	listener, err := net.ListenTCP("tcp", &p.ip)
	if err != nil {
		panic(fmt.Errorf("unable to listen on IP %s for TCP: %s", p.ip.String(), err))
	}

	p.listener = listener
	go http.Serve(p.listener, nil)
	log.Printf("RPC PCalc server started at %s", p.ip.String())
}

func (p *PCalc) Stop() {
	if p.listener != nil {
		p.listener.Close()
		p.listener = nil
	}
}
