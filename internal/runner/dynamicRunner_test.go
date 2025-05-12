package runner

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	. "GOTAS/internal/job"
	logger "GOTAS/internal/log"
)

type Result struct {
	ID      int
	Data    *[1024]byte // Padding to simulate a larger struct
	Message string
}

var l = logger.NewDefaultLogger(logger.LogLevelWarn)

func TestRunnerParallelExecution(t *testing.T) {

	numJobs := 3 // Number of jobs to run concurrently

	ctx := context.Background()
	runner := NewRunnerBuilder[Result]().
		WithStrategy(StrategyParallel{}).
		WithLogger(l).
		Build()

	var wg sync.WaitGroup
	wg.Add(3) // Expect all 3 jobs to run concurrently

	jobs := make([]*Job[Result], numJobs)

	for i := 0; i < numJobs; i++ {

		// Use Create to create each job
		job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
			// Simulate work
			select {
			case <-ctx.Done():
				// Handle context cancellation
				return Result{}, ctx.Err()
			case <-time.After(100 * time.Millisecond): // Simulate work delay
				wg.Done() // Decrement the WaitGroup counter
				// Return a successful result
				return Result{
					ID:      i,
					Message: fmt.Sprintf("Job %d completed successfully", i),
				}, nil
			}
		})

		if err != nil {
			t.Fatalf("Failed to create job %d, %e", i, err)
		}

		jobs[i] = job
	}

	// Add jobs to the runner
	for _, val := range jobs {
		runner.AddJob(val)
	}

	// Run the jobs
	go runner.Run(ctx)

	// Wait for all jobs to start
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All jobs ran concurrently
	case <-time.After(5 * time.Second):
		t.Errorf("Jobs did not run in parallel")
	}
}

func TestRunnerProgress(t *testing.T) {
	t.Skip("Skipping until progress is implemented")
}

func TestRunnerSequentialExecution(t *testing.T) {
	ctx := context.Background()
	runner := NewRunnerBuilder[Result]().
		WithStrategy(StrategySequential{}).
		WithLogger(l).
		Build()

	// Number of jobs
	numJobs := 25

	// Create jobs
	jobs := make([]*Job[Result], numJobs)
	executionOrder := make([]int, 0, numJobs) // Track the order of execution
	var mu sync.Mutex                         // Protect access to executionOrder

	for i := 0; i < numJobs; i++ {
		jobID := i + 1 // Assign a unique ID to each job

		job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
			// Simulate work
			select {
			case <-ctx.Done():
				// Handle context cancellation
				return Result{}, ctx.Err()
			case <-time.After(100 * time.Millisecond): // Simulate work delay
				// Record the execution order
				mu.Lock()
				executionOrder = append(executionOrder, jobID)
				mu.Unlock()

				// Return a successful result
				return Result{
					ID:      jobID,
					Message: fmt.Sprintf("Job %d completed successfully", jobID),
				}, nil
			}
		})

		if err != nil {
			t.Fatalf("Failed to create job %d: %v", jobID, err)
		}

		jobs[i] = job
	}

	// Add jobs to the runner
	for _, job := range jobs {
		runner.AddJob(job)
	}

	// Run the jobs
	go runner.Run(ctx)

	// Wait for all jobs to complete
	// for _, job := range jobs {
	// 	for job.GetStatus() != StatusCompleted {
	// 		time.Sleep(10 * time.Millisecond) // Polling interval
	// 	}
	// }

	// Verify that jobs were executed in order
	for i, jobID := range executionOrder {
		if jobID != i+1 {
			t.Errorf("Jobs executed out of order: expected Job %d, got Job %d", i+1, jobID)
		}
	}

	// Verify that all jobs completed successfully
	// for i, job := range jobs {
	// 	if job.GetStatus() != StatusCompleted {
	// 		t.Errorf("Job %d did not complete successfully", i+1)
	// 	}

	// 	result := job.GetResult()
	// 	if result.ID != i+1 || result.Message != fmt.Sprintf("Job %d completed successfully", i+1) {
	// 		t.Errorf("Job %d result mismatch: got %+v", i+1, result)
	// 	}

	// 	// // t.log(job.String())
	// }
}

func TestRunnerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := NewRunnerBuilder[Result]().
		WithStrategy(StrategyParallel{}).
		WithLogger(l).
		Build()

	// Number of jobs
	numJobs := 3

	// Create jobs
	jobs := make([]*Job[Result], numJobs)
	for i := 0; i < numJobs; i++ {
		jobID := i + 1 // Assign a unique ID to each job

		job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
			// Simulate work
			select {
			case <-ctx.Done():
				// Handle context cancellation
				return Result{}, ctx.Err()
			case <-time.After(1000 * time.Millisecond): // Simulate work delay
				// Return a successful result
				return Result{
					ID:      jobID,
					Message: fmt.Sprintf("Job %d completed successfully", jobID),
				}, nil
			}
		})

		if err != nil {
			t.Fatalf("Failed to create job %d: %v", jobID, err)
		}

		jobs[i] = job
	}

	// Add jobs to the runner
	for _, job := range jobs {
		runner.AddJob(job)
	}

	// Cancel the context before running
	cancel()

	// Run the jobs
	runner.Run(ctx)

	// Verify that no jobs were executed
	// for i, job := range jobs {
	// 	if job.GetStatus() == StatusCompleted {
	// 		t.Errorf("Job %d was executed despite context cancellation", i+1)
	// 	}

	// 	if job.GetError() != ctx.Err() {
	// 		t.Errorf("Job %d did not return the expected context cancellation error: got %v, expected %v", i+1, job.GetError(), ctx.Err())
	// 	}

	// 	// // t.log(job.String())
	// }
}

func TestDynamicRunner(t *testing.T) {
	totalJobs := 100
	workers := 100
	chanSize := 100

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewRunnerBuilder[Result]().WithStrategy(StrategyParallel{}).WithWorkers(workers).WithChanSize(chanSize).WithLogger(l).Build()
	runner.Run(ctx)

	jobs := make([]*Job[Result], totalJobs)

	// Create jobs
	for i := 0; i < totalJobs; i++ {
		jobID := i + 1
		job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
			select {
			case <-ctx.Done():
				return Result{}, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return Result{
					ID:      jobID,
					Message: fmt.Sprintf("Job %d completed successfully", jobID),
				}, nil
			}
		})
		if err != nil {
			t.Fatalf("Failed to create job %d: %v", jobID, err)
		}
		jobs[i] = job
		jobErr := runner.AddJob(job)
		if jobErr != nil {
			t.Fatalf("Failed to add job %d: %v", jobID, jobErr)
		}
	}

	// Shutdown the runner and wait for all jobs to complete
	runner.ShutdownGracefully(cancel)
}

// This test is not deterministic
func TestDynamicRunnerContextCancellation(t *testing.T) {
	totalJobs := 10      // Total number of jobs to process
	cancelThreshold := 5 // Number of jobs to process before canceling the context
	workers := 3         // Number of workers processing jobs
	chanSize := 10       // Size of the job queue

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure the context is canceled at the end of the test

	// Counters for successes and failures
	var successCount, failureCount int
	var mu sync.Mutex // Protect access to the counters

	// Define a callback function to update job results
	callback := func(result *Result, err *error) {
		l.Debug(fmt.Sprintf("Callback triggered for %d", result.ID))
		mu.Lock()
		defer mu.Unlock()
		if *err != nil {
			failureCount++
		} else {
			l.Debug(fmt.Sprintf("Success Result - %+v", result))
			successCount++
		}
	}

	// Build the runner with the callback
	runner := NewRunnerBuilder[Result]().
		WithStrategy(StrategyParallel{}). // Use parallel strategy
		WithWorkers(workers).             // Set the number of workers
		WithChanSize(chanSize).           // Set the channel size
		WithLogger(l).                    // Use the logger
		WithCallback(callback).           // Attach the callback
		Build()

	// Start the runner
	runner.Run(ctx)

	// Create and add jobs to the runner
	for i := 0; i < totalJobs; i++ {
		jobID := i + 1 // Assign a unique ID to each job

		// Create a job with simulated work
		job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
			select {
			case <-ctx.Done():
				// If the context is canceled, return a failure result
				return Result{}, ctx.Err()
			case <-time.After(100 * time.Millisecond): // Simulate work delay
				if jobID == cancelThreshold {
					// Cancel the context when the threshold is reached
					cancel()
				}
				// Simulate a successful job
				return Result{
					ID:      jobID,
					Message: fmt.Sprintf("Job %d completed successfully", jobID),
				}, nil
			}
		})

		if err != nil {
			t.Fatalf("Failed to create job %d: %v", jobID, err)
		}

		// Add the job to the runner
		runner.AddJob(job)
	}

	// Gracefully shut down the runner
	runner.ShutdownGracefully(cancel)

	// Log the results
	t.Logf("Total jobs: %d, Successes: %d, Failures: %d", totalJobs, successCount, failureCount)

	// Define expected results
	expectedFailures := totalJobs - cancelThreshold
	expectedSuccesses := cancelThreshold

	// Validate the results
	if successCount != expectedSuccesses {
		t.Errorf("Unexpected number of successful jobs: got %d, want %d", successCount, expectedSuccesses)
	}
	if failureCount != expectedFailures {
		t.Errorf("Unexpected number of failed jobs: got %d, want %d", failureCount, expectedFailures)
	}
}

