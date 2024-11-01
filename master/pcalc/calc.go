package pcalc

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"sync"
	"time"

	"ds-pi.com/master/config"
)

const filename = "pcalc.state"

// Calc keeps track of calculations, terms and workers available
type Calc struct {
	mutex     *sync.Mutex
	Jobs      map[uint64]WorkerJob // List of jobs per worker. Map key is worker name.
	LastTerm  uint64               // This contain the last term given by the master
	LastJobID uint64               // keeps track of given job ids
}

func NewCalc() *Calc {
	return &Calc{
		mutex: &sync.Mutex{},
		Jobs:  make(map[uint64]WorkerJob),
	}
}

func (c *Calc) GetNewJob(workerName string) WorkerJob {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	startTerm := c.LastTerm
	jobId := c.LastJobID

	job := WorkerJob{
		ID:         jobId,
		SendAt:     time.Now(),
		WorkerName: workerName,
		FirstTerm:  startTerm,
		NumTerms:   config.TermSize,
	}

	c.Jobs[jobId] = job
	c.LastJobID++
	c.LastTerm = job.FirstTerm + config.TermSize

	c.Save()
	return job
}

func (c *Calc) CompleteJob(jobId uint64, result []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	job, ok := c.Jobs[jobId]
	if !ok {
		log.Printf("Job with id [%d] not found.", jobId)
		return
	}

	now := time.Now()
	job.Completed = true
	job.ReturnedAt = &now
	job.Result = result

	c.Jobs[jobId] = job
	c.Save()
}

// WorkerJob contains information about the job sent to a worker
type WorkerJob struct {
	ID         uint64
	SendAt     time.Time
	ReturnedAt *time.Time
	Completed  bool
	WorkerName string
	FirstTerm  uint64
	NumTerms   uint64
	Result     []byte
}

// Save saves the instance to a file. Save do not lock the mutex.
func (c *Calc) Save() {
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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("unable to open Calc file: %s", err)
		}
		return
	}

	defer file.Close()

	// err = gob.NewDecoder(file).Decode(c)
	err = json.NewDecoder(file).Decode(c)
	if err != nil {
		log.Printf("unable to read Calc from bytes: %s", err)
	}

	log.Printf("Calc restored from file. Jobs [%d] Last Term [%d] Last Job Id [%d]", len(c.Jobs), c.LastTerm, c.LastJobID)
	for _, job := range c.Jobs {
		log.Printf("Job %d - Worker [%s] First Term [%d] Term Size [%d] Sent At [%s]", job.ID, job.WorkerName, job.FirstTerm, job.NumTerms, job.SendAt)
	}

	c.Save()
}

func (c *Calc) delete() {
	os.Remove(filename)
}
