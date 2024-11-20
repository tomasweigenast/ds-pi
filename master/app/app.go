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
	a.run()
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

		case "exit":
			os.Exit(0)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading input:", err)
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
	log.Printf("Ping timer.")
}
