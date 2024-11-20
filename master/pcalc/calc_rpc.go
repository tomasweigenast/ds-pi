package pcalc

import (
	"log"

	"ds-pi.com/master/registry"
	"ds-pi.com/master/shared"
)

type CalcRPC struct {
	calc *Calc
	reg  *registry.WorkerRegistry
}

// Ask gives a range of terms to calculate to a worker.
func (s *CalcRPC) Ask(args *shared.AskArgs, reply *shared.AskReply) error {
	log.Printf("Job ask received from worker %q", args.WorkerName)
	job := s.calc.GetJob(args.WorkerName)
	reply.StartTerm = job.FirstTerm
	reply.NumTerms = job.NumTerms
	reply.JobID = job.ID
	return nil
}

// Give returns a calculates range of terms to the master
func (s *CalcRPC) Give(args *shared.GiveArgs, reply *shared.GiveReply) error {
	s.calc.CompleteJob(args.JobID, args.Result, args.Precision)
	return nil
}

func (s *CalcRPC) Connect(args *shared.ConnectArgs, reply *shared.ConnectReply) error {
	log.Printf("Connect request from worker")
	workerName := s.reg.GetWorker(args.WorkerIP)
	reply.WorkerName = workerName
	return nil
}

func (s *CalcRPC) Ping(args *shared.PingArgs, reply *shared.PingResponse) error {
	if s.reg.NotifyPing(args.WorkerName) {
		log.Printf("Ping %s", args.WorkerName)
	}
	return nil
}
