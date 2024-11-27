package app

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math/big"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"ds-pi.com/master/config"
	"ds-pi.com/master/shared"
)

const filename = "calc.state"

var currentDecimalCount = 0

type calculator struct {
	jobMutex    sync.Mutex
	sumMutex    sync.Mutex
	saveMutex   sync.Mutex
	bufferMutex sync.RWMutex

	buffer    map[uint64]mergeRequest
	Jobs      map[uint64]*WorkerJob
	LastTerm  uint64
	TermSize  uint64
	LastJobID uint64
	PI        *big.Float
	tempPI    *big.Float

	mergeTimer *shared.Timer
	stopped    bool
}

type saveObj struct {
	Jobs      map[uint64]WorkerJob
	LastTerm  uint64
	LastJobID uint64
	TermSize  uint64
	PIPrec    uint
	PI        string
}

type WorkerJob struct {
	ID         uint64
	SendAt     time.Time
	ReturnedAt *time.Time
	Completed  bool   // a flag that indicates the job was completed
	Lost       bool   // a flag that indicates if the connection with the actual worker was lost
	WorkerName string // the name of the worker who owns the job
	FirstTerm  uint64
	NumTerms   uint64
	Result     []byte
	ResultPrec uint
}

type mergeRequest struct {
	jobId     uint64
	result    []byte
	precision uint
}

func new_calculator() *calculator {
	c := &calculator{
		buffer:   make(map[uint64]mergeRequest),
		Jobs:     make(map[uint64]*WorkerJob),
		PI:       big.NewFloat(0).SetPrec(50_000),
		TermSize: config.TermSize,
		tempPI:   big.NewFloat(0).SetPrec(50_000),
		stopped:  false,
	}

	c.mergeTimer = shared.NewTimer(10*time.Second, func() {
		c.merge()
	})

	shared.NewTimer(1*time.Minute, func() {
		log.Printf("Calculating decimals of PI...")
		currentDecimalCount = countDecimals(c.PI)
		log.Printf("New decimal count is: %d", currentDecimalCount)
	})

	return c
}

func (c *calculator) forget_jobs_of(workerName string) {
	c.jobMutex.Lock()
	defer c.jobMutex.Unlock()

	for _, job := range c.Jobs {
		if !job.Completed && job.WorkerName == workerName {
			job.Lost = true
			log.Printf("Job %d of %s marked as lost.", job.ID, workerName)
		}
	}

	c.save()
}

func (c *calculator) get_job(workerName string) WorkerJob {
	if c.stopped {
		return WorkerJob{}
	}

	c.jobMutex.Lock()
	defer c.jobMutex.Unlock()

	var job *WorkerJob
	for _, j := range c.Jobs {
		if j.Lost {
			job = j
		}
	}

	if job != nil {
		job.Lost = false
		job.WorkerName = workerName
		job.SendAt = time.Now()
		log.Printf("Gave lost job %d to worker %s", job.ID, workerName)
	} else {
		startTerm := c.LastTerm
		jobId := c.LastJobID

		job = &WorkerJob{
			ID:         jobId,
			SendAt:     time.Now(),
			WorkerName: workerName,
			FirstTerm:  startTerm,
			NumTerms:   c.TermSize,
		}

		c.Jobs[job.ID] = job
		c.LastJobID++
		c.LastTerm = job.FirstTerm + c.TermSize
		log.Printf("Gave new job [id=%d] to worker %s", job.ID, workerName)
	}

	go c.save()
	return *job
}

func (c *calculator) complete_job(jobId uint64, result []byte, precision uint) {
	if c.stopped {
		return
	}
	c.bufferMutex.Lock()
	defer c.bufferMutex.Unlock()

	c.buffer[jobId] = mergeRequest{
		jobId:     jobId,
		result:    result,
		precision: precision,
	}
	log.Printf("Job %d added to merge buffer. Total jobs to merge: %d", jobId, len(c.buffer))

	if len(c.buffer) > 5 {
		go c.merge()
	}
}

