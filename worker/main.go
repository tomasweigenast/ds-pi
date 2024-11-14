package main

import (
	"flag"
	"log"
	"net"

	"ds-pi.com/master/shared"
	"ds-pi.com/worker/calculator"
	"ds-pi.com/worker/ping"
)

func main() {
	var workerName string
	var masterIP string

	flag.StringVar(&workerName, "name", "", "")
	flag.StringVar(&masterIP, "ip", "", "")
	flag.Parse()

	if len(workerName) == 0 {
		panic("workerName not specified. Specify with -name=[name]")
	}

	if len(masterIP) == 0 {
		panic("masterIP not specified. Specify with -ip=[ip]")
	}

	ip, err := shared.GetIPv4()
	if err != nil {
		panic(err)
	}

	log.Printf("Worker Name: %s", workerName)
	log.Printf("Using IP: %s", ip.String())
	log.Printf("Master IP: %s", masterIP)

	pingServer := ping.NewPingServer(ip.String(), shared.PING_PORT)
	pingServer.Start()

	// resolve masterIP
	// masterIP := connect.Connect(workerName)

	calculator := calculator.NewCalculator(workerName, net.ParseIP(masterIP), shared.PCALC_PORT)
	calculator.Run()

	defer func() {
		calculator.Stop()
		pingServer.Stop()
	}()

	select {}
}
