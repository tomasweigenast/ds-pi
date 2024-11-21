package app

import (
	"encoding/json"
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

type calculator struct {
	jobMutex    sync.Mutex
	sumMutex    sync.Mutex
	saveMutex   sync.Mutex
	bufferMutex sync.RWMutex

	buffer    map[uint64]mergeRequest
	Jobs      map[uint64]*WorkerJob
	LastTerm  uint64
	LastJobID uint64
	PI        *big.Float
	tempPI    *big.Float

	mergeTimer *shared.Timer
	stopped    bool
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
	Result     []byte `json:"-"`
}

type mergeRequest struct {
	jobId     uint64
	result    []byte
	precision uint
}

func new_calculator() *calculator {
	c := &calculator{
		buffer:  make(map[uint64]mergeRequest),
		Jobs:    make(map[uint64]*WorkerJob),
		PI:      big.NewFloat(0).SetPrec(50_000),
		tempPI:  big.NewFloat(0).SetPrec(50_000),
		stopped: false,
	}

	c.mergeTimer = shared.NewTimer(10*time.Second, func() {
		c.merge()
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
			NumTerms:   config.TermSize,
		}

		c.Jobs[job.ID] = job
		c.LastJobID++
		c.LastTerm = job.FirstTerm + config.TermSize
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
	c.saveMutex.Lock()
	defer c.saveMutex.Unlock()

	buf, err := json.MarshalIndent(c, "", "   ")
	if err != nil {
		log.Printf("unable to encode calculator state: %s", err)
		return
	}

	err = os.WriteFile(filename, buf, os.ModePerm)
	if err != nil {
		log.Printf("unable to save calculator state to file: %s", err)
	}
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

	err = json.NewDecoder(file).Decode(c)
	if err != nil {
		log.Printf("unable to read calculator state from bytes: %s", err)
	}

	for _, job := range c.Jobs {
		log.Printf("Job [%d] Worker [%s] First Term [%d] Term Size [%d] Sent At [%s]", job.ID, job.WorkerName, job.FirstTerm, job.NumTerms, job.SendAt)
		if !job.Completed {
			job.Lost = true
		}
	}

	log.Printf("PI number: %s", c.PI.Text('f', -1))

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
	log.Printf("Merged %d jobs in %s.", len(buffer), time.Since(start))

	c.bufferMutex.Lock()
	c.jobMutex.Lock()
	defer func() {
		c.jobMutex.Unlock()
		c.bufferMutex.Unlock()
	}()

	log.Printf("Total jobs in buffer before delete: %d", len(c.buffer))
	now := time.Now()
	for _, mergeReq := range buffer {
		if job, ok := c.Jobs[mergeReq.jobId]; ok {
			job.Completed = true
			job.ReturnedAt = &now
			delete(c.buffer, job.ID)
		}
	}
	log.Printf("Total jobs in buffer now: %d", len(c.buffer))

	c.save()
}

func (c *calculator) stop() {
	c.stopped = true
	c.mergeTimer.Cancel()
	c.save()
}
