package main

import (
	"log"
	"net"
	"os"
	"strings"

	"ds-pi.com/master/shared"
	"ds-pi.com/worker/calculator"
	"ds-pi.com/worker/ping"
)

func main() {
	var workerName string
	var masterIP net.IP

	args := os.Args
	for _, arg := range args {
		if strings.HasPrefix(arg, "--name:") {
			parts := strings.Split(arg, ":")
			if len(parts) == 2 && len(parts[1]) <= 10 {
				workerName = parts[1]
			}
		}

		if strings.HasPrefix(arg, "--ip:") {
			parts := strings.Split(arg, ":")
			if len(parts) == 2 && len(parts[1]) <= 10 {
				masterIP = net.ParseIP(parts[1])
			}
		}
	}

	if len(workerName) == 0 {
		panic("workerName not specified. Specify with --name:[name]")
	}

	ip, err := shared.GetIPv4()
	if err != nil {
		panic(err)
	}

	log.Printf("Worker Name: %s", workerName)
	log.Printf("Using IP: %s", ip.String())

	pingServer := ping.NewPingServer(ip.String(), shared.PING_PORT)
	pingServer.Start()

	// resolve masterIP
	// masterIP := connect.Connect(workerName)

	calculator := calculator.NewCalculator(workerName, masterIP, shared.PCALC_PORT)
	calculator.Run()

	defer func() {
		calculator.Stop()
		pingServer.Stop()
	}()

	select {}
}
