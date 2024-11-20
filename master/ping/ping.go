package ping

import (
	"time"

	"ds-pi.com/master/registry"
)

const timerPeriod = 10 * time.Second

type PingService struct {
	wr      *registry.WorkerRegistry
	ticker  *time.Ticker
	stopCh  chan struct{}
	pinging bool
}

func NewPingService(wr *registry.WorkerRegistry) PingService {
	return PingService{wr: wr}
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
				p.checkPings()
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

func (p *PingService) checkPings() {
	workers := p.wr.ListWorkers()
	now := time.Now()
	for _, w := range workers {
		if now.Sub(w.LastPingTime).Abs() > 1*time.Minute {
			p.wr.Delete(w.Name())
		}
	}
}