func (c *calculator) save() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from a panic trying to save file")
			}
		}()

		c.saveMutex.Lock()
		defer c.saveMutex.Unlock()

		log.Printf("Saving state...")

		buf := bytes.Buffer{}
		obj := saveObj{
			LastTerm:  c.LastTerm,
			LastJobID: c.LastJobID,
			TermSize:  c.TermSize,
			Jobs:      make(map[uint64]WorkerJob, len(c.Jobs)),
			PIPrec:    c.PI.Prec(),
			PI:        c.PI.Text('f', -1),
		}
		for jobId, job := range c.Jobs {
			obj.Jobs[jobId] = *job
		}
		err := gob.NewEncoder(&buf).Encode(obj)
		if err != nil {
			log.Printf("Unable to encode calculator state: %s", err)
			return
		}

		log.Printf("State encoded, saving...")
		err = os.WriteFile(filename, buf.Bytes(), os.ModePerm)
		if err != nil {
			log.Printf("unable to save calculator state to file: %s", err)
		}
		log.Printf("State saved.")
	}()
}

func (c *calculator) restore() {
	c.jobMutex.Lock()
	defer c.jobMutex.Unlock()

	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("unable to open calculator state file: %s", err)
		}
		return
	}

	defer file.Close()

	obj := saveObj{}
	err = gob.NewDecoder(file).Decode(&obj)
	if err != nil {
		log.Printf("unable to read calculator state from bytes: %s", err)
	}

	c.LastJobID = obj.LastJobID
	c.LastTerm = obj.LastTerm
	c.TermSize = obj.TermSize
	for jobId, job := range obj.Jobs {
		c.Jobs[jobId] = &WorkerJob{
			ID:         job.ID,
			SendAt:     job.SendAt,
			ReturnedAt: job.ReturnedAt,
			Completed:  job.Completed,
			Lost:       job.Lost,
			WorkerName: job.WorkerName,
			FirstTerm:  job.FirstTerm,
			NumTerms:   job.NumTerms,
			Result:     job.Result,
		}
	}

	for _, job := range c.Jobs {
		log.Printf("Job [%d] Worker [%s] First Term [%d] Term Size [%d] Sent At [%s]", job.ID, job.WorkerName, job.FirstTerm, job.NumTerms, job.SendAt)
		if !job.Completed {
			job.Lost = true
		}
	}

	c.PI.SetPrec(obj.PIPrec)
	_, ok := c.PI.SetString(obj.PI)
	if !ok {
		log.Printf("Unable to restore PI")
		return
	}

	// merge all jobs into PI
	// c.PI.SetPrec(5000000)

	// mergeableJobs := 0
	// mergedJobs := 0
	// for _, job := range c.Jobs {
	// 	if job.Completed && job.Result != nil && len(job.Result) > 0 {
	// 		mergeableJobs++
	// 	}
	// }

	// for _, job := range c.Jobs {
	// 	if job.Completed && job.Result != nil && len(job.Result) > 0 {
	// 		pi := big.NewFloat(0).SetPrec(job.ResultPrec)
	// 		if err := pi.GobDecode(job.Result); err != nil {
	// 			log.Printf("Invalid big.Float bytes, ignoring marking the job [%d] as lost", job.ID)
	// 			job.Lost = true
	// 			job.Completed = false
	// 			continue
	// 		}

	// 		count := 0
	// 		log.Printf("Going to merge job %d [%s]", job.ID, pi.Text('g', -1))
	// 		for count < 6 {
	// 			c.PI.Add(c.PI, pi)

	// 			if pi.Acc() == big.Exact {
	// 				break
	// 			}

	// 			log.Printf("Unable to sum job [%d] without losing precision, trying to increment precision and trying again", job.ID)
	// 			count++
	// 		}
	// 		mergedJobs++
	// 		log.Printf("Job %d merged. %d of %d", job.ID, mergedJobs, mergeableJobs)
	// 	}
	// }

	// log.Printf("Decimals of PI calculated: %d", countDecimals(c.PI))

	c.save()
}

func (c *calculator) delete_state_file() {
	os.Remove(filename)
	log.Printf("State file deleted")
}

