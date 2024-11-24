package main

import (
	"fmt"
	"io"
	"log"
	"math"

	app "ds-pi.com/master/app"
	"ds-pi.com/master/config"
	"ds-pi.com/master/dashboard"
)

func main() {
	log.Printf("Max uint allowed by the arch: %d", uint(math.MaxUint))
	config.Load()

	if !config.Logs {
		log.SetOutput(io.Discard)
		log.SetFlags(0)
	}

	dashboard.Start()
	app.Run()
	fmt.Println("Running")

	defer app.Stop()

	// block app
	select {}
}