func TestDynamicRunnerFailFast(t *testing.T) {
	totalJobs := 100   // Total number of jobs to process
	failThreshold := 5 // Job ID at which the first failure occurs
	workers := 3       // Number of workers processing jobs
	chanSize := 10     // Size of the job queue

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure the context is canceled at the end of the test

	// Counters for successes and failures
	var successCount, failureCount int
	var mu sync.Mutex // Protect access to the counters

	// Channel to control job completion
	jobControl := make(chan struct{}, 1)

	// Define a callback function to update job results
	callback := func(result *Result, err *error) {
		l.Debug(fmt.Sprintf("Callback triggered for %d", result.ID))
		mu.Lock()
		defer mu.Unlock()
		if *err != nil {
			failureCount++
		} else {
			l.Debug(fmt.Sprintf("Success Result - %+v", result))
			successCount++
		}
	}

	// Build the runner with the callback
	runner := NewRunnerBuilder[Result]().
		WithStrategy(StrategyFailFast{}). // Use fail-fast strategy
		WithWorkers(workers).             // Set the number of workers
		WithChanSize(chanSize).           // Set the channel size
		WithLogger(l).                    // Use the logger
		WithCallback(callback).           // Attach the callback
		Build()

		// Start the runner
	runner.Run(ctx)

	// Create jobs
	jobs := make([]*Job[Result], totalJobs)
	for i := 0; i < totalJobs; i++ {
		jobID := i + 1 // Assign a unique ID to each job

		// Create a job with controlled execution
		job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
			// Wait for permission to proceed
			select {
			case <-jobControl:
				// Simulate work
				time.Sleep(100 * time.Millisecond)

				// Fail the job if it reaches the fail threshold
				if jobID >= failThreshold {
					return Result{}, fmt.Errorf("job %d failed", jobID)
				}

				// Simulate a successful job
				return Result{
					ID:      jobID,
					Message: fmt.Sprintf("Job %d completed successfully", jobID),
				}, nil
			case <-ctx.Done():
				// Handle context cancellation
				return Result{}, ctx.Err()
			}
		})

		if err != nil {
			t.Fatalf("Failed to create job %d: %v", jobID, err)
		}

		jobs[i] = job
		runner.AddJob(job) // Add the job to the runner
	}

	// Allow jobs to complete one by one
	go func() {
		for i := 0; i < totalJobs; i++ {
			jobControl <- struct{}{}          // Allow the next job to proceed
			time.Sleep(50 * time.Millisecond) // Simulate delay between job completions
		}
		close(jobControl) // Close the channel after all jobs are processed
	}()

	// Gracefully shut down the runner
	runner.ShutdownGracefully(cancel)

	// Log the results
	t.Logf("Total jobs: %d, Successes: %d, Failures: %d", totalJobs, successCount, failureCount)

	// Define expected results
	expectedFailures := totalJobs - failThreshold + 1
	expectedSuccesses := failThreshold - 1

	// Validate the results
	if successCount != expectedSuccesses {
		t.Errorf("Unexpected number of successful jobs: got %d, want %d", successCount, expectedSuccesses)
	}
	if failureCount != expectedFailures {
		t.Errorf("Unexpected number of failed jobs: got %d, want %d", failureCount, expectedFailures)
	}
}

