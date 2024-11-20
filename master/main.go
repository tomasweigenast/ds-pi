package main

import (
	"log"
	"math"

	app "ds-pi.com/master/app"
	"ds-pi.com/master/config"
)

func main() {
	log.Printf("Max uint allowed by the arch: %d", uint(math.MaxUint))
	config.Load()

	go app.Run()
	go app.Commands()

	defer app.Stop()

	// block app
	select {}
}
