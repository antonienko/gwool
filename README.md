#GWOOL

GWOOL stands for Golang Worker pOOL, a very simplistic and basic worker pool, with a naive auto-scaling approach. 

[![Build Status](https://travis-ci.org/antonienko/gwool.svg?branch=master)](https://travis-ci.org/antonienko/gwool)
[![Maintainability](https://api.codeclimate.com/v1/badges/d7c85499faa5944b2c2d/maintainability)](https://codeclimate.com/github/antonienko/gwool/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/d7c85499faa5944b2c2d/test_coverage)](https://codeclimate.com/github/antonienko/gwool/test_coverage)

Given an initial number of concurrent workers that are launched as goroutines, and a provided timeout duration
Gwool tries to keep the workers number at the minimum possible. When an idle worker reaches the timeout it dies
so resources can be freed.

In the event that the number of workers reach 0 but new jobs come in, new workers will
be automatically spawned to serve the request.

In a similar way, if the queue is full and a new job arrives, a worker will be created so it can help with the tasks peak.

**Warning**: As of version 1.0 there's no upper limit in the number of workers that can be spawned.

## Install

``` bash
go get github.com/antonienko/gwool
```

Or, using dep:

``` bash
dep ensure -add github.com/antonienko/gwool
```

## Use

Your worker must implement gwool's `Worker` interface. A simple example of a worker that counts how many jobs
are being performed at any given, each job consisting of an integer that has to be printed after a certain timeout, 
would look like:

``` go
package main
import (
    "fmt"
    "time"

    "github.com/antonienko/gwool"
)

type myWorker struct {
    concurrentJobs int
}

func (mw *myWorker) Perform(number Job) {
    mw.concurrentJobs++             //Maybe a mutex is needed here to avoid race conditions. Left out for simplicity
    time.Sleep(10 * time.Second)
    fmt.Println(number)
    mw.concurrentJobs--             //same applies here, mutex maybe needed.
}

func main() {
    pool := gwool.NewPool(3, 10, worker, 10 * time.Second)
    //3 workers will be launched, queue size is 10
    
    go func() {
        pool.QueueJob(1)
        pool.QueueJob(2)
        pool.QueueJob(3)
        pool.Finish()
    }()

    pool.Work() 
}
```