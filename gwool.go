package gwool

import (
	"sync"
	"time"
)

type GwoolWorker interface {
	Perform(job GwoolJob)
}

type GwoolJob interface{}

type Pool struct {
	jobsQueue     chan GwoolJob
	worker        GwoolWorker
	workerTimeout time.Duration
	numWorkers    int
	minWorkers    int
	stopSignal    chan struct{}
	wg sync.WaitGroup
}

func NewPool(
	initialNoOfWorkers int,
	jobsQueue chan GwoolJob,
	performer GwoolWorker,
	workerTimeout time.Duration,
) *Pool {
	p := &Pool{
		jobsQueue:     jobsQueue,
		worker:        performer,
		workerTimeout: workerTimeout,
		stopSignal:    make(chan struct{}),
	}
	for i := 0; i < initialNoOfWorkers; i++ {
		p.launchWorker()
	}
	return p
}

func (p *Pool) QueueJob(job GwoolJob) {
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
	p.numWorkers++
	workerLaunched := make(chan struct{})
	go func() {
		defer func() {
			p.numWorkers--
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

func (p *Pool) Finish() {
	close(p.stopSignal)
	p.wg.Wait()
}

func (p *Pool) NumOfWorkers() int {
	return p.numWorkers
}

func (p *Pool) Work() {
	<-p.stopSignal
}
