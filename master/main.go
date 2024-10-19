package main

import (
	"ds-pi.com/master/discover"
	"ds-pi.com/master/registry"
)

type master struct {
	discover discover.Discover
	wr       registry.WorkerRegistry
}

func main() {

	master := &master{
		discover: discover.NewDiscover(9933),
		wr:       registry.NewWorkerRegistry(),
	}

	// Start the discovering service
	master.discover.OnDiscover(master.wr.AddWorker)
	master.discover.Begin()

	// block app
	select {}
}
