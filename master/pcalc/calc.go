package pcalc

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"ds-pi.com/master/config"
)

const filename = "pcalc.state"

// Calc keeps track of calculations, terms and workers available
type Calc struct {
	jobMutex    sync.Mutex
	sumMutex    sync.Mutex
	saveMutex   sync.Mutex
	bufferMutex sync.RWMutex
	JobsBuffer  []mergeRequest
	Jobs        map[uint64]WorkerJob // key is job id
	LastTerm    uint64               // This contain the last term given by the master
	LastJobID   uint64               // keeps track of given job ids
	PI          *big.Float           // The calculated PI number
}

// WorkerJob contains information about the job sent to a worker
type WorkerJob struct {
	ID         uint64
	SendAt     time.Time
	ReturnedAt *time.Time
	Completed  bool   // a flag that indicates the job was completed
	Lost       bool   // a flag that indicates if the connection with the actual worker was lost
	WorkerName string // the name of the worker who owns the job
	FirstTerm  uint64
	NumTerms   uint64
	Result     []byte `json:"-"`
	// ResultString string
}

type mergeRequest struct {
	jobId     uint64
	result    []byte
	precision uint
}

func NewCalc() *Calc {
	return &Calc{
		jobMutex:  sync.Mutex{},
		saveMutex: sync.Mutex{},
		sumMutex:  sync.Mutex{},
		Jobs:      make(map[uint64]WorkerJob),
		PI:        big.NewFloat(0).SetPrec(50_000),
	}
}

func (c *Calc) GetJob(workerName string) WorkerJob {
	c.jobMutex.Lock()
	defer c.jobMutex.Unlock()

	// first check if there is a lost job
	var lostJob *WorkerJob
	for _, job := range c.Jobs {
		if job.Lost {
			lostJob = &job
		}
	}

	var job WorkerJob
	if lostJob != nil {
		lostJob.Lost = false
		lostJob.WorkerName = workerName
		lostJob.SendAt = time.Now()
		c.Jobs[lostJob.ID] = *lostJob
		job = *lostJob
		log.Printf("Gave lost job %d to worker %s", lostJob.ID, workerName)
	} else {
		startTerm := c.LastTerm
		jobId := c.LastJobID

		job = WorkerJob{
			ID:         jobId,
			SendAt:     time.Now(),
			WorkerName: workerName,
			FirstTerm:  startTerm,
			NumTerms:   config.TermSize,
		}

		c.Jobs[job.ID] = job
		c.LastJobID++
		c.LastTerm = job.FirstTerm + config.TermSize
		log.Printf("Gave new job [id=%d] to worker %s [startTerm=%d,endTerm=%d]", job.ID, workerName, job.FirstTerm, job.FirstTerm+config.TermSize)
	}

	go c.Save()
	return job
}

func (c *Calc) CompleteJob(jobId uint64, result []byte, precision uint) {
	c.bufferMutex.Lock()
	defer c.bufferMutex.Unlock()
	c.JobsBuffer = append(c.JobsBuffer, mergeRequest{
		jobId:     jobId,
		result:    result,
		precision: precision,
	})
}

// Save saves the instance to a file. Save do not lock the mutex.
func (c *Calc) Save() {
	c.saveMutex.Lock()
	defer c.saveMutex.Unlock()

	// var buf bytes.Buffer
	// err := gob.NewEncoder(&buf).Encode(c)
	buf, err := json.MarshalIndent(c, "", "   ")

	if err != nil {
		log.Printf("unable to encode Calc to binary: %s", err)
		return
	}

	err = os.WriteFile(filename, buf, os.ModePerm)
	if err != nil {
		log.Printf("unable to save Calc to file: %s", err)
	}
}

// Restore restores the state of Calc from a file, if exists
func (c *Calc) Restore() {
	c.jobMutex.Lock()
	defer c.jobMutex.Unlock()

	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("unable to open Calc file: %s", err)
		}
		return
	}

	defer file.Close()

	err = json.NewDecoder(file).Decode(c)
	if err != nil {
		log.Printf("unable to read Calc from bytes: %s", err)
	}

	log.Printf("Calc restored from file. Jobs [%d] Last Term [%d] Last Job Id [%d]", len(c.Jobs), c.LastTerm, c.LastJobID)
	for jobId, job := range c.Jobs {
		log.Printf("Job %d - Worker [%s] First Term [%d] Term Size [%d] Sent At [%s]", job.ID, job.WorkerName, job.FirstTerm, job.NumTerms, job.SendAt)
		if !job.Completed {
			job.Lost = true
			c.Jobs[jobId] = job
		}
	}

	go c.Save()
}

func (c *Calc) delete() {
	os.Remove(filename)
}

func (c *Calc) merge() {
	c.sumMutex.Lock()
	c.bufferMutex.RLock()
	defer c.sumMutex.Unlock()

	for _, mergeReq := range c.JobsBuffer {
		job, ok := c.Jobs[mergeReq.jobId]
		if !ok {
			log.Printf("Job with id [%d] not found.", mergeReq.jobId)
			return
		}

		now := time.Now()
		job.Completed = true
		job.ReturnedAt = &now
		job.Result = mergeReq.result

		termPi := big.NewFloat(0).SetPrec(mergeReq.precision)
		if err := termPi.GobDecode(job.Result); err != nil {
			log.Printf("invalid big.Float bytes, ignoring result.")
			return
		}
		// job.ResultString = termPi.Text('f', -1)

		tempPI := new(big.Float).SetPrec(c.PI.Prec())
		tempPI.Copy(c.PI)

		for {
			tempPI.Add(tempPI, termPi)
			accuracy := tempPI.Acc()
			decimalCount := len(tempPI.Text('f', -1)[2:])

			log.Printf("Total decimal count: %d. Accuracy: %s", decimalCount, accuracy)

			if accuracy == big.Exact {
				c.PI.Copy(tempPI)
				break
			}

			currentPrecision := tempPI.Prec()
			newPrecision := currentPrecision * 5
			tempPI.SetPrec(newPrecision)
			log.Printf("Increased temporary PI precision to %d bits to meet accuracy requirements. Reached max: %t", newPrecision, newPrecision > big.MaxPrec)

			c.PI.Copy(tempPI)
		}

		c.Jobs[mergeReq.jobId] = job
	}
	c.bufferMutex.Lock()
	log.Printf("Merged %d jobs.", len(c.JobsBuffer))

	c.JobsBuffer = make([]mergeRequest, 0, 10)
	c.bufferMutex.RUnlock()
	c.bufferMutex.Unlock()
	go c.Save()
}
