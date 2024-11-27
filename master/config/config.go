package config

import (
	"flag"
	"log"
)

var (
	TermSize       uint64 = 5_000 // the term size sent to a worker (StartTerm+termSize = LastTerm)
	Reset          bool   = false
	Logs           bool   = true
	ReduceTermSize bool   = true
)

func Load() {
	var termSize int64
	flag.Int64Var(&termSize, "termSize", 5_000, "")
	flag.BoolVar(&Reset, "reset", false, "")
	flag.BoolVar(&Logs, "logs", true, "")
	flag.BoolVar(&ReduceTermSize, "reduceTermSize", true, "")

	flag.Parse()

	if termSize > 0 {
		TermSize = uint64(termSize)
	}

	log.Printf("Using config: TermSize [%d] Reset [%t] Logs [%t] ReduceTermSize [%t]", TermSize, Reset, Logs, ReduceTermSize)
}
