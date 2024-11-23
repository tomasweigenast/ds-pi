package stats

import "time"

type Stats struct {
	Server *ServerStats `json:"server"`
	PI     *PIStats     `json:"pi"`
}

// ServerStats represents the system and master server statistics
type ServerStats struct {
	Workers []Worker `json:"workers"`
	Memory  MemStats `json:"memory"`
}

type PIStats struct {
	PI           string `json:"pi"`
	DecimalCount int    `json:"pi_decimals"`
}

type Worker struct {
	ID       string    `json:"id"`
	Active   bool      `json:"active"`
	LastPing time.Time `json:"lastPing"`
	LastJob  string    `json:"lastJob"`
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