func TestRunnerGracefulShutdownTimeout(t *testing.T) {
	totalJobs := 100
	workers := 3
	chanSize := 3

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callback := func(r *Result, err *error) {
		t.Logf("Callback invoked for job: %v, Status: %v", r.ID, *err)
	}

	runner := NewRunnerBuilder[Result]().WithStrategy(StrategyParallel{}).WithWorkers(workers).WithChanSize(chanSize).WithLogger(l).WithCallback(callback).Build()
	runner.Run(ctx)

	jobs := make([]*Job[Result], totalJobs)

	// Create jobs with long execution times
	for i := 0; i < totalJobs; i++ {
		jobID := i + 1
		job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
			select {
			case <-ctx.Done():
				return Result{}, ctx.Err()
			case <-time.After(1 * time.Second): // Simulate long-running job
				return Result{
					ID:      jobID,
					Message: fmt.Sprintf("Job %d completed successfully", jobID),
				}, nil
			}
		})
		if err != nil {
			t.Fatalf("Failed to create job %d: %v", jobID, err)
		}
		jobs[i] = job
		runner.AddJob(job)
	}

	// Attempt graceful shutdown with a timeout
	done := make(chan struct{})
	go func() {
		runner.ShutdownGracefully(cancel)
		close(done)
		// t.logf("Progress: %v percent", runner.CheckProgress())
	}()

	select {
	case <-done:
		// t.log("Graceful shutdown completed")
	case <-time.After(20 * time.Second): // Timeout shorter than job duration
		t.Error("Graceful shutdown ran out of time")
	}

	// Verify that some jobs are still pending
	// for i, job := range jobs {
	// 	// // t.log("\n")
	// 	// // t.log(job.String())
	// 	if job.GetStatus() == StatusPending {
	// 		t.Errorf("Job %d is still pending", i)
	// 	}
	// }
}

func BenchmarkRunnerMemoryUsage(b *testing.B) {
	totalJobs := 10000
	workers := 100
	chanSize := 100

	// Pre-create jobs to exclude job creation overhead from the benchmark
	jobs := make([]*Job[Result], totalJobs)
	for j := 0; j < totalJobs; j++ {
		job, _ := Create(func(ctx context.Context, args ...any) (Result, error) {
			// Simulate job processing
			return Result{ID: j, Message: "Job completed"}, nil
		})
		jobs[j] = job
	}

	// Measure memory usage before the benchmark starts
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)
	b.Logf("Memory Before: Alloc = %v KB, TotalAlloc = %v KB",
		memStatsBefore.Alloc/1024, memStatsBefore.TotalAlloc/1024)

	// Configure the runner
	r := NewRunnerBuilder[Result]().
		WithStrategy(StrategyParallel{}).
		WithWorkers(workers).
		WithChanSize(chanSize).
		WithMaxWaitForClose(20 * time.Second).
		// WithLogger(l).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the runner
	r.Run(ctx)

	// Reset the timer to exclude setup time
	b.ResetTimer()

	// Benchmark loop
	for i := 0; i < b.N; i++ {
		// Add pre-created jobs
		for _, job := range jobs {
			r.AddJob(job)
		}
	}

	// Stop the timer before shutdown
	b.StopTimer()

	r.ShutdownGracefully(cancel)

	// Measure memory usage after the benchmark ends
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)
	b.Logf("Memory After: Alloc = %v KB, TotalAlloc = %v KB",
		memStatsAfter.Alloc/1024, memStatsAfter.TotalAlloc/1024)

	// Validate memory usage
	if memStatsAfter.Alloc > memStatsBefore.Alloc*2 {
		b.Errorf("Potential memory leak detected: Alloc increased from %v KB to %v KB",
			memStatsBefore.Alloc/1024, memStatsAfter.Alloc/1024)
	}
}

