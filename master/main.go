package main

import (
	"log"
	"math"

	"ds-pi.com/master/config"
	"ds-pi.com/master/discover"
	"ds-pi.com/master/pcalc"
	"ds-pi.com/master/ping"
	"ds-pi.com/master/registry"
	"ds-pi.com/master/shared"
)

type master struct {
	discover discover.Discover
	wr       registry.WorkerRegistry
	pcalc    pcalc.PCalc
	ping     ping.PingService
}

var m *master

func main() {

	log.Printf("Using max uint: %d", uint(math.MaxUint))

	config.Load()

	ip, err := shared.GetIPv4()
	if err != nil {
		panic(err)
	}

	m = &master{
		discover: discover.NewDiscover(shared.DISCOVER_PORT),
		wr:       registry.NewWorkerRegistry(),
		pcalc:    pcalc.NewPCalc(ip.String(), shared.PCALC_PORT),
	}
	m.ping = ping.NewPingService(&m.wr, onPing)

	// Start the discovering service
	m.discover.OnDiscover(m.wr.AddWorker)
	m.discover.Begin()

	// Start the pcalc service
	m.pcalc.Start()

	// Start the ping service
	m.ping.Run()

	defer func() {
		m.ping.Stop()
		m.pcalc.Stop()
		m.discover.Stop()
		m.wr.Clean()
	}()

	// block app
	select {}
}

// onPing only called when the Available status changes, not every time a ping is made
func onPing(w *registry.Worker) {
	if !w.Available && w.UnavailableCount > 30 {
		m.wr.RemoveWorker(w.Name())
	}
}
