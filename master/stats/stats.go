package stats

import "time"

type Stats struct {
	Server *ServerStats `json:"server"`
	PI     *PIStats     `json:"pi"`
}

// ServerStats represents the system and master server statistics
type ServerStats struct {
	TermSize uint64   `json:"termSize"`
	Workers  []Worker `json:"workers"`
	Jobs     []Job    `json:"jobs"`
	Memory   MemStats `json:"memory"`
}

type PIStats struct {
	PI           string `json:"pi"`
	DecimalCount int    `json:"pi_decimals"`
}

type Worker struct {
	ID       string    `json:"id"`
	IP       string    `json:"ip"`
	Active   bool      `json:"active"`
	LastPing time.Time `json:"lastPing"`
}

type Job struct {
	ID         uint64     `json:"id"`
	WorkerID   string     `json:"worker"`
	Completed  bool       `json:"completed"`
	SentAt     time.Time  `json:"sent_at"`
	ReceivedAt *time.Time `json:"received_at"`
	StartTerm  uint64     `json:"start_term"`
}

type MemStats struct {
	Allocated          uint64        `json:"allocated"`
	TotalAlloc         uint64        `json:"totalAlloc"`
	Freed              uint64        `json:"freed"`
	SysMem             uint64        `json:"sysMem"`
	Variables          []VariableMem `json:"variables"`
	TotalVariableAlloc uint64        `json:"totalVariableAlloc"`
}

type VariableMem struct {
	VariableName string `json:"varName"`
	Alloc        uint64 `json:"alloc"`
}
