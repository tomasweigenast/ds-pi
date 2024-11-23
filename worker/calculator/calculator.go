package calculator

import (
	"log"
	"math"
	"math/big"
	"net"
	"net/rpc"
	"time"

	"ds-pi.com/master/shared"
)

type Calculator struct {
	masterAddr net.TCPAddr
	client     *rpc.Client
	workerName string

	job        *currentJob
	pingTimer  *shared.Timer
	connecting bool
}

type currentJob struct {
	id        uint64
	startTerm uint64
	numTerms  uint64
	result    big.Float
	completed bool
	precision uint
}

func NewCalculator(masterIP net.IP, port int) *Calculator {
	c := &Calculator{
		masterAddr: net.TCPAddr{
			IP:   masterIP,
			Port: port,
		},
	}

	c.pingTimer = shared.NewTimer(time.Second*5, func() {
		onPingTimerTick(c)
	})

	return c
}

func (c *Calculator) Run() {
	c.run()
}

func (c *Calculator) Stop() {
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}

func (c *Calculator) run() {
	tryCount := 0
	for {
		log.Printf("Trying to connect. Count = %d", tryCount)
		if c.connect() {
			break
		}

		if tryCount > 20 {
			log.Fatalf("Giving up after 20 tries.")
			return
		}

		log.Printf("Unable to connect to master. Trying again in 5 seconds...")
		tryCount++
		time.Sleep(5 * time.Second)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered")
			c.reconnect()
		}
	}()

	// ask jobs
	for {
		if c.askJob() {
			if result := c.calculate(); result != nil {
				c.send(result)
			}
		} else {
			c.reconnect()
		}
	}
}

func (c *Calculator) connect() bool {
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}

	client, err := rpc.DialHTTP("tcp", c.masterAddr.String())
	if err != nil {
		log.Printf("failed to dial tcp to master: %s", err)
		return false
	}

	c.client = client

	myIP, err := shared.GetIPv4()
	if err != nil {
		log.Printf("Unable to get my ip: %s", err)
		return false
	}

	args := &shared.ConnectArgs{WorkerIP: myIP.String()}
	var reply shared.ConnectReply
	err = c.client.Call("ConnectService.Connect", args, &reply)
	if err != nil {
		log.Printf("Unable to call Connect rpc method: %s", err)
		c.tryAgain()
	}

	c.workerName = reply.WorkerName
	log.Printf("My name is: %s", c.workerName)

	return true
}

func (c *Calculator) reconnect() {
	if c.connecting {
		return
	}

	log.Printf("Trying to reconnect")
	c.connecting = true

	if c.client != nil {
		c.client.Close()
		c.client = nil
		c.workerName = ""
		c.job = nil
	}
	time.Sleep(10 * time.Second)
	c.connecting = false
	c.run()
}

func (c *Calculator) tryAgain() {
	log.Printf("Trying to reconnect in 5 seconds...")
	time.Sleep(5 * time.Second)
	c.run()
}

func (c *Calculator) askJob() bool {
	log.Printf("Asking for job...")

	if c.client == nil {
		return false
	}

	args := &shared.AskArgs{WorkerName: c.workerName}
	var reply shared.AskReply
	err := c.client.Call("JobsService.Ask", args, &reply)
	if err != nil {
		log.Printf("unable to ask for a job: %s", err)
		return false
	}

	/*
		la precision se calcula como:
		precision_bits≈(log2 N + n*3.32)

		donde:
		N es el número de terminos asignados al worker.
		n es la cantidad de decimales esperados calcula
		3.32 viene de cuantos bits se necesitan por digito decimal (aprox):
		1/(log10(2)) ≈ 3.32193

		nota: la precision no puede ser mayor que un uint32 (tecnicamente limitado por la memoria del sistema)
	*/
	precision := uint(math.Log2(float64(reply.NumTerms))+10000*3.32) * 3
	c.job = &currentJob{
		id:        reply.JobID,
		startTerm: reply.StartTerm,
		numTerms:  reply.NumTerms,
		precision: precision,
	}
	log.Printf("Job received. Id [%d] First Term [%d] Num Terms [%d] Precision Set [%d]", reply.JobID, reply.StartTerm, reply.NumTerms, c.job.precision)
	return true
}

func (c *Calculator) calculate() []byte {
	start := time.Now()

	result := new(big.Float).SetPrec(uint(c.job.precision) * 2).SetFloat64(0)
	until := c.job.startTerm + c.job.numTerms
	for k := c.job.startTerm; k < until; k++ {
		term := calculateTerm(k, c.job.precision)
		result.Add(result, term)

		if result.Acc() != big.Exact {
			log.Printf("Accuracy is not exact. It is %s", result.Acc())
		}
	}

	elapsed := time.Now().Sub(start)
	log.Printf("Job calculated in %s. Result (rounded) %s", elapsed, result.String())

	buffer, err := result.GobEncode()
	if err != nil {
		log.Printf("unable to encode result as gob: %s", err)
		return nil
	}

	return buffer
}

func (c *Calculator) send(buffer []byte) bool {
	if c.client == nil {
		return false
	}

	args := &shared.GiveArgs{
		JobID:     c.job.id,
		Result:    buffer,
		Precision: c.job.precision,
	}
	var reply shared.GiveReply
	err := c.client.Call("JobsService.Give", args, &reply)
	if err != nil {
		log.Printf("unable to call JobsService.Give: %s", err)
		return false
	}

	log.Printf("Job result sent.")
	return true
}

func calculateTerm(k uint64, precision uint) *big.Float {
	// Se crea un nuevo numero para guardar el resultado
	term := new(big.Float).SetPrec(uint(precision))

	// Calculo de terminos de BBP
	part1 := new(big.Float).Quo(big.NewFloat(4), big.NewFloat(float64(8*k+1)))
	part2 := new(big.Float).Quo(big.NewFloat(2), big.NewFloat(float64(8*k+4)))
	part3 := new(big.Float).Quo(big.NewFloat(1), big.NewFloat(float64(8*k+5)))
	part4 := new(big.Float).Quo(big.NewFloat(1), big.NewFloat(float64(8*k+6)))

	// Se suman los terminos
	term = term.Add(term, part1)
	term = term.Sub(term, part2)
	term = term.Sub(term, part3)
	term = term.Sub(term, part4)

	// Se multiplica por 1/16^k (vaye uno a saber por qué)
	power := new(big.Int).Exp(big.NewInt(16), big.NewInt(int64(k)), nil)
	multiplier := new(big.Float).SetPrec(uint(precision)).Quo(big.NewFloat(1), new(big.Float).SetInt(power))
	term.Mul(term, multiplier)

	return term
}

func onPingTimerTick(c *Calculator) {
	if c.client != nil {
		args := &shared.PingArgs{WorkerName: c.workerName}
		var reply shared.PingReply
		err := c.client.Call("PingService.Ping", args, &reply)
		if err != nil {
			log.Printf("Unable to ping master: %s", err)
			log.Println("Disconnecting and trying again in 10 seconds...")
			c.reconnect()
		}
	}
}
