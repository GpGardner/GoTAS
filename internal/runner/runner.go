package runner

import (
	"context"
	"math"
	"sync"

	. "GOTAS/internal/job"
)

type Runnable interface {
	Run(ctx context.Context)
	AddJob(j Job)
	CheckProgress() float64
}

// Runner represents the job runner that executes jobs based on a strategy.
type Runner struct {
	strategy Strategy        // strategy is the execution strategy for the runner
	jobs     []Job           // jobs is the list of jobs to be executed
	wg       *sync.WaitGroup // wg is the wait group for tracking job completion
	mu       *sync.Mutex     // mu is the mutex for updating job status

	completedJobs int // completedJobs is the number of jobs completed. complete happens regardless error
	totalJobs     int // totalJobs is the total number of jobs in the runner
}

// NewRunner creates a new job runner with the given strategy.
func NewRunner(strategy Strategy, callback func(*Job)) *Runner {
	return &Runner{
		strategy:      strategy,
		jobs:          make([]Job, 0),
		wg:            &sync.WaitGroup{},
		mu:            &sync.Mutex{},
		completedJobs: 0,
		totalJobs:     0,
	}
}

// AddJob adds a new job to the runner.
func (r *Runner) AddJob(j Job) {
	r.jobs = append(r.jobs, j)
	r.incrementTotalJobs()
}

func (r *Runner) CheckProgress() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Handle division by zero
	if r.totalJobs == 0 {
		return 0.0
	}

	return math.Round((float64(r.completedJobs) / float64(r.totalJobs)) * 100)
}

// Run executes all jobs in the runner based on the strategy.
func (r *Runner) Run(ctx context.Context) {
	switch r.strategy {
	case StrategySequential:
		r.runSequential(ctx)
	case StrategyParallel:
		r.runParallel(ctx)
	// case StrategyBatch:
	// 	r.runBatch(ctx)
	// case StrategyPriority:
	// 	r.runPriority(ctx)
	// case StrategyRetry:
	// 	r.runRetry(ctx)
	// case StrategyCron:
	// 	r.runCron(ctx)
	default:
		r.runSequential(ctx)
	}
}

func (r *Runner) runParallel(ctx context.Context) {
	for _, j := range r.jobs {
		r.wg.Add(1) // Increment WaitGroup counter
		go func() {
			defer r.wg.Done() // Ensure WaitGroup counter is decremented even if the job panics
			defer r.incrementCompletedJobs()
			j.Run(ctx)
		}()
	}
	r.wg.Wait() // Wait for all jobs to complete
}

// Issue with ctx, args ??
func (r *Runner) runSequential(ctx context.Context) {
	for _, j := range r.jobs {
		j.Run(ctx)
		r.incrementCompletedJobs()
	}
}

func (r *Runner) incrementCompletedJobs() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completedJobs++
}

func (r *Runner) incrementTotalJobs() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalJobs++
}
