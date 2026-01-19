package worker

// WorkerPool manages a pool of workers to process tasks
type WorkerPool struct {
	workers []*Worker
	stop    chan struct{}
}

// NewWorkerPool creates a new WorkerPool with the specified number of workers
func NewWorkerPool(numWorkers int) *WorkerPool {
	pool := &WorkerPool{
		workers: make([]*Worker, numWorkers),
		stop:    make(chan struct{}),
	}

	for i := 0; i < numWorkers; i++ {
		worker := NewWorker()
		worker.Start()
		pool.workers[i] = worker
	}

	return pool
}

// Stop stops all workers in the pool
func (p *WorkerPool) Stop() {
	for _, worker := range p.workers {
		worker.Stop()
	}
	close(p.stop)
}

// Submit submits a task to the worker pool
func (p *WorkerPool) Submit(task Task) {
	worker := p.workers[len(p.workers)%cap(p.workers)]
	worker.Submit(task)
}
