package runner

import (
	"context"
	"sync"

	. "GOTAS/internal/job"
)

var _ Runnable[any] = (*dynamic[any])(nil) // Ensure runner implements Runnable

// static represents the job static that executes jobs based on a strategy.
type dynamic[T any] struct {
	strategy DynamicStrategy       // strategy is the execution strategy for the runner
	jobs     []Processable[T]      // jobs is the list of jobs to be executed
	wg       *sync.WaitGroup       // wg is the wait group for tracking job completion
	mu       *sync.Mutex           // mu is the mutex for updating job status
	callback func(*Processable[T]) // callback function to be invoked after each job execution, can be nil

	completedJobs int // completedJobs is the number of jobs completed. complete happens regardless of error
	totalJobs     int // totalJobs is the total number of jobs in the runner
}

// NewDynamicRunner creates a new job runner with the specified execution strategy and an optional callback function.
func NewDynamicRunner[T any](strategy DynamicStrategy, callback func(*Processable[T])) *dynamic[T] {
	return &dynamic[T]{
		strategy:      strategy,
		jobs:          make([]Processable[T], 0),
		wg:            &sync.WaitGroup{},
		mu:            &sync.Mutex{},
		callback:      callback,
		completedJobs: 0,
		totalJobs:     0,
	}
}

// AddJob implements Runnable.
func (r *dynamic[T]) AddJob(j Processable[T]) {
	panic("unimplemented")
}

// CheckProgress implements Runnable.
func (r *dynamic[T]) CheckProgress() float64 {
	panic("unimplemented")
}

// Run implements Runnable.
func (r *dynamic[T]) Run(ctx context.Context) error {
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
		r.runRetry(ctx)
	default:
		return ErrInvalidRunnerStrategy
	}
	// Return nil to indicate successful execution of all jobs
	return nil
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

func (r *dynamic[T]) runParallel(ctx context.Context) {
	panic("unimplemented")
}

func (r *dynamic[T]) runSequential(ctx context.Context) {
	panic("unimplemented")
}
