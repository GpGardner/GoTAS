package runner

import (
	"context"
	"math"
	"sync"

	. "GOTAS/internal/job"
	logger "GOTAS/internal/log"
)

var _ Runnable[any] = (*static[any])(nil) // Ensure runner implements Runnable

// static represents the job static that executes jobs based on a strategy.
type static[T any] struct {
	log      logger.Logger    // log is the logger for the runner
	strategy StaticStrategy   // strategy is the execution strategy for the runner
	jobs     []Processable[T] // jobs is the list of jobs to be executed
	wg       *sync.WaitGroup  // wg is the wait group for tracking job completion
	mu       *sync.Mutex      // mu is the mutex for updating job status
	callback func(*T, *error) // callback function to be invoked after each job execution, can be nil

	completedJobs int // completedJobs is the number of jobs completed. complete happens regardless of error
	totalJobs     int // totalJobs is the total number of jobs in the runner
}

// StaticBuilder is a builder for creating a static runner.
// It allows setting the logger, strategy, and callback function.
// The builder pattern is used to create a static runner instance with optional configurations.
// Example usage:
// builder := NewStaticBuilder[Result](). WithLogger(logger).WithStrategy(strategy).WithCallback(callback)
// runner :=
type StaticBuilder[T any] struct {
	log      logger.Logger    // log is the logger for the runner
	strategy StaticStrategy   // strategy is the execution strategy for the runner
	callback func(*T, *error) // callback function to be invoked after each job execution, can be nil
}

func NewStaticBuilder[T any]() *StaticBuilder[T] {
	return &StaticBuilder[T]{
		log:      logger.NewNoOpLogger(),
		strategy: StrategySequential{},
		callback: nil,
	}
}

func (b *StaticBuilder[T]) WithLogger(l logger.Logger) *StaticBuilder[T] {
	b.log = l
	return b
}

func (b *StaticBuilder[T]) WithStrategy(s StaticStrategy) *StaticBuilder[T] {
	b.strategy = s
	return b
}

func (b *StaticBuilder[T]) WithCallback(c func(*T, *error)) *StaticBuilder[T] {
	b.callback = c
	return b
}

func (b *StaticBuilder[T]) Build() *static[T] {

	j := make([]Processable[T], 0)

	return &static[T]{
		log:           b.log,
		strategy:      b.strategy,
		jobs:          j,
		wg:            &sync.WaitGroup{},
		mu:            &sync.Mutex{},
		callback:      b.callback,
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
func (r *static[T]) AddJob(j Processable[T]) error {
	if j == nil {
		return ErrInvalidJobStatus // or return an error based on your requirements
	}
	r.log.Debug("Adding job to runner")
	r.mu.Lock()
	r.jobs = append(r.jobs, j)
	r.mu.Unlock()
	r.incrementTotalJobs()
	return nil
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
func (r *static[T]) Run(ctx context.Context) {
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
	}
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
		res, err := j.Run(ctx)
		r.incrementCompletedJobs()
		if r.callback != nil {
			r.callback(&res, &err) // Invoke the callback
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
			res, err := j.Run(ctx)
			if r.callback != nil {
				r.callback(&res, &err) // Invoke the callback
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
			res, err := j.Run(ctx)
			r.incrementCompletedJobs()
			if err != nil {
				failed = true // Mark as failed if an error occurs
			}
			if r.callback != nil {
				r.callback(&res, &err) // Invoke the callback
			}
		} else {
			if r.callback != nil {
				e := ErrJobCancelled
				r.callback(nil, &e.Err) // Invoke the callback
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

	pq := &PriorityQueue[T]{}
	// Populate the priority queue with jobs
	for _, j := range r.jobs {
		if j == nil {
			continue // Skip nil jobs
		}
		// Ensure the job implements the Priority interface
		if priorityJob, ok := j.(*PriorityJob[T]); ok {
			// Add to priority queue
			pq.Push(priorityJob)
		} else {
			// Fallback to adding as a normal job if it doesn't implement Priority
			pj, _ := NewPriorityJob(j, 0)
			pq.Push(pj)
		}
		r.incrementTotalJobs()
	}

	// Execute jobs based on priority
	for pq.Len() > 0 {
		// Pop the highest priority job
		job := pq.Pop()

		job.Run(ctx) // Execute the job
		r.incrementCompletedJobs()
	}

}

// incrementCompletedJobs increments the count of completed jobs in a thread-safe manner.
//
// Notes:
// - This method uses a mutex to ensure safe access to the completedJobs counter.
func (r *static[T]) incrementCompletedJobs() {
	r.log.Debug("Incrementing completed jobs")
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completedJobs++
}

// incrementTotalJobs increments the count of total jobs in a thread-safe manner.
//
// Notes:
// - This method uses a mutex to ensure safe access to the totalJobs counter.
func (r *static[T]) incrementTotalJobs() {
	r.log.Debug("Incrementing total jobs")
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalJobs++
}
