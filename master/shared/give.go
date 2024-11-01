package shared

type GiveArgs struct {
	JobID  uint64
	Result []byte
}

type GiveReply struct{}
