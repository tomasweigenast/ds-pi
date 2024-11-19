package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"ds-pi.com/master/config"
	"ds-pi.com/master/pcalc"
	"ds-pi.com/master/ping"
	"ds-pi.com/master/registry"
	"ds-pi.com/master/shared"
)

type master struct {
	wr    registry.WorkerRegistry
	pcalc pcalc.PCalc
	ping  ping.PingService
}

var m *master

func main() {

	log.Printf("Using max uint: %d", uint(math.MaxUint))

	config.Load()

	ip, err := shared.GetIPv4()
	if err != nil {
		panic(err)
	}

	wr := registry.NewWorkerRegistry()
	m = &master{
		wr:    wr,
		pcalc: pcalc.NewPCalc(ip.String(), shared.PCALC_PORT, &wr),
	}

	// Start the pcalc service
	m.pcalc.Start()

	// Start the ping service
	m.ping = ping.NewPingService(&m.wr)
	m.ping.Run()

	defer func() {
		m.ping.Stop()
		m.pcalc.Stop()
	}()

	go commands()

	// block app
	select {}
}

func commands() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Listening for commands. Type 'exit' to quit.")
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())

		switch command {
		case "pi":
			pi := m.pcalc.GetPI().Text('f', -1)
			decimalCount := len(pi[2:])
			fmt.Printf("PI (decimals = %d): %s", decimalCount, pi)
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
