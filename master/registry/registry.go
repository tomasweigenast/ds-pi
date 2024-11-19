package registry

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"

	"ds-pi.com/master/shared"
)

// WorkerRegistry keeps track of workers registered in the master
type WorkerRegistry struct {
	mutex   *sync.RWMutex
	workers map[string]Worker
}

type Worker struct {
	remoteConn net.IP
	name       string

	PingClient       *rpc.Client
	Available        bool
	UnavailableCount uint // the number of pings that passed and it was reported as unavailable
}

func (w *Worker) IP() net.IP {
	return w.remoteConn
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

func (w *WorkerRegistry) GetWorker(ip string) string {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	name := fmt.Sprintf("worker-%s", shared.RandomString())
	w.workers[name] = Worker{
		name:       name,
		remoteConn: net.ParseIP(ip),
		Available:  true,
	}
	log.Printf("Worker %q added at %s", name, ip)

	return name
}

func (w *WorkerRegistry) Delete(name string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	delete(w.workers, name)
	log.Printf("Worker %q deleted", name)
}

func (w *WorkerRegistry) ListWorkers() []*Worker {
	list := make([]*Worker, 0, len(w.workers))
	for _, worker := range w.workers {
		list = append(list, &worker)
	}

	return list
}
