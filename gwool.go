package gwool

import (
	"sync"
	"time"
)

type GwoolWorker interface {
	Perform(job GwoolJob)
}

type GwoolJob interface{}

type gwool struct {
	jobsQueue     chan GwoolJob
	worker        GwoolWorker
	workerTimeout time.Duration
	numWorkers    int
	minWorkers    int
	stopSignal    chan struct{}
	sync.WaitGroup
}

func NewPool(
	initialNoOfWorkers int,
	jobsQueue chan GwoolJob,
	performer GwoolWorker,
	workerTimeout time.Duration,
) *gwool {
	p := &gwool{
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

func (p *gwool) QueueJob(job GwoolJob) {
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

func (p *gwool) launchWorker() {
	p.Add(1)
	p.numWorkers++
	workerLaunched := make(chan struct{})
	go func() {
		defer func() {
			p.numWorkers--
			p.Done()
		}()
		p.acceptWork(workerLaunched)
	}()
	<-workerLaunched
}

func (p *gwool) acceptWork(workerLaunched chan struct{}) {
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

func (p *gwool) Finish() {
	close(p.stopSignal)
	p.Wait()
}

func (p *gwool) NumOfWorkers() int {
	return p.numWorkers
}

func (p *gwool) Work() {
	<-p.stopSignal
}
