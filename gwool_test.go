package gwool

import (
	"testing"
	"time"
)

type emptyPerformer struct{}

func (ep emptyPerformer) Perform(job GwoolJob) {}

type blockingThenDonePerformer struct {
	makeWork chan struct{}
	doneChan chan string
}

func (btdp blockingThenDonePerformer) Perform(job GwoolJob) {
	<-btdp.makeWork
	btdp.doneChan <- (job).(string)
}

func TestCreationCreatesWorkersAndFinishDestroysThem(t *testing.T) {
	p := NewPool(10, make(chan GwoolJob), emptyPerformer{}, 1*time.Second)
	if exp, nworkers := 10, p.numWorkers; exp != nworkers {
		t.Errorf("wrong number of workers created: expected %d got %d", exp, nworkers)
	}
	p.Finish()
	if exp, nworkers := 0, p.numWorkers; exp != nworkers {
		t.Errorf("wrong number of workers after finish: expected %d got %d", exp, nworkers)
	}
}

func TestWorkersPerformWorkAndWaitToFinish(t *testing.T) {
	makeWork := make(chan struct{})
	doneChan := make(chan string, 2)
	p := NewPool(2, make(chan GwoolJob), blockingThenDonePerformer{makeWork, doneChan}, 10*time.Second)
	go func() {
		p.QueueJob("done1")
		p.QueueJob("done2")
		go p.Finish()
		p.QueueJob("done3")
		if expected, actual := 3, p.numWorkers; expected != actual {
			t.Errorf("expected %d workers, got %d", expected, actual)
		}
		close(makeWork)
	}()
	p.Work()
	actualValues := make([]string, 0, 3)

	for i:= 0; i < 3; i++ {
		val := <-doneChan
		if val != "done1" && val != "done2" && val != "done3" {
			t.Errorf("unexpected value %s", val)
		}
		actualValues = append(actualValues, val)
	}

	if len(actualValues) != 3 {
		t.Errorf("unexpected number of values: wanted %d got %d", 3, len(actualValues))
	}
	if actualValues[0] == actualValues[1] || actualValues[0] == actualValues[2] {
		t.Errorf("repeated value %s", actualValues[0])
	}
	if actualValues[1] == actualValues[2] {
		t.Errorf("repeated value %s", actualValues[1])
	}
}

func TestWorkersDieAfterTimeoutAndWorkerIsCreatedWhenNoWorkersLeftAndJobAdded(t *testing.T) {
	makeWork := make(chan struct{})
	doneChan := make(chan string, 1)
	p := NewPool(1, make(chan GwoolJob), blockingThenDonePerformer{makeWork, doneChan}, 1*time.Millisecond)
	go func() {
		time.Sleep(2 * time.Millisecond)
		if expected, actual := 0, p.numWorkers; expected != actual {
			t.Errorf("expected %d workers, got %d", expected, actual)
		}
		p.QueueJob("done1")
		if expected, actual := 1, p.numWorkers; expected != actual {
			t.Errorf("expected %d workers, got %d", expected, actual)
		}
		go p.Finish()
		close(makeWork)
	}()
	p.Work()
	val := <-doneChan
	if val != "done1" {
		t.Errorf("unexpected value %s", val)
	}
}
