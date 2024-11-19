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
	var masterIP string

	flag.StringVar(&masterIP, "ip", "", "")
	flag.Parse()

	if len(masterIP) == 0 {
		panic("masterIP not specified. Specify with -ip=[ip]")
	}

	ip, err := shared.GetIPv4()
	if err != nil {
		panic(err)
	}

	log.Printf("Using IP: %s", ip.String())
	log.Printf("Master IP: %s", masterIP)

	pingServer := ping.NewPingServer(ip.String(), shared.PING_PORT)
	pingServer.Start()

	// resolve masterIP
	// masterIP := connect.Connect(workerName)

	calculator := calculator.NewCalculator(net.ParseIP(masterIP), shared.PCALC_PORT)
	calculator.Run()

	defer func() {
		calculator.Stop()
		pingServer.Stop()
	}()

	select {}
}