func (c *calculator) merge() {
	if len(c.buffer) == 0 {
		return
	}

	if !c.sumMutex.TryLock() {
		log.Printf("Ignoring this merge request because there is another one in process.")
		return
	}

	defer c.sumMutex.Unlock()

	// Hold bufferMutex to copy buffer to a temp location first
	c.bufferMutex.RLock()
	buffer := make([]mergeRequest, 0, len(c.buffer))
	for _, req := range c.buffer {
		buffer = append(buffer, mergeRequest{
			jobId:     req.jobId,
			result:    req.result,
			precision: req.precision,
		})
	}
	c.bufferMutex.RUnlock()

	slices.SortFunc(buffer, func(a, b mergeRequest) int {
		if a.jobId > b.jobId {
			return 1
		}

		return -1
	})

	log.Printf("Going to merge %d jobs (%s)", len(buffer), strings.Join(shared.MapArray(buffer, func(mr mergeRequest) string { return fmt.Sprint(mr.jobId) }), ", "))
	start := time.Now()

	// reset tempPI to the current value of PI
	c.tempPI.Copy(c.PI)

	// Accumulate terms
	batchSum := new(big.Float).SetPrec(c.tempPI.Prec())
	for _, mergeReq := range buffer {
		termPi := big.NewFloat(0).SetPrec(mergeReq.precision)
		if err := termPi.GobDecode(mergeReq.result); err != nil {
			log.Printf("Invalid big.Float bytes, ignoring result")
			c.jobMutex.Lock()
			c.Jobs[mergeReq.jobId].Lost = true
			c.jobMutex.Unlock()
			continue
		}

		batchSum.Add(batchSum, termPi)
	}

	// Sum total
	for {
		c.tempPI.Add(c.tempPI, batchSum)

		// Check accuracy and decimal count
		accuracy := c.tempPI.Acc()
		log.Printf("Accuracy: %s", accuracy)

		if accuracy == big.Exact {
			break
		}

		// Increase precision to improve accuracy
		currentPrecision := c.tempPI.Prec()
		newPrecision := currentPrecision * 2
		if newPrecision > big.MaxPrec {
			log.Fatalf("Reached maximum precision limit of big.Float. Theorical limit.")
			break
		}

		c.tempPI.SetPrec(newPrecision)
		batchSum.SetPrec(newPrecision) // Ensure batchSum matches the new precision
		log.Printf("Increased tempPI precision to %d bits. Doing addition again.", newPrecision)
	}

	// Copy result
	log.Printf("Copying temp into PI...")
	c.PI.Copy(c.tempPI)

	log.Printf("Copied!")
	// log.Printf("Copied! New decimals are: %d", countDecimals(c.PI))

	log.Printf("Merged %d jobs in %s.", len(buffer), time.Since(start))

	c.bufferMutex.Lock()
	c.jobMutex.Lock()
	defer func() {
		c.jobMutex.Unlock()
		c.bufferMutex.Unlock()
	}()

	log.Printf("Total jobs in buffer before delete: %d", len(c.buffer))
	now := time.Now()
	var lastJob WorkerJob
	for i, mergeReq := range buffer {
		if job, ok := c.Jobs[mergeReq.jobId]; ok {
			job.Completed = true
			job.ReturnedAt = &now
			job.Result = mergeReq.result
			job.ResultPrec = mergeReq.precision
			delete(c.buffer, job.ID)

			if i == len(buffer)-1 {
				lastJob = *job
			}
		}
	}
	log.Printf("Total jobs in buffer now: %d", len(c.buffer))

	if time.Since(lastJob.SendAt) > 10*time.Second && c.TermSize > 10 {
		c.TermSize = c.TermSize - (c.TermSize / 10)
		if c.TermSize <= 10 {
			c.TermSize = 10
		}
		log.Printf("TermSize reduced to %d", c.TermSize)
	}

	c.save()
}

func (c *calculator) stop() {
	c.stopped = true
	c.mergeTimer.Cancel()
	c.save()
}

func (c *calculator) onConnect(worker worker) {
	c.jobMutex.Lock()
	defer c.jobMutex.Unlock()

	for _, job := range c.Jobs {
		// this indicates a job was lost, so forget jobs
		if !job.Completed && job.WorkerName == worker.name {
			job.Lost = true
			log.Printf("Job %d of %s marked as lost.", job.ID, job.WorkerName)
		}
	}

	c.save()
}

func (a *calculator) CurrentDecimalCount() int {
	if currentDecimalCount == 0 {
		currentDecimalCount = countDecimals(a.PI)
	}

	return currentDecimalCount
}

func countDecimals(x *big.Float) int {
	// Prepare a threshold to determine integer status.
	// threshold := new(big.Float).SetPrec(x.Prec()).SetInt64(1)
	zero := new(big.Float).SetInt64(0)

	// Create a copy of x to avoid modifying the original.
	temp := new(big.Float).Copy(x)
	decimals := 0

	for temp.Cmp(zero) != 0 {
		_, acc := temp.Int(nil) // Check if the value is an integer.
		if acc == big.Exact {
			break
		}
		temp.Mul(temp, big.NewFloat(10)) // Multiply by 10.
		decimals++
	}

	return decimals
}
