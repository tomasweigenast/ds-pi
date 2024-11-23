package app

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
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
	memTimer  *shared.Timer
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
		memTimer:  shared.NewTimer(1*time.Minute, printMemoryUsage),
	}
	a.run()
	printMemoryUsage()
}

func Stop() {
	a.stop()
}

func Commands() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Listening for commands. Type 'exit' to quit.")
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())

		switch command {
		case "pi":
			pi := a.calculator.PI.Text('f', -1)
			if len(pi) < 4 {
				log.Printf("PI not yet available")
				break
			}

			decimalCount := len(pi[2:])
			log.Printf("PI (decimals = %d): %s", decimalCount, pi)
			break

		case "mem":
			printMemoryUsage()
			break

		case "exit":
			os.Exit(0)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading input:", err)
	}
}

func Stats() stats.ServerStats {
	return a.stats()
}

func PIStats() stats.PIStats {
	return a.pi_stats()
}

func (a *app) pi_stats() stats.PIStats {
	pi := a.calculator.PI.Text('f', -1)
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
	for name, worker := range a.wr.workers {
		workers = append(workers, stats.Worker{
			ID:       name,
			Active:   worker.available,
			LastPing: worker.lastPingTime,
			LastJob:  "",
		})
	}
	return stats.ServerStats{
		Memory:  memStats,
		Workers: workers,
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
			log.Printf("Worker %s didnt notify its status in the last 10 seconds, disconnecting...", worker.name)
			a.wr.delete(worker.name)
			a.calculator.forget_jobs_of(worker.name)
		}
	}
}

func printMemoryUsage() {
	shared.PrintMemUsage(map[string]any{
		"pi":          a.calculator.PI,
		"temp_pi":     a.calculator.tempPI,
		"jobs":        &a.calculator.Jobs,
		"merge_queue": &a.calculator.buffer,
	})
}