func TestRunnerMemoryLeak(t *testing.T) {
	// Configure the runner
	r := NewRunnerBuilder[Result]().
		WithStrategy(StrategyParallel{}).
		WithWorkers(50).
		WithChanSize(500).
		WithMaxWaitForClose(10 * time.Second).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the runner
	r.Run(ctx)

	// Measure memory usage before adding jobs
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)
	t.Logf("Memory Before: Alloc = %v KB, TotalAlloc = %v KB",
		memStatsBefore.Alloc/1024, memStatsBefore.TotalAlloc/1024)

	// Add jobs
	for i := 0; i < 100; i++ {
		job, _ := Create(func(ctx context.Context, args ...any) (Result, error) {
			time.Sleep(5 * time.Millisecond) // Simulate job processing
			return Result{ID: i, Message: "Job completed"}, nil
		})
		r.AddJob(job)
	}

	// Shutdown the runner
	r.ShutdownGracefully(cancel)

	// Measure memory usage after adding jobs
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)
	t.Logf("Memory After: Alloc = %v KB, TotalAlloc = %v KB",
		memStatsAfter.Alloc/1024, memStatsAfter.TotalAlloc/1024)

	// Check for memory leaks
	if memStatsAfter.Alloc > memStatsBefore.Alloc*2 {
		t.Errorf("Potential memory leak detected: Alloc increased from %v KB to %v KB",
			memStatsBefore.Alloc/1024, memStatsAfter.Alloc/1024)
	}
}

func TestRunnerHighConcurrency(t *testing.T) {
	// Configure the runner
	r := NewRunnerBuilder[Result]().
		WithStrategy(StrategyParallel{}).
		WithWorkers(200).
		WithChanSize(10000).
		WithMaxWaitForClose(30 * time.Second).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the runner
	r.Run(ctx)

	// Add a large number of jobs
	numJobs := 500000
	jobs := make([]*Job[Result], numJobs)
	for i := 0; i < numJobs; i++ {
		job, _ := Create(func(ctx context.Context, args ...any) (Result, error) {
			time.Sleep(1 * time.Millisecond) // Simulate job processing
			return Result{ID: i, Message: "Job completed"}, nil
		})
		jobs[i] = job
		// // t.logf("Adding job: %v", job.GetID())
		r.AddJob(job)
	}

	// Shutdown the runner
	r.ShutdownGracefully(cancel)

	// Verify all jobs completed
	// for _, job := range jobs {
	// 	if job.GetStatus() != StatusCompleted {
	// 		// // t.log(job.String())
	// 		t.Errorf("Job %d did not complete successfully", job.GetID())
	// 	}
	// }
}

func TestRunnerProgressAccuracy(t *testing.T) {
	// Configure the runner
	r := NewRunnerBuilder[Result]().
		WithStrategy(StrategySequential{}).
		WithWorkers(1).
		WithChanSize(10).
		WithMaxWaitForClose(10 * time.Second).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the runner
	r.Run(ctx)

	// Add jobs
	numJobs := 10
	for i := 0; i < numJobs; i++ {
		job, _ := Create(func(ctx context.Context, args ...any) (Result, error) {
			time.Sleep(50 * time.Millisecond) // Simulate job processing
			return Result{ID: i, Message: "Job completed"}, nil
		})
		r.AddJob(job)
	}

	// Monitor progress
	progressChan := make(chan float64)
	go func() {
		for {
			progress := r.CheckProgress()
			progressChan <- progress
			if progress >= 100.0 {
				close(progressChan)
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Verify progress updates
	for progress := range progressChan {
		t.Logf("Progress: %.2f%%", progress)
		if progress < 0 || progress > 100 {
			t.Errorf("Invalid progress value: %.2f%%", progress)
		}
	}

	// Shutdown the runner
	r.ShutdownGracefully(cancel)
}
