package main

import (
	"flag"
	"log"
	"net"

	"ds-pi.com/master/shared"
	"ds-pi.com/worker/calculator"
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

	calculator := calculator.NewCalculator(net.ParseIP(masterIP), shared.MASTER_PORT)
	calculator.Run()

	defer func() {
		calculator.Stop()
	}()

	select {}
}
