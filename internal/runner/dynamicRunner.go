package runner

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	. "GOTAS/internal/job"
)

var _ Runnable[any] = (*dynamic[any])(nil) // Ensure runner implements Runnable

const (
	DefaultMaxWorkers      = 1
	MaxAllowedWorkers      = 1000
	DefaultChanSize        = 100
	MaxAllowedChanSize     = 10000
	DefaultBackoff         = 3 * time.Second
	DefaultMaxRetries      = 3
	DefaultMaxWaitForClose = 10 * time.Second
)

type RunnerBuilder[T any] struct {
	strategy        DynamicStrategy
	callback        func(Processable[T])
	workers         int
	chanSize        int
	backoff         time.Duration
	maxRetries      int
	maxWaitForClose time.Duration
}

// NewRunnerBuilder creates a new RunnerBuilder with default values.
// It allows for fluent configuration of the runner's properties.
//
// Returns:
// - *RunnerBuilder[T]: A pointer to the RunnerBuilder instance with default values.
// Example usage:
// builder := NewRunnerBuilder[Job]().WithWorkers(5).WithChanSize(200).WithBackoff(2 * time.Second)
// This creates a new RunnerBuilder for jobs with 5 workers, a channel size of 200, and a backoff duration of 2 seconds.
//
// Notes:
// - Can override default values using the provided methods.
// - WithStrategy, WithCallback, WithWorkers, WithChanSize, WithBackoff, WithMaxRetries, and WithMaxWaitForClose methods are available for configuration.
func NewRunnerBuilder[T any]() *RunnerBuilder[T] {
	return &RunnerBuilder[T]{
		workers:         DefaultMaxWorkers,
		chanSize:        DefaultChanSize,
		backoff:         DefaultBackoff,
		maxRetries:      DefaultMaxRetries,
		maxWaitForClose: DefaultMaxWaitForClose,
	}
}

// WithStrategy sets the execution strategy for the runner.
// It allows for fluent configuration of the runner's properties.
// Parameters:
// - strategy: The execution strategy to be used by the runner.
// Returns:
// - *RunnerBuilder[T]: A pointer to the RunnerBuilder instance with the updated strategy.
func (b *RunnerBuilder[T]) WithStrategy(strategy DynamicStrategy) *RunnerBuilder[T] {
	b.strategy = strategy
	return b
}

// WithCallback sets the callback function to be invoked after each job execution.
// It allows for fluent configuration of the runner's properties.
// Parameters:
// - callback: The callback function to be invoked after each job execution.
// Returns:
// - *RunnerBuilder[T]: A pointer to the RunnerBuilder instance with the updated callback.
func (b *RunnerBuilder[T]) WithCallback(callback func(Processable[T])) *RunnerBuilder[T] {
	b.callback = callback
	return b
}

// WithWorkers sets the number of workers for the runner.
// It allows for fluent configuration of the runner's properties.
// Parameters:
// - workers: The number of workers to be used by the runner.
// Returns:
// - *RunnerBuilder[T]: A pointer to the RunnerBuilder instance with the updated number of workers.
// Notes:
// - The number of workers is clamped to a maximum of MaxAllowedWorkers(10000) and a minimum of DefaultMaxWorkers(1).
func (b *RunnerBuilder[T]) WithWorkers(workers int) *RunnerBuilder[T] {
	b.workers = clampWorkers(workers)
	return b
}

// WithChanSize sets the size of the buffered channel for jobs.
// It allows for fluent configuration of the runner's properties.
// Parameters:
// - chanSize: The size of the buffered channel for jobs.
// Returns:
// - *RunnerBuilder[T]: A pointer to the RunnerBuilder instance with the updated channel size.
// Notes:
// - The channel size is clamped to a maximum of MaxAllowedChanSize(10000) and a minimum of DefaultChanSize(100).
func (b *RunnerBuilder[T]) WithChanSize(chanSize int) *RunnerBuilder[T] {
	b.chanSize = clampChan(chanSize)
	return b
}

// WithBackoff sets the backoff duration for retrying failed jobs.
// It allows for fluent configuration of the runner's properties.
// Parameters:
// - backoff: The backoff duration for retrying failed jobs.
// Returns:
// - *RunnerBuilder[T]: A pointer to the RunnerBuilder instance with the updated backoff duration.
// Notes:
// - The default backoff duration is DefaultBackoff(3 seconds).
func (b *RunnerBuilder[T]) WithBackoff(backoff time.Duration) *RunnerBuilder[T] {
	b.backoff = backoff
	return b
}

