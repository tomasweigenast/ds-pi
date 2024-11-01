package pcalc

import (
	"log"

	"ds-pi.com/master/shared"
)

type CalcRPC struct {
	calc *Calc
}

// Ask gives a range of terms to calculate to a worker.
func (s *CalcRPC) Ask(args *shared.AskArgs, reply *shared.AskReply) error {
	log.Printf("Job ask received from worker %q", args.WorkerName)
	job := s.calc.GetNewJob(args.WorkerName)
	reply.StartTerm = job.FirstTerm
	reply.NumTerms = job.NumTerms
	reply.JobID = job.ID
	return nil
}

// Give returns a calculates range of terms to the master
func (s *CalcRPC) Give(args *shared.GiveArgs, reply *shared.GiveReply) error {
	s.calc.CompleteJob(args.JobID, args.Result)
	return nil
}