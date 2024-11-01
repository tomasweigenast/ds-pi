package ping

import (
	"log"

	"ds-pi.com/master/shared"
)

type PingService struct{}

func (s *PingService) Ping(args *shared.PingArgs, reply *shared.PingResponse) error {
	log.Printf("Received ping request. Magic [%d]", args.Magic)
	reply.Magic = args.Magic
	return nil
}
