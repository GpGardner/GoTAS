package runner

import (
	"context"
	"math"
	"sync"

	. "GOTAS/internal/job"
)

var _ Runnable[any] = (*static[any])(nil) // Ensure runner implements Runnable

// static represents the job static that executes jobs based on a strategy.
type static[T any] struct {
	strategy StaticStrategy        // strategy is the execution strategy for the runner
	jobs     []Processable[T]      // jobs is the list of jobs to be executed
	wg       *sync.WaitGroup       // wg is the wait group for tracking job completion
	mu       *sync.Mutex           // mu is the mutex for updating job status
	callback func(*Processable[T]) // callback function to be invoked after each job execution, can be nil

	completedJobs int // completedJobs is the number of jobs completed. complete happens regardless of error
	totalJobs     int // totalJobs is the total number of jobs in the runner
}

// NewStaticRunner creates a new job runner with the specified execution strategy and an optional callback function.
//
// Parameters:
//
// - strategy: The execution strategy for the runner (e.g., StrategySequential, StrategyParallel, etc.).
//
// - callback: An optional function to be invoked after each job completes. Can be nil.
//
// Returns:
// - A pointer to the newly created runner instance.
//
// Example:
//
//	runner := NewStaticRunner(StrategyParallel, func(job *Processable[any]) {
//	    fmt.Printf("Job completed with status: %v\n", job.GetStatus())
//	})
func NewStaticRunner[T any](strategy StaticStrategy, callback func(*Processable[T])) *static[T] {
	return &static[T]{
		strategy:      strategy,
		jobs:          make([]Processable[T], 0),
		wg:            &sync.WaitGroup{},
		mu:            &sync.Mutex{},
		completedJobs: 0,
		totalJobs:     0,
	}
}

// AddJob adds a new job to the runner.
//
// Parameters:
// - j: The job to be added. Must implement the Processable interface.
//
// Example:
//
//	job := &Job[Result]{ID: 1}
//	runner.AddJob(job)
func (r *static[T]) AddJob(j Processable[T]) {
	r.mu.Lock()
	r.jobs = append(r.jobs, j)
	r.mu.Unlock()
	r.incrementTotalJobs()
}

// CheckProgress calculates and returns the progress of the runner as a percentage.
//
// Returns:
// - A float64 value representing the percentage of completed jobs (0.0 to 100.0).
//
// Example:
//
//	progress := runner.CheckProgress()
//	fmt.Printf("Progress: %.2f%%\n", progress)
func (r *static[T]) CheckProgress() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Handle division by zero
	if r.totalJobs == 0 {
		return 0.0
	}

	return math.Round((float64(r.completedJobs) / float64(r.totalJobs)) * 100)
}

// Run executes all jobs in the runner based on the specified strategy.
//
// Parameters:
// - ctx: The context for managing cancellation and timeouts.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
//	defer cancel()
//	runner.Run(ctx)
func (r *static[T]) Run(ctx context.Context) error {
	switch r.strategy.(type) {
	case StrategySequential:
		r.runSequential(ctx)
	case StrategyParallel:
		r.runParallel(ctx)
	case StrategyFailFast:
		r.runFailFast(ctx)
	case StrategyPriority:
		r.runPriority(ctx)
	case StrategyRetry:
		r.runPriority(ctx)
	default:
		return ErrInvalidRunnerStrategy
	}
	// Return nil to indicate successful execution of all jobs
	return nil
}

// runSequential executes all jobs in the order they were added to the runner.
// This method is used when the StrategySequential strategy is selected.
//
// Parameters:
// - ctx: The context for managing cancellation and timeouts.
//
// Notes:
// - This method blocks until all jobs are completed.
// - If a callback is provided, it is invoked after each job completes.
func (r *static[T]) runSequential(ctx context.Context) {
	if len(r.jobs) == 0 {
		// No jobs to run, return early
		return
	}
	for _, j := range r.jobs {
		j.Run(ctx)
		r.incrementCompletedJobs()
		if r.callback != nil {
			r.callback(&j) // Invoke the callback
		}
	}
}

// runParallel executes all jobs concurrently.
// This method is used when the StrategyParallel strategy is selected.
//
// Parameters:
// - ctx: The context for managing cancellation and timeouts.
//
// Notes:
// - This method uses a WaitGroup to ensure all jobs are completed before returning.
// - If a callback is provided, it is invoked after each job completes.
func (r *static[T]) runParallel(ctx context.Context) {
	if len(r.jobs) == 0 {
		// No jobs to run, return early
		return
	}
	for _, j := range r.jobs {
		r.wg.Add(1) // Increment WaitGroup counter
		go func(j Processable[T]) {
			defer r.wg.Done() // Ensure WaitGroup counter is decremented even if the job panics
			defer r.incrementCompletedJobs()
			j.Run(ctx)
			if r.callback != nil {
				r.callback(&j)
			}
		}(j)
	}
	r.wg.Wait() // Wait for all jobs to complete
}

// runFailFast executes jobs sequentially but stops execution if a job fails.
// This method is used when the StrategyFailFast strategy is selected.
//
// Parameters:
// - ctx: The context for managing cancellation and timeouts.
//
// Notes:
// - Jobs after the first failure are marked as failed without being executed.
// - If a callback is provided, it is invoked after each job completes or is marked as failed.
func (r *static[T]) runFailFast(ctx context.Context) {
	if len(r.jobs) == 0 {
		// No jobs to run, return early
		return
	}
	failed := false

	for _, j := range r.jobs {
		if !failed {
			j.Run(ctx)
			r.incrementCompletedJobs()
			if j.GetStatus() == StatusError {
				failed = true
			}
			if r.callback != nil {
				r.callback(&j) // Invoke the callback
			}
		} else {
			// Mark the job as failed without running it
			j.Complete(StatusFailed)
			if r.callback != nil {
				r.callback(&j) // Invoke the callback
			}
		}
	}
}

// runPriority executes jobs based on their priority.
// Jobs with higher priority values are executed first.
// This method is used when the StrategyPriority strategy is selected.
//
// Parameters:
// - ctx: The context for managing cancellation and timeouts.
//
// Notes:
// - Jobs are sorted by priority before execution.
// - If a callback is provided, it is invoked after each job completes.
func (r *static[T]) runPriority(ctx context.Context) {
	if len(r.jobs) == 0 {
		// No jobs to run, return early
		return
	}

	pq := &PriorityQueue[any]{}
	// Populate the priority queue with jobs
	for _, j := range r.jobs {
		if j == nil {
			continue // Skip nil jobs
		}
		// Ensure the job implements the Priority interface
		if priorityJob, ok := j.(*PriorityJob[any]); ok {
			// Add to priority queue
			pq.Push(priorityJob)
		} else {
			// Fallback to adding as a normal job if it doesn't implement Priority
			pj, _ := NewPriorityJob[any](j, 0)
			pq.Push(pj)
		}
		r.incrementTotalJobs()
	}

	// Execute jobs based on priority
	for pq.Len() > 0 {
		// Pop the highest priority job
		job := pq.Pop()

		job.(*PriorityJob[any]).Run(ctx) // Execute the job
		r.incrementCompletedJobs()
	}

}

// incrementCompletedJobs increments the count of completed jobs in a thread-safe manner.
//
// Notes:
// - This method uses a mutex to ensure safe access to the completedJobs counter.
func (r *static[T]) incrementCompletedJobs() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completedJobs++
}

// incrementTotalJobs increments the count of total jobs in a thread-safe manner.
//
// Notes:
// - This method uses a mutex to ensure safe access to the totalJobs counter.
func (r *static[T]) incrementTotalJobs() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalJobs++
}
