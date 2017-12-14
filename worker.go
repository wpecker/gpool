package gpool

// Worker represents the worker that executes the job
type Worker struct {
	workerChannel chan chan interface{}
	jobChannel    chan interface{}
	worker        TunnyWorker
	inControl     chan struct{}
	outControl    chan struct{}
	quit          chan bool
	no            int
}

func NewWorker(workerPool chan chan interface{}, f TunnyWorker, i, o chan struct{}, no int) Worker {
	return Worker{
		workerChannel: workerPool,
		worker:        f,
		inControl:     i,
		outControl:    o,
		no:            no,
		jobChannel:    make(chan interface{}),
		quit:          make(chan bool),
	}
}

// Stop signals the worker to stop listening for work requests.
func (w *Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w *Worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.workerChannel <- w.jobChannel

			select {
			case job := <-w.jobChannel:
				// Receive the new task

				// we have received a work request.
				w.worker.TunnyJob(job)

				<-w.inControl
				w.outControl <- struct{}{}

			case <-w.quit:
				// we have received a signal to stop
				return
			}
		}
	}()
}
