package gwool

import (
	"testing"
	"time"
)

type emptyPerformer struct{}

func (ep emptyPerformer) Perform(job Job) {}

type blockingThenDonePerformer struct {
	makeWork chan struct{}
	doneChan chan string
}

func (btdp blockingThenDonePerformer) Perform(job Job) {
	<-btdp.makeWork
	btdp.doneChan <- (job).(string)
}

func TestCreationCreatesWorkersAndFinishDestroysThem(t *testing.T) {
	p := NewPool(10, 1, emptyPerformer{}, 1*time.Second)
	if exp, nworkers := 10, p.NumOfWorkers(); exp != nworkers {
		t.Errorf("wrong number of workers created: expected %d got %d", exp, nworkers)
	}
	p.Finish()
	if exp, nworkers := 0, p.NumOfWorkers(); exp != nworkers {
		t.Errorf("wrong number of workers after finish: expected %d got %d", exp, nworkers)
	}
}

func TestWorkersPerformWorkAndWaitToFinish(t *testing.T) {
	makeWork := make(chan struct{})
	doneChan := make(chan string, 3)
	p := NewPool(1, 1, blockingThenDonePerformer{makeWork, doneChan}, 10*time.Second)
	go func() {
		p.QueueJob("done1")
		p.QueueJob("done2")
		p.QueueJob("done3")
		if expected, actual := 2, p.NumOfWorkers(); expected != actual {
			t.Errorf("expected %d workers, got %d", expected, actual)
		}
		go p.Finish()
		close(makeWork)
	}()
	p.Work()
	actualValues := make([]string, 0, 3)

	for i := 0; i < 3; i++ {
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
		t.Errorf("job executed twice %s", actualValues[0])
	}
	if actualValues[1] == actualValues[2] {
		t.Errorf("job executed twice %s", actualValues[1])
	}
}

func TestWorkersDieAfterTimeoutAndWorkerIsCreatedWhenNoWorkersLeftAndJobAdded(t *testing.T) {
	makeWork := make(chan struct{})
	doneChan := make(chan string, 1)
	p := NewPool(1, 1, blockingThenDonePerformer{makeWork, doneChan}, 1*time.Millisecond)
	go func() {
		time.Sleep(10 * time.Millisecond)
		if expected, actual := 0, p.NumOfWorkers(); expected != actual {
			t.Errorf("expected %d workers, got %d", expected, actual)
		}
		p.QueueJob("done1")
		if expected, actual := 1, p.NumOfWorkers(); expected != actual {
			t.Errorf("expected %d workers, got %d", expected, actual)
		}
		close(makeWork)
		p.Finish()
	}()
	p.Work()
	val := <-doneChan
	if val != "done1" {
		t.Errorf("unexpected value %s", val)
	}
}
