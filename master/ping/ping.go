package ping

import (
	"fmt"
	"log"
	"net/rpc"
	"time"

	"ds-pi.com/master/registry"
	"ds-pi.com/master/shared"
)

const timerPeriod = 10 * time.Second

type OnPingFunc func(*registry.Worker)

// PingService loops over every registered worker and sends a ping request
type PingService struct {
	wr      *registry.WorkerRegistry
	ticker  *time.Ticker
	stopCh  chan struct{}
	onPing  OnPingFunc
	pinging bool
}

func NewPingService(wr *registry.WorkerRegistry, onPing OnPingFunc) PingService {
	return PingService{wr: wr, onPing: onPing}
}

func (p *PingService) Run() {
	p.ticker = time.NewTicker(timerPeriod)
	go func() {
		for {
			select {
			case <-p.ticker.C:
				if p.pinging {
					return
				}

				p.pinging = true
				p.pingWorkers()
				p.pinging = false
			case <-p.stopCh:
				p.ticker.Stop()
				return
			}
		}
	}()
}

func (p *PingService) Stop() {
	close(p.stopCh)
}

func (p *PingService) pingWorkers() {
	workers := p.wr.GetWorkers()
	for _, w := range workers {
		log.Printf("Pinging worker %q (%s)", w.Name(), w.IP())
		if w.PingClient == nil {
			client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", w.IP(), shared.PING_PORT))
			if err != nil {
				log.Printf("failed to dial tcp to worker %s: %s", w.Name(), err)
				w.Available = false
				continue
			} else {
				w.PingClient = client
			}
		}

		oldState := w.Available

		// do ping
		args := &shared.PingArgs{Magic: 9}
		var reply shared.PingResponse
		err := w.PingClient.Call("PingService.Ping", args, &reply)
		if err == nil && reply.Magic == 9 {
			log.Printf("Response received. Magic [%d]", reply.Magic)
			w.Available = true
			w.UnavailableCount = 0
		} else {
			log.Printf("unable to ping worker %s. Error [%s] Magic [%d]", w.Name(), err, reply.Magic)
			w.Available = false
			w.UnavailableCount++
		}

		if oldState != w.Available {
			p.onPing(w)
		}

	}
}
