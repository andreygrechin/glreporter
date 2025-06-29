package worker_test

import (
	"sync"
	"testing"
	"time"

	"github.com/andreygrechin/glreporter/internal/worker"
)

func TestWorkerPool(t *testing.T) {
	pool := worker.NewPool(5)
	defer pool.Shutdown()

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	counter := 0

	for range 10 {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
		})
	}

	wg.Wait()

	if counter != 10 {
		t.Errorf("expected counter to be 10, got %d", counter)
	}
}
