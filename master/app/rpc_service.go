package app

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type rpc_service struct {
	listener *net.TCPListener
}

func new_rpc_service(addr *net.TCPAddr) rpc_service {
	rpc.Register(new(JobsService))
	rpc.Register(new(ConnectService))
	rpc.Register(new(PingService))
	rpc.HandleHTTP()

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}

	rpc := rpc_service{
		listener,
	}

	log.Printf("RPC Sevice started at %s", addr.String())
	return rpc
}

func (rpc *rpc_service) start() {
	go http.Serve(rpc.listener, nil)
}

func (rpc *rpc_service) stop() {
	if rpc.listener != nil {
		rpc.listener.Close()
		rpc.listener = nil
	}
}
