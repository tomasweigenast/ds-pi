package registry

import (
	"net"
	"net/rpc"
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

	PingClient       *rpc.Client
	Available        bool
	UnavailableCount uint // the number of pings that passed and it was reported as unavailable
}

func (w *Worker) IP() net.IP {
	return w.remoteConn.IP
}

func (w *Worker) Name() string {
	return w.name
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

	if _, ok := wr.workers[name]; ok {
		return false
	}

	wr.workers[name] = Worker{
		remoteConn: *addr,
		name:       name,
		Available:  true,
	}
	return true
}

func (wr *WorkerRegistry) RemoveWorker(name string) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	worker, ok := wr.workers[name]
	if ok && worker.PingClient != nil {
		worker.PingClient.Close()
	}

	delete(wr.workers, name)
}

func (wr *WorkerRegistry) GetWorker(addr *net.UDPAddr) *Worker {
	worker, ok := wr.workers[addr.String()]
	if !ok {
		return nil
	}

	return &worker
}

func (wr *WorkerRegistry) GetWorkers() []*Worker {
	wr.mutex.RLock()
	defer wr.mutex.RUnlock()

	workers := make([]*Worker, 0, len(wr.workers))
	for _, w := range wr.workers {
		workers = append(workers, &w)
	}
	return workers
}

func (wr *WorkerRegistry) Clean() {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	for _, w := range wr.workers {
		if w.PingClient != nil {
			w.PingClient.Close()
		}
	}

	clear(wr.workers)
}
