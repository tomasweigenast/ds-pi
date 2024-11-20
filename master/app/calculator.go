package app

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
		stopped: false,
	}

	c.mergeTimer = shared.NewTimer(10*time.Second, func() {
		c.merge()
	})

	return c
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

	go c.save()
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

	// held bufferMutex to copy buffer to a temp location first
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

	// toRemove := make(map[int]uint64)
	for _, mergeReq := range buffer {
		job, ok := c.Jobs[mergeReq.jobId]
		if !ok {
			log.Printf("Job with id [%d] not found.", mergeReq.jobId)
			return
		}

		log.Printf("Trying to merge job %d", job.ID)

		termPi := big.NewFloat(0).SetPrec(mergeReq.precision)
		if err := termPi.GobDecode(job.Result); err != nil {
			log.Printf("invalid big.Float bytes, ignoring result.")
			return
		}
		// job.ResultString = termPi.Text('f', -1)

		tempPI := new(big.Float).SetPrec(c.PI.Prec())
		tempPI.Copy(c.PI)

		now := time.Now()
		job.Completed = true
		job.ReturnedAt = &now
		job.Result = mergeReq.result

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

		log.Printf("Total jobs now: %d. Buffer: %d", len(c.Jobs), len(c.buffer))
		c.bufferMutex.Lock()
		c.jobMutex.Lock()
		delete(c.buffer, job.ID)
		delete(c.Jobs, job.ID)
		c.save()
		log.Printf("Job %d merged. Total jobs now: %d. Buffer now is: %d", job.ID, len(c.Jobs), len(c.buffer))
		for _, e := range c.buffer {
			log.Printf("\tJobID=%d", e.jobId)
		}
		c.bufferMutex.Unlock()
		c.jobMutex.Unlock()
	}
	// log.Printf("Merged %d jobs", len(toRemove))

	// // lock buffer for write to delete merged job
	// c.bufferMutex.Lock()
	// c.jobMutex.Lock()
	// defer c.jobMutex.Unlock()
	// defer c.bufferMutex.Unlock()

	// newBuffer := c.buffer[:0]
	// for i, value := range buffer {
	// 	if jobID, found := toRemove[i]; !found {
	// 		newBuffer = append(newBuffer, value)
	// 		delete(c.Jobs, jobID)
	// 		log.Printf("Job %d deleted.", jobID)
	// 	}
	// }
	// c.buffer = newBuffer
	// go c.save()
}

func (c *calculator) stop() {
	c.stopped = true
	c.mergeTimer.Cancel()
	c.save()
}
