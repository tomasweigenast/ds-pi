package main

import (
	"log"
	"math"

	app "ds-pi.com/master/app"
	"ds-pi.com/master/config"
	"ds-pi.com/master/dashboard"
)

func main() {
	log.Printf("Max uint allowed by the arch: %d", uint(math.MaxUint))
	config.Load()

	go app.Run()
	go app.Commands()
	go dashboard.Start()

	defer app.Stop()

	// block app
	select {}
}