// WithMaxRetries sets the maximum number of retries for failed jobs.
// It allows for fluent configuration of the runner's properties.
// Parameters:
// - maxRetries: The maximum number of retries for failed jobs.
// Returns:
// - *RunnerBuilder[T]: A pointer to the RunnerBuilder instance with the updated maximum number of retries.
// Notes:
// - The default maximum number of retries is DefaultMaxRetries(3).
func (b *RunnerBuilder[T]) WithMaxRetries(maxRetries int) *RunnerBuilder[T] {
	b.maxRetries = maxRetries
	return b
}

// WithMaxWaitForClose sets the maximum wait time for the runner to close gracefully.
// It allows for fluent configuration of the runner's properties.
// Parameters:
// - maxWait: The maximum wait time for the runner to close gracefully.
// Returns:
// - *RunnerBuilder[T]: A pointer to the RunnerBuilder instance with the updated maximum wait time.
// Notes:
// - The default maximum wait time is DefaultMaxWaitForClose(10 seconds).
func (b *RunnerBuilder[T]) WithMaxWaitForClose(maxWait time.Duration) *RunnerBuilder[T] {
	b.maxWaitForClose = maxWait
	return b
}

// Build creates a new dynamic runner with the specified configuration.
// It allows for fluent configuration of the runner's properties.
// Returns:
// - *dynamic[T]: A pointer to the newly created dynamic runner instance.
// Example usage:
// runner := NewRunnerBuilder[Job]().
//
//	WithStrategy(StrategyParallel).
//	WithWorkers(5).
//	WithChanSize(200).
//	WithBackoff(2 * time.Second).
//	WithMaxRetries(5).
//	WithMaxWaitForClose(30 * time.Second).
//	Build()
//
// This creates a new dynamic runner for jobs with the specified configuration.
func (b *RunnerBuilder[T]) Build() *dynamic[T] {
	return &dynamic[T]{
        strategy:        b.strategy,
        workers:         b.workers,
        chanSize:        b.chanSize,
        maxRetries:      b.maxRetries,
        backoff:         b.backoff,
        maxWaitForClose: b.maxWaitForClose,
        jobs:            make(chan Processable[T], b.chanSize),
        wg:              &sync.WaitGroup{},
        callback:        b.callback,
        completedJobs:   0,
        totalJobs:       0,
    }
}

// dynamic represents a runner that executes jobs based on a strategy and can continuously run
type dynamic[T any] struct {
	strategy DynamicStrategy // strategy is the execution strategy for the runner

	workers         int           //total amount of workers to run
	chanSize        int           // chanSize is the size of the buffered channel for jobs default to 100 max 10000
	maxRetries      int           // maxRetries is the maximum number of retries for failed jobs
	backoff         time.Duration // backoff is the backoff duration for retrying failed jobs
	maxWaitForClose time.Duration // maxWaitForClose is the maximum wait time for the runner to close gracefully

	closeOnce sync.Once            // Ensures the channel is closed only once
	jobs      chan Processable[T]  // jobs is the list of jobs to be executed
	wg        *sync.WaitGroup      // wg is the wait group for tracking job completion
	callback  func(Processable[T]) // callback function to be invoked after each job execution, can be nil

	completedJobs int32 // completedJobs is the number of jobs completed. complete happens regardless of error
	totalJobs     int32 // totalJobs is the total number of jobs in the runner
}

