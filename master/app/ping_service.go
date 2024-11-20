package app

import (
	"log"

	"ds-pi.com/master/shared"
)

type PingService struct{}

func (s *PingService) Ping(args *shared.PingArgs, reply *shared.PingReply) error {
	a.wr.notify_ping(args.WorkerName)
	log.Printf("Ping received from %s", args.WorkerName)
	return nil
}
