package config

import (
	"flag"
	"log"
)

var (
	TermSize uint64 = 50_000 // the term size sent to a worker (StartTerm+termSize = LastTerm)
	Reset    bool   = false
)

func Load() {
	var termSize int64
	flag.Int64Var(&termSize, "termSize", 50_000, "")
	flag.BoolVar(&Reset, "reset", false, "")

	flag.Parse()

	if termSize > 0 {
		TermSize = uint64(termSize)
	}

	log.Printf("Config loaded. TermSize [%d] Reset [%t]", TermSize, Reset)
}
