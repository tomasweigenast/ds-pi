package app

import (
	"log"
	"net"
	"time"

	"ds-pi.com/master/config"
	"ds-pi.com/master/shared"
	"ds-pi.com/master/stats"
)

type app struct {
	ip         net.TCPAddr
	wr         wregistry
	rpc        rpc_service
	calculator *calculator

	pingTimer *shared.Timer
}

var a app

func Run() {
	myIP, err := shared.GetIPv4()
	if err != nil {
		panic(err)
	}

	addr := net.TCPAddr{
		IP:   myIP,
		Port: shared.MASTER_PORT,
	}
	a = app{
		ip:         addr,
		wr:         new_wregistry(),
		rpc:        new_rpc_service(&addr),
		calculator: new_calculator(),

		pingTimer: shared.NewTimer(10*time.Second, onPingTimerTick),
	}
	a.wr.onConnect = a.calculator.onConnect
	a.run()
}

func Stop() {
	a.stop()
}

func Stats() stats.ServerStats {
	return a.stats()
}

func PIStats() stats.PIStats {
	return a.pi_stats()
}

func PIDecimals() stats.PIStats {
	return stats.PIStats{
		DecimalCount: a.calculator.CurrentDecimalCount(),
		PI:           "3.1415926535897932384626433832795028841971693993751058209749445923078164062862089986280348253421170679",
	}
}

func (a *app) pi_stats() stats.PIStats {
	pi := a.calculator.PI.Text('f', -1)
	if len(pi) < 2 {
		return stats.PIStats{
			PI:           pi,
			DecimalCount: 0,
		}
	}

	return stats.PIStats{
		PI:           pi,
		DecimalCount: len(pi[2:]),
	}
}

func (a *app) stats() stats.ServerStats {
	memStats := shared.GetMemStats(map[string]any{
		"pi":          a.calculator.PI,
		"temp_pi":     a.calculator.tempPI,
		"jobs":        &a.calculator.Jobs,
		"merge_queue": &a.calculator.buffer,
	})
	workers := make([]stats.Worker, 0, len(a.wr.workers))
	jobs := make([]stats.Job, 0, len(a.calculator.Jobs))
	for name, worker := range a.wr.workers {
		workers = append(workers, stats.Worker{
			ID:       name,
			Active:   worker.available,
			LastPing: worker.lastPingTime,
			IP:       worker.ip.String(),
		})
	}
	for _, job := range a.calculator.Jobs {
		jobs = append(jobs, stats.Job{
			ID:         job.ID,
			WorkerID:   job.WorkerName,
			Completed:  job.Completed,
			SentAt:     job.SendAt,
			ReceivedAt: job.ReturnedAt,
			StartTerm:  job.FirstTerm,
		})
	}
	return stats.ServerStats{
		TermSize: config.TermSize,
		Memory:   memStats,
		Workers:  workers,
		Jobs:     jobs,
	}
}

func (a *app) run() {
	a.rpc.start()

	if config.Reset {
		a.calculator.delete_state_file()
	}

	a.calculator.restore()
}

func (a *app) stop() {
	a.pingTimer.Cancel()
	a.rpc.stop()
	a.calculator.stop()
}

func onPingTimerTick() {
	workers := a.wr.list_workers()
	now := time.Now()
	for _, worker := range workers {
		if now.Sub(worker.lastPingTime).Abs() > 10*time.Second {
			log.Printf("Worker %s didnt notify its status in the last 10 seconds, deactivating...", worker.name)
			a.wr.set_inactive(worker.name)
			a.calculator.forget_jobs_of(worker.name)
		}
	}
}
