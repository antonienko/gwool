package gwool

import (
	"sync"
	"time"
)

// Worker is the interface that must be implemented in order to allow Gwool to call your code
type Worker interface {
	Perform(job Job)
}

// Any data type is accepted as Job parameter for the worker
type Job interface{}

// The pool is the data structure that manages the collection of workers. The workers read from
// the jobsQueue channel in order to get the next Job to execute.
type Pool struct {
	jobsQueue     chan Job
	worker        Worker
	workerTimeout time.Duration
	numWorkers    int
	minWorkers    int
	stopSignal    chan struct{}
	wg            sync.WaitGroup
	mux           sync.RWMutex
}

// Creates a new pool. You must provide the initial number of workers that you want to spawn, as
// well as the size of the queue channel. In case that all the workers are busy, no new workers
// will be spawned until the queue is full.
func NewPool(
	initialNoOfWorkers int,
	queueSize int,
	performer Worker,
	workerTimeout time.Duration,
) *Pool {
	p := &Pool{
		jobsQueue:     make(chan Job, queueSize),
		worker:        performer,
		workerTimeout: workerTimeout,
		stopSignal:    make(chan struct{}),
	}
	for i := 0; i < initialNoOfWorkers; i++ {
		p.launchWorker()
	}
	return p
}

// Sends a new Job to the queue. If there's no workers, a new one will be spawned. If the queue
// is full, a new worker will be spawned too.
func (p *Pool) QueueJob(job Job) {
	if p.numWorkers == 0 {
		p.launchWorker()
	}
	select {
	case p.jobsQueue <- job:
	default:
		p.launchWorker()
		p.jobsQueue <- job
	}
}

func (p *Pool) launchWorker() {
	p.wg.Add(1)
	p.mux.Lock()
	p.numWorkers++
	p.mux.Unlock()
	workerLaunched := make(chan struct{})
	go func() {
		defer func() {
			p.mux.Lock()
			p.numWorkers--
			p.mux.Unlock()
			p.wg.Done()
		}()
		p.acceptWork(workerLaunched)
	}()
	<-workerLaunched
}

func (p *Pool) acceptWork(workerLaunched chan struct{}) {
	close(workerLaunched)
	for {
		select {
		case <-p.stopSignal:
			return
		case job := <-p.jobsQueue:
			p.worker.Perform(job)
		case <-time.After(p.workerTimeout):
			return
		}
	}
}

// Sends a signal to all the workers and waits for them to complete their current job and exit
func (p *Pool) Finish() {
	close(p.stopSignal)
	p.wg.Wait()
}

// Gets the current number of spawned workers
func (p *Pool) NumOfWorkers() int {
	p.mux.RLock()
	result := p.numWorkers
	p.mux.RUnlock()
	return result
}

// Waits until a finish signal is sent, to let the workers do their jobs.
func (p *Pool) Work() {
	<-p.stopSignal
}
