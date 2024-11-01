package shared

// AskArgs are passed to Ask method
type AskArgs struct {
	WorkerName string
}

// AskReply is returned from an Ask method call
type AskReply struct {
	StartTerm uint64
	NumTerms  uint64
	JobID     uint64
}