// NewDynamicRunner creates a new job runner with the specified execution strategy and an optional callback function.
func NewDynamicRunner[T any](strategy DynamicStrategy, callback func(Processable[T]), workers, chanSize int) *dynamic[T] {
	//default workers 8
	workers = clampWorkers(workers)

	// default channel size if not provided
	chanSize = clampChan(chanSize)

	return &dynamic[T]{
		strategy:        strategy,
		workers:         workers,                             // total amount of workers to run
		chanSize:        chanSize,                            // size of the buffered channel for jobs
		maxRetries:      DefaultMaxRetries,                   // maximum number of retries for failed jobs
		backoff:         DefaultBackoff,                      // backoff duration for retrying failed jobs
		maxWaitForClose: DefaultMaxWaitForClose,              // maximum wait time for the runner to close gracefully
		jobs:            make(chan Processable[T], chanSize), // Buffered channel to hold jobs, can be adjusted based on requirements
		wg:              &sync.WaitGroup{},
		callback:        callback,
		completedJobs:   0,
		totalJobs:       0,
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
	baseDelay := 100 * time.Millisecond // Base delay for retries

	for {
		select {
		case r.jobs <- j: // Successfully added the job
			// fmt.Printf("Job added to queue: %v\n", j)
			r.incrementTotalJobs()
			return nil
		default: // Channel is either full or closed
			// Check if the channel is closed
			if isChannelClosed(r.jobs) {
				return fmt.Errorf("runner is closed, cannot accept new jobs")
			}

			//fmt.Printf("Job queue is full, retrying in %v...\n", baseDelay)
			time.Sleep(baseDelay) // Wait before retrying
		}
	}
}

// Helper function to check if a channel is closed
func isChannelClosed[T any](ch <-chan T) bool {
	select {
	case _, ok := <-ch:
		return !ok // If `ok` is false, the channel is closed
	default:
		return false // Channel is not closed
	}
}

// CheckProgress implements Runnable.
func (r *dynamic[T]) CheckProgress() float64 {
	totalJobs := atomic.LoadInt32(&r.totalJobs)         // Use atomic for reading
	completedJobs := atomic.LoadInt32(&r.completedJobs) // Use atomic for reading

	if totalJobs == 0 {
		return 0.0 // Avoid division by zero
	}
	if completedJobs < 0 {
		completedJobs = 0 // Ensure completed jobs is not negative
	}

	// Calculate progress as a percentage
	progress := (float64(completedJobs) / float64(totalJobs)) * 100
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
	// case StrategyPriority:
	// 	r.runPriority(ctx)
	case StrategyRetry:
		r.runRetry(ctx)
	}
}

// runParallel executes all jobs concurrently using a worker pool.
// This method is used when the StrategyParallel strategy is selected.
// each job will always be run ctx cancelled and errors are handled by the job
// Parameters:
// - ctx: The context for managing cancellation and timeouts.
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
//	defer cancel()
//	runner.Run(ctx)
func (r *dynamic[T]) runParallel(ctx context.Context) {
	for i := 0; i < r.workers; i++ {
		r.wg.Add(1)

		go func(workerID int) {
			defer r.wg.Done() // Ensure Done is called when the worker exits
			for {
				select {
				case j, ok := <-r.jobs:
					if j != nil {
						// fmt.Printf("Worker %d working job %+v\n", workerID, j)
					} else {
						// fmt.Printf("Worker %d received nil job\n", workerID)
					}

					if !ok {
						//fmt.Printf("Worker %d shutting down\n", workerID)
						return // Channel closed, stop worker
					}
					fmt.Printf("Worker %d dequeued job: %v", workerID, j)

					_, err := j.Run(ctx)
					if r.callback != nil {
						r.callback(j)
					}
					if err != nil {
						// fmt.Printf("Worker %d encountered error: %v\n", workerID, err)
						continue
					}

					r.incrementCompletedJobs()
				case <-time.After(10 * time.Second):
					// fmt.Printf("Worker %d waiting for work\n", workerID)
					continue // Worker is idle, continue waiting for jobs
				}
			}
		}(i)
	}
}

// runRetry executes all jobs concurrently using a worker pool.
// This method is used when the StrategyRetry strategy is selected.
// each job will always be run ctx cancelled and errors are retried
// Parameters:
// - ctx: The context for managing cancellation and timeouts.
func (r *dynamic[T]) runRetry(ctx context.Context) {

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
						retryCount := 0
						for retryCount < r.maxRetries {
							retryCtx, cancel := context.WithTimeout(ctx, time.Duration(r.backoff)*time.Second)
							defer cancel()
							_, err = j.Run(retryCtx)
							if err == nil {
								break // Job succeeded
							}
							retryCount++
							r.backoff *= 2 // Exponential backoff
						}
					}
					if r.callback != nil {
						r.callback(j)
					}

					r.incrementCompletedJobs()
				}
			}
		}(i)
	}
}

// func (r *dynamic[T]) runPriority(ctx context.Context) {
// 	pq := &PriorityQueue[T]{}

// 	// Constantly read from the jobs channel and push to the priority queue
// 	go func() {
// 		for _, j := range r.jobs {
// 			if j == nil {
// 				continue // Skip nil jobs
// 			}
// 			// Ensure the job implements the Priority interface
// 			if priorityJob, ok := j.(*PriorityJob[T]); ok {
// 				// Add to priority queue
// 				pq.Push(priorityJob)
// 			} else {
// 				// Fallback to adding as a normal job if it doesn't implement Priority
// 				pj, _ := NewPriorityJob(j, 0)
// 				pq.Push(pj)
// 			}
// 		}
// 	}()

// 	// Create a worker pool to process jobs concurrently
// 	for i := 0; i < r.workers; i++ {
// 		r.wg.Add(1)

