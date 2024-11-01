package shared

// PingArgs is sent to a worker
type PingArgs struct {
	Magic int // A random number just to verify
}

// PingResponse is sent to a master
type PingResponse struct {
	Magic int // The same random number as sent in the request
}
