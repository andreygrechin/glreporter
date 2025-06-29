package worker

const taskQueueBufferMultiplier = 2

// Pool is a worker pool that executes tasks concurrently.
type Pool struct {
	workers   int
	taskQueue chan func()
}

// NewPool creates a new worker pool with the specified number of workers.
func NewPool(workers int) *Pool {
	p := &Pool{
		workers:   workers,
		taskQueue: make(chan func(), workers*taskQueueBufferMultiplier),
	}

	for range workers {
		go p.worker()
	}

	return p
}

// Submit adds a task to the worker pool.
func (p *Pool) Submit(task func()) {
	p.taskQueue <- task
}

// Shutdown closes the task queue, signaling workers to exit after completing current tasks.
func (p *Pool) Shutdown() {
	close(p.taskQueue)
}

func (p *Pool) worker() {
	for task := range p.taskQueue {
		task()
	}
}
