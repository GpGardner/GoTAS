package runner

import (
	"context"
	"fmt"
	"math"
	"sync"

	. "GOTAS/internal/job"
)

var _ Runnable[any] = (*dynamic[any])(nil) // Ensure runner implements Runnable

const (
	DefaultMaxWorkers  = 8
	MaxAllowedWorkers  = 1000
	DefaultChanSize    = 100
	MaxAllowedChanSize = 10000
)

// dynamic represents a runner that executes jobs based on a strategy and can continuously run
type dynamic[T any] struct {
	strategy DynamicStrategy // strategy is the execution strategy for the runner
	workers  int             //total amount of workers to run
	chanSize int             // chanSize is the size of the buffered channel for jobs default to 100 max 10000
	closed   bool            // closed indicates whether the runner has been closed or not, used to stop accepting jobs

	jobs     chan Processable[T]  // jobs is the list of jobs to be executed
	wg       *sync.WaitGroup      // wg is the wait group for tracking job completion
	mu       *sync.Mutex          // mu is the mutex for updating job status
	callback func(Processable[T]) // callback function to be invoked after each job execution, can be nil

	completedJobs int // completedJobs is the number of jobs completed. complete happens regardless of error
	totalJobs     int // totalJobs is the total number of jobs in the runner
}

// NewDynamicRunner creates a new job runner with the specified execution strategy and an optional callback function.
func NewDynamicRunner[T any](strategy DynamicStrategy, callback func(Processable[T]), workers, chanSize int) *dynamic[T] {
	//default workers 8
	workers = clampWorkers(workers)

	// default channel size if not provided
	chanSize = clampChan(chanSize)

	return &dynamic[T]{
		strategy:      strategy,
		workers:       workers,  // total amount of workers to run
		chanSize:      chanSize, // size of the buffered channel for jobs
		closed:        false,
		jobs:          make(chan Processable[T], chanSize), // Buffered channel to hold jobs, can be adjusted based on requirements
		wg:            &sync.WaitGroup{},
		mu:            &sync.Mutex{},
		callback:      callback,
		completedJobs: 0,
		totalJobs:     0,
	}
}

func clampChan(chanSize int) int {
	// Ensure the channel size is a positive integer, default to 100 if not provided or invalid
	if chanSize <= 0 {
		chanSize = DefaultChanSize // Default buffered channel size
	}

	//protect against too large of a channel size, cap it at 10000 to avoid memory issues
	if chanSize > MaxAllowedChanSize {
		chanSize = MaxAllowedChanSize // Cap the channel size to avoid excessive memory usage
	}
	return chanSize
}

func clampWorkers(workers int) int {
	if workers > MaxAllowedWorkers {
		return MaxAllowedWorkers // Cap at MaxAllowedWorkers
	}
	if workers < DefaultMaxWorkers {
		return DefaultMaxWorkers // Ensure at least the default
	}
	return workers
}

// AddJob implements Runnable.
func (r *dynamic[T]) AddJob(j Processable[T]) error {
	r.mu.Lock()
	if r.closed {
		return fmt.Errorf("runner is closed, cannot accept new jobs")
	}
	r.mu.Unlock()

	r.jobs <- j
	r.incrementTotalJobs()
	return nil
}

func (r *dynamic[T]) IsClosed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

// CheckProgress implements Runnable.
func (r *dynamic[T]) CheckProgress() float64 {

	r.mu.Lock()
	if r.totalJobs == 0 {
		return 0.0 // Avoid division by zero
	}
	if r.completedJobs < 0 {
		r.completedJobs = 0 // Ensure completed jobs is not negative
	}
	// Calculate progress as a percentage
	progress := float64(r.completedJobs) / float64(r.totalJobs)
	r.mu.Unlock()
	return math.Round(progress*100) / 100
}

// Run implements Runnable.
func (r *dynamic[T]) Run(ctx context.Context) {
	switch r.strategy.(type) {
	case StrategyFIFO:
		r.runSequential(ctx)
	case StrategyParallel:
		r.runParallel(ctx)
	case StrategyFailFast:
		r.runFailFast(ctx)
	case StrategyPriority:
		r.runPriority(ctx)
	case StrategyRetry:
		r.runRetry(ctx)
	}
}

func (r *dynamic[T]) runParallel(ctx context.Context) {

	// Create a worker pool to process jobs concurrently
	// Each worker will run a job from the jobs channel
	// The number of workers is determined by the workers field
	// The workers will run until the context is cancelled or the channel is closed
	for i := 0; i < r.workers; i++ {
		r.wg.Add(1)

		go func(workerID int) {
			for {
				select {
				case j, ok := <-r.jobs:
					if !ok {
						r.wg.Done()
						return // Channel closed, stop worker
					}
					_, err := j.Run(ctx)
					if err != nil {
						continue
					}
					if r.callback != nil {
						r.callback(j)
					}
					r.incrementCompletedJobs()
				case <-ctx.Done():
					r.wg.Done()
					return // Context canceled, stop worker
				}
			}
		}(i)
	}
}

func (r *dynamic[T]) runRetry(ctx context.Context) {
	panic("unimplemented")
}

func (r *dynamic[T]) runPriority(ctx context.Context) {
	panic("unimplemented")
}

func (r *dynamic[T]) runFailFast(ctx context.Context) {
	panic("unimplemented")
}

func (r *dynamic[T]) runSequential(ctx context.Context) {

}

// incrementCompletedJobs increments the count of completed jobs in a thread-safe manner.
//
// Notes:
// - This method uses a mutex to ensure safe access to the completedJobs counter.
func (r *dynamic[T]) incrementCompletedJobs() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completedJobs++
}

// incrementTotalJobs increments the count of total jobs in a thread-safe manner.
//
// Notes:
// - This method uses a mutex to ensure safe access to the totalJobs counter.
func (r *dynamic[T]) incrementTotalJobs() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalJobs++
}

func (r *dynamic[T]) Shutdown() {
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		close(r.jobs)
	}

	r.mu.Unlock()
	r.wg.Wait() // Wait for all workers to finish processing
}