// 		go func(workerID int) {
// 			for {
// 				select {
// 				case j, ok := <-r.jobs:
// 					if !ok {
// 						r.wg.Done()
// 						return // Channel closed, stop worker
// 					}

// 					if atomic.LoadInt32(&errState) == 1 {
// 						// If an error has occurred, cancel the job and run it with the done context
// 						done, cancel := context.WithCancel(ctx)
// 						cancel()
// 						_, err := j.Run(done)
// 						if err != nil {
// 						}
// 						if r.callback != nil {
// 							r.callback(j)
// 						}
// 					} else {
// 						_, err := j.Run(ctx)
// 						if r.callback != nil {
// 							r.callback(j)
// 						}

// 						if err != nil {
// 							// Set the error state and stop all workers
// 							atomic.StoreInt32(&errState, 1)
// 						}
// 					}

// 					r.incrementCompletedJobs()

// 				}
// 			}
// 		}(i)
// 	}
// 	}
// }

func (r *dynamic[T]) runFailFast(ctx context.Context) {
	// Local atomic error state
	var errState int32 // 0 means no error, 1 means an error occurred

	// Create a worker pool to process jobs concurrently
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

					if atomic.LoadInt32(&errState) == 1 {
						// If an error has occurred, cancel the job and run it with the done context
						done, cancel := context.WithCancel(ctx)
						cancel()
						_, err := j.Run(done)
						if err != nil {
						}
						if r.callback != nil {
							r.callback(j)
						}
					} else {
						_, err := j.Run(ctx)
						if r.callback != nil {
							r.callback(j)
						}

						if err != nil {
							//fmt.Printf("Worker %d encountered error on job: %v\n", workerID, err)
							atomic.StoreInt32(&errState, 1)
						}
					}

					r.incrementCompletedJobs()

				}
			}
		}(i)
	}
}

// runSequential needs to be fleshed out more; should it implement some sort of waiting system for a number of jobs to be completed before moving
// on to the next group of jobs?
// executes all jobs in the order they were added to the runner.
// This method is used when the StrategySequential strategy is selected.
// Parameters:
// - ctx: The context for managing cancellation and timeouts.
// Notes:
// - This method blocks until all jobs are completed.
// - If a callback is provided, it is invoked after each job completes.
func (r *dynamic[T]) runSequential(ctx context.Context) {

	// Create a worker pool to process jobs concurrently
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
					}
					if r.callback != nil {
						r.callback(j)
					}
					if err != nil {
					}
				}

				r.incrementCompletedJobs()

			}
		}(i)
	}
}

// incrementCompletedJobs increments the count of completed jobs in a thread-safe manner.
//
// Notes:
// - This method uses atomic to ensure safe access to the completedJobs counter.
func (r *dynamic[T]) incrementCompletedJobs() {
	atomic.AddInt32(&r.completedJobs, 1)
}

// incrementTotalJobs increments the count of total jobs in a thread-safe manner.
//
// Notes:
// - This method uses atomic to ensure safe access to the totalJobs counter.
func (r *dynamic[T]) incrementTotalJobs() {
	atomic.AddInt32(&r.totalJobs, 1)
}

func (r *dynamic[T]) ShutdownGracefully(cancel context.CancelFunc) {
	r.closeOnce.Do(func() {
		close(r.jobs) // Close the jobs channel to signal no more jobs will be added
	})

	// // Wait for the queue to be fully drained
	// for {
	// 	remainingJobs := len(r.jobs)
	// 	if remainingJobs == 0 {
	// 		break
	// 	}
	// 	//fmt.Printf("jobs remaining in the queue %d, waiting for workers to finish\n", remainingJobs)
	// 	time.Sleep(10 * time.Millisecond) // Polling interval
	// }

	// Wait for all workers to finish processing
	done := make(chan struct{})
	go func() {
		r.wg.Wait() // Wait for all workers to finish
		close(done)
	}()

	// Wait for workers to finish or timeout
	select {
	case <-done:
		//fmt.Println("All workers shut down gracefully")
	case <-time.After(r.maxWaitForClose):
		//fmt.Println("Graceful shutdown timed out")
		cancel() // Cancel the context to forcefully stop workers
	}
}

func (r *dynamic[T]) ShutdownImmediately() {
	//fmt.Println("Forcing immediate shutdown")
	r.closeOnce.Do(func() {
		close(r.jobs) // Close the channel immediately
	})
	r.wg.Wait() // Wait for workers to exit
	//fmt.Println("All workers exited")
}
