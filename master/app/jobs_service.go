package app

import (
	"log"

	"ds-pi.com/master/shared"
)

type JobsService struct{}

func (s *JobsService) Ask(args *shared.AskArgs, reply *shared.AskReply) error {
	log.Printf("Job ask received from worker %q", args.WorkerName)
	job := a.calculator.get_job(args.WorkerName)
	reply.JobID = job.ID
	reply.StartTerm = job.FirstTerm
	reply.NumTerms = job.NumTerms
	return nil
}

func (s *JobsService) Give(args *shared.GiveArgs, reply *shared.GiveReply) error {
	a.calculator.complete_job(args.JobID, args.Result, args.Precision)
	return nil
}
