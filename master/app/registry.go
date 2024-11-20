package app

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"ds-pi.com/master/shared"
)

// wregistry contains the list of connected workers
type wregistry struct {
	mutex   sync.RWMutex // a mutex to avoid race conditions when adding/removing workers
	workers map[string]*worker
}

type worker struct {
	ip           net.IP    // worker ip
	name         string    // worker name, assigned by the master
	available    bool      // a flag that indicates if the worker is available or not
	lastPingTime time.Time // the last time we received a ping from the worker
}

func new_wregistry() wregistry {
	return wregistry{
		mutex:   sync.RWMutex{},
		workers: make(map[string]*worker),
	}
}

func (r *wregistry) add_new_worker(ip string) string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := fmt.Sprintf("worker-%s", shared.RandomString())
	r.workers[name] = &worker{
		name:         name,
		ip:           net.ParseIP(ip),
		available:    true,
		lastPingTime: time.Now(),
	}
	log.Printf("Added worker %q at %s", name, ip)
	return name
}

func (r *wregistry) delete(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.workers, name)
	log.Printf("Worker %q deleted", name)
}

func (r *wregistry) list_workers() []worker {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	list := make([]worker, 0, len(r.workers))
	for _, worker := range r.workers {
		list = append(list, *worker)
	}
	return list
}

func (r *wregistry) notify_ping(workerName string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	worker, ok := r.workers[workerName]
	if !ok {
		return false
	}

	worker.lastPingTime = time.Now()
	return true
}
