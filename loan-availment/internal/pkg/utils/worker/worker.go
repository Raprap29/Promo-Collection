package worker

// Task represents a unit of work to be processed by a worker
type Task func()

// Worker is a goroutine that processes tasks from a channel
type Worker struct {
	taskQueue chan Task
	stop      chan struct{}
}

// NewWorker creates a new Worker
func NewWorker() *Worker {
	return &Worker{
		taskQueue: make(chan Task),
		stop:      make(chan struct{}),
	}
}

// Start starts the worker to process tasks
func (w *Worker) Start() {
	go func() {
		for {
			select {
			case task := <-w.taskQueue:
				task()
			case <-w.stop:
				return
			}
		}
	}()
}

// Stop stops the worker
func (w *Worker) Stop() {
	close(w.stop)
}

// Submit submits a task to the worker
func (w *Worker) Submit(task Task) {
	w.taskQueue <- task
}
