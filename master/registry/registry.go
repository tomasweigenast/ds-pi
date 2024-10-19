package registry

import (
	"net"
	"sync"
)

// WorkerRegistry keeps track of workers registered in the master
type WorkerRegistry struct {
	mutex   *sync.RWMutex
	workers map[string]Worker
}

type Worker struct {
	remoteConn net.UDPAddr
	name       string
}

func NewWorkerRegistry() WorkerRegistry {
	return WorkerRegistry{
		mutex:   &sync.RWMutex{},
		workers: make(map[string]Worker),
	}
}

func (wr *WorkerRegistry) AddWorker(addr *net.UDPAddr, name string) bool {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	address := addr.String()

	if _, ok := wr.workers[address]; ok {
		return false
	}

	wr.workers[address] = Worker{
		remoteConn: *addr,
		name:       name,
	}
	return true
}

func (wr *WorkerRegistry) RemoveWorker(addr *net.UDPAddr) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	delete(wr.workers, addr.String())
}

func (wr *WorkerRegistry) GetWorker(addr *net.UDPAddr) *Worker {
	worker, ok := wr.workers[addr.String()]
	if !ok {
		return nil
	}

	return &worker
}
