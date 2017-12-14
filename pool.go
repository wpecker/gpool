package gpool

import (
	"sync/atomic"
	"time"
)

/*
TunnyWorker - The basic interface of a tunny worker.
*/
type TunnyWorker interface {

	// Called for each job, expects the result to be returned synchronously
	TunnyJob(interface{}) interface{}

	// Called after each job, this indicates whether the worker is ready for the next job.
	// The default implementation is to return true always. If false is returned then the
	// method is called every five milliseconds until either true is returned or the pool
	// is closed. For efficiency you should have this call block until your worker is ready,
	// otherwise you introduce a 5ms latency between jobs.
	TunnyReady() bool
}

// Pool
type Pool struct {
	workerPool     chan chan interface{} // A pool of workers channels that are registered with the dispatcher
	jobQueue       chan interface{}      // A buffered channel that we can send work requests on.
	handler        TunnyWorker
	inControl      chan struct{}
	outControl     chan struct{}
	controlTimeout time.Duration
	count          uint64
	maxWorkers     int // A Max works
}

func NewPool(maxWorkers, maxQueue int, timeOut time.Duration) *Pool {
	return &Pool{
		workerPool:     make(chan chan interface{}, maxWorkers),
		jobQueue:       make(chan interface{}, maxQueue),
		inControl:      make(chan struct{}, maxQueue),
		outControl:     make(chan struct{}, maxQueue),
		controlTimeout: timeOut,
		maxWorkers:     maxWorkers,
	}
}

func (d *Pool) Open(f TunnyWorker) {
	d.handler = f
	// starting n number of workers
	for i := 1; i < d.maxWorkers+1; i++ {
		worker := NewWorker(d.workerPool, d.handler, d.inControl, d.outControl, i)
		worker.Start()
	}
	go d.dispatch()
	go d.recycling()
}

func (d *Pool) SendJob(work interface{}) bool {
	select {
	case <-time.After(d.controlTimeout):
		return false
	case d.inControl <- struct{}{}:
		// Push the work onto the queue.
		atomic.AddUint64(&d.count, 1)
		d.jobQueue <- work
		return true
	}
}

func (d *Pool) GetCount() uint64 {
	c := atomic.LoadUint64(&d.count)
	return c
}

func (d *Pool) recycling() {
	for {
		select {
		case <-d.outControl:
			atomic.AddUint64(&d.count, ^uint64(1-1))
		}
	}
}

func (d *Pool) dispatch() {
	for {
		select {
		case receiveJob := <-d.jobQueue:
			go func(job interface{}) {

				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				jobChannel := <-d.workerPool

				// dispatch the job to the worker job channel
				jobChannel <- job
			}(receiveJob)
		}
	}
}
