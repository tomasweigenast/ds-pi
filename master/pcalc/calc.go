package pcalc

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
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
	PI        *big.Float           // The calculated PI number
	Counter   *atomic.Uint32
}

func NewCalc() *Calc {
	return &Calc{
		mutex: &sync.Mutex{},
		Jobs:  make(map[uint64]WorkerJob),
		// PI:    big.NewFloat(0).SetPrec(math.MaxUint),
		PI:      big.NewFloat(0).SetPrec(50_000),
		Counter: &atomic.Uint32{},
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

func (c *Calc) CompleteJob(jobId uint64, result []byte, precision uint) {
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

	termPi := big.NewFloat(0).SetPrec(precision)
	if err := termPi.GobDecode(job.Result); err != nil {
		log.Printf("invalid big.Float bytes, ignoring result.")
		return
	}

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

	job.Merged = true
	c.Jobs[jobId] = job
	c.Save()

}

// WorkerJob contains information about the job sent to a worker
type WorkerJob struct {
	ID         uint64
	SendAt     time.Time
	ReturnedAt *time.Time
	Completed  bool
	Merged     bool // indicates if Result has been merged in total result
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

// func (u Calc) MarshalJSON() ([]byte, error) {
// 	m := make(map[string]any)
// 	m["jobs"] = u.Jobs
// 	m["lastTerm"] = u.LastTerm
// 	m["lastJobId"] = u.LastJobID

// 	pi, err := u.PI.GobEncode()
// 	if err != nil {
// 		panic(fmt.Errorf("unable to encode pi: %s", err))
// 	}

// 	m["pi"] = pi
// 	m["rawPi"] = u.PI.String()
// 	return json.Marshal(m)
// }

// func (u *Calc) UnmarshalJSON(data []byte) error {
// 	m := struct {
// 		jobs      map[uint64]WorkerJob
// 		lastTerm  uint64
// 		lastJobId uint64
// 		pi        []byte
// 	}{}
// 	if err := json.Unmarshal(data, &m); err != nil {
// 		return err
// 	}

// 	fmt.Printf("data: %v", m)

// 	u.Jobs = m.jobs
// 	u.LastJobID = m.lastJobId
// 	u.LastTerm = m.lastTerm
// 	return u.PI.GobDecode(m.pi)
// }
