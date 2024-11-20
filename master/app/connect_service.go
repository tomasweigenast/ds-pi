package app

import (
	"log"

	"ds-pi.com/master/shared"
)

type ConnectService struct{}

func (s *ConnectService) Connect(args *shared.ConnectArgs, reply *shared.ConnectReply) error {
	name := a.wr.add_new_worker(args.WorkerIP)
	reply.WorkerName = name
	log.Printf("Connect request received from %s. Given name: %q", args.WorkerIP, name)
	return nil
}
