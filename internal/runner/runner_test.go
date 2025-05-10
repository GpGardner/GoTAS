package runner

import (
	"context"
	"fmt"

	// "strconv"

	"log"
	"runtime"
	"sync"
	"testing"
	"time"

	. "GOTAS/internal/job"
	logger "GOTAS/internal/log"
)

type Result struct {
	ID      int
	Message string
}

var l = logger.NewDefaultLogger(logger.LogLevelWarn)

func TestRunnerParallelExecution(t *testing.T) {

	numJobs := 3 // Number of jobs to run concurrently

	ctx := context.Background()
	runner := NewStaticRunner[Result](StrategyParallel{}, nil)

	var wg sync.WaitGroup
	wg.Add(3) // Expect all 3 jobs to run concurrently

	jobs := make([]*Job[Result], numJobs)

	for i := 0; i < numJobs; i++ {

		// Use NewJob to create each job
		job, err := NewJob(func(ctx context.Context, args ...any) (Result, error) {
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
			log.Fatalf("Failed to create job %d, %e", i, err)
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
		for i, j := range jobs {
			if j.GetStatus() != StatusCompleted {
				t.Errorf("Job %d did not complete successfully", i)
			}
			result := j.GetResult()
			if result.Message != fmt.Sprintf("Job %d completed successfully", i) {
				t.Errorf("Job %d result message mismatch: expected '%s', got '%s'", i, fmt.Sprintf("Job %d completed successfully", i), result.Message)
			}
			if j.GetDuration() < 100*time.Millisecond {
				t.Errorf("Job %d duration is less than expected: %v", i, j.GetDuration())
			}
			if j.GetDuration() > 200*time.Millisecond {
				t.Errorf("Job %d duration is more than expected: %v", i, j.GetDuration())
			}
			if j.GetError() != nil {
				t.Errorf("Job %d encountered an error: %v", i, j.GetError())
			}
			if j.GetStatus() != StatusCompleted {
				t.Errorf("Job %d status is not completed: %v", i, j.GetStatus())
			}

			if j.GetResult() == (Result{}) {
				t.Errorf("Job %d result is nil", i)
			}

			// // t.log(j.String())
		}

	case <-time.After(5 * time.Second):
		t.Errorf("Jobs did not run in parallel")
	}
}

func TestRunnerProgress(t *testing.T) {
	ctx := context.Background()
	runner := NewStaticRunner[Result](StrategySequential{}, nil)

	// Number of jobs
	numJobs := 3

	// Create jobs
	jobs := make([]*Job[Result], numJobs)
	progressChan := make(chan float64, numJobs) // Channel to track progress updates

	for i := 0; i < numJobs; i++ {
		jobID := i + 1 // Assign a unique ID to each job

		job, err := NewJob(func(ctx context.Context, args ...any) (Result, error) {
			// Simulate work
			select {
			case <-ctx.Done():
				// Handle context cancellation
				return Result{}, ctx.Err()
			case <-time.After(100 * time.Millisecond): // Simulate work delay
				// Send progress update
				progressChan <- float64(jobID) / float64(numJobs) * 100
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

	// Run the jobs in a separate goroutine
	go runner.Run(ctx)

	// Verify sequential execution and check progress after each job completes
	var wg sync.WaitGroup
	wg.Add(numJobs)

	// go func() {
	// 	for progress := range progressChan {
	// 		// t.logf("Progress update: %f%%", progress)
	// 	}
	// }()

	for i, job := range jobs {
		go func(i int, job *Job[Result]) {
			defer wg.Done()

			// Wait for the job to complete
			for job.GetStatus() != StatusCompleted {
				time.Sleep(10 * time.Millisecond) // Polling interval
			}

			// Verify job completion
			if job.GetStatus() != StatusCompleted {
				t.Errorf("Job %d did not complete successfully", job.GetResult().ID)
			}

			// Verify progress
			progress := runner.CheckProgress()
			expectedProgress := float64((i + 1) * 100 / numJobs)
			if progress < expectedProgress-1 || progress > expectedProgress+1 {
				t.Errorf("Progress is out of bounds after job %d: got %f, expected ~%f", job.GetResult().ID, progress, expectedProgress)
			}
		}(i, job)
	}

	// Wait for all jobs to complete
	wg.Wait()
	close(progressChan)

	// Verify final progress
	finalProgress := runner.CheckProgress()
	if finalProgress != 100.0 {
		t.Errorf("Expected progress to be 100.0, got %f", finalProgress)
	}

	// for _, job := range jobs {
	// 	// t.log(job.String())
	// }
}

func TestRunnerSequentialExecution(t *testing.T) {
	ctx := context.Background()
	runner := NewStaticRunner[Result](StrategySequential{}, nil)

	// Number of jobs
	numJobs := 25

	// Create jobs
	jobs := make([]*Job[Result], numJobs)
	executionOrder := make([]int, 0, numJobs) // Track the order of execution
	var mu sync.Mutex                         // Protect access to executionOrder

	for i := 0; i < numJobs; i++ {
		jobID := i + 1 // Assign a unique ID to each job

		job, err := NewJob(func(ctx context.Context, args ...any) (Result, error) {
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
	for _, job := range jobs {
		for job.GetStatus() != StatusCompleted {
			time.Sleep(10 * time.Millisecond) // Polling interval
		}
	}

	// Verify that jobs were executed in order
	for i, jobID := range executionOrder {
		if jobID != i+1 {
			t.Errorf("Jobs executed out of order: expected Job %d, got Job %d", i+1, jobID)
		}
	}

	// Verify that all jobs completed successfully
	for i, job := range jobs {
		if job.GetStatus() != StatusCompleted {
			t.Errorf("Job %d did not complete successfully", i+1)
		}

		result := job.GetResult()
		if result.ID != i+1 || result.Message != fmt.Sprintf("Job %d completed successfully", i+1) {
			t.Errorf("Job %d result mismatch: got %+v", i+1, result)
		}

		// // t.log(job.String())
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := NewStaticRunner[Result](StrategyParallel{}, nil)

	// Number of jobs
	numJobs := 3

	// Create jobs
	jobs := make([]*Job[Result], numJobs)
	for i := 0; i < numJobs; i++ {
		jobID := i + 1 // Assign a unique ID to each job

		job, err := NewJob(func(ctx context.Context, args ...any) (Result, error) {
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
	for i, job := range jobs {
		if job.GetStatus() == StatusCompleted {
			t.Errorf("Job %d was executed despite context cancellation", i+1)
		}

		if job.GetError() != ctx.Err() {
			t.Errorf("Job %d did not return the expected context cancellation error: got %v, expected %v", i+1, job.GetError(), ctx.Err())
		}

		// // t.log(job.String())
	}
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
		job, err := NewJob(func(ctx context.Context, args ...any) (Result, error) {
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

	// Verify all jobs were processed
	for _, job := range jobs {

		if !job.GetStatus().IsCompleted() {
			t.Errorf("\nJob %d was not completed\n%s", job.GetID(), job.String())
		} else {
			// // t.log(job.String())
		}
	}
}

func TestDynamicRunnerCancel(t *testing.T) {
	totalJobs := 1000
	workers := 100
	chanSize := 250
	cancelThreshold := 500

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Counter to track completed jobs
	var completedJobs int
	var mu sync.Mutex // Protect access to the counter

	// Define the callback function
	callback := func(job Processable[Result]) {
		mu.Lock()
		defer mu.Unlock()

		completedJobs++
		if completedJobs >= cancelThreshold {
			cancel() // Cancel the context once the threshold is reached
		}
	}

	runner := NewRunnerBuilder[Result]().WithStrategy(StrategyParallel{}).WithWorkers(workers).WithChanSize(chanSize).WithLogger(l).WithCallback(callback).Build()
	runner.Run(ctx)

	jobs := make([]*Job[Result], totalJobs)

	// Create jobs
	for i := 0; i < totalJobs; i++ {
		jobID := i + 1
		job, err := NewJob(func(ctx context.Context, args ...any) (Result, error) {
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
		runner.AddJob(job)
	}

	// Shutdown the runner and wait for all jobs to complete
	runner.ShutdownGracefully(cancel)

	// // Check which jobs were executed successfully
	// for _, job := range jobs {
	// 	// t.log(job.String())
	// }

	totalSuccess := 0
	for _, job := range jobs {
		if job.GetStatus() == StatusCompleted {
			// add to the count of failed jobs
			totalSuccess++
		}
	}

	jobFailed := 0

	for _, job := range jobs {
		if job.GetStatus() != StatusCompleted {
			// add to the count of failed jobs
			jobFailed++
		}
	}

	// jobsExpectToPass := totalJobs - cancelThreshold
	// t.log("Number of jobs failed: ", jobFailed)
	// t.log("Number of jobs expected to pass: ", jobsExpectToPass)

	// Verify that the number of executed jobs is within an acceptable range
	if jobFailed != totalJobs-(cancelThreshold) {

		t.Errorf("Unexpected number of jobs failed: %d/%d", jobFailed, totalJobs)
	}

	// Verify that canceled jobs returned the correct error
	for i, job := range jobs {
		if job.GetStatus() == StatusError && job.GetError() != ctx.Err() {
			t.Errorf("Job %d did not return the expected context cancellation error: got %v, expected %v", i+1, job.GetError(), ctx.Err())
		}
	}
}

// TestDynamicRunnerFailFast tests the fail-fast strategy in a dynamic runner.
// It simulates a scenario where jobs are added dynamically, and the runner cancels
// TODO: Currently leaving jobs in pending state, need to fix this
// func TestDynamicRunnerFailFast(t *testing.T) {

// 	totalJobs := 6
// 	workers := 1
// 	chanSize := 3
// 	cancelThreshold := 2

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	// Counter to track completed jobs
// 	var completedJobs int
// 	var mu sync.RWMutex // Protect access to the counter

// 	runner := NewDynamicRunner[Result](StrategyFailFast{}, nil, workers, chanSize)
// 	runner.Run(ctx)

// 	jobs := make([]*Job[Result], totalJobs)

// 	// Create jobs
// 	for i := 0; i < totalJobs; i++ {
// 		jobID := i + 1
// 		job, err := NewJob(func(ctx context.Context, args ...any) (Result, error) {
// 			var shouldFail bool
// 			// Use a read lock to check the counter
// 			mu.RLock()
// 			if completedJobs >= cancelThreshold {
// 				shouldFail = true
// 			}
// 			mu.RUnlock()

// 			if shouldFail {
// 				// Simulate a failure after the threshold is reached
// 				return Result{}, fmt.Errorf("job %d simulated error", jobID)
// 			}

// 			// Use a write lock to increment the counter
// 			mu.Lock()
// 			completedJobs++
// 			mu.Unlock()

// 			// Simulate successful job completion
// 			return Result{
// 				ID:      jobID,
// 				Message: fmt.Sprintf("Job %d completed successfully", jobID),
// 			}, nil
// 		})
// 		if err != nil {
// 			t.Fatalf("Failed to create job %d: %v", jobID, err)
// 		}
// 		jobs[i] = job
// 		runner.AddJob(job)
// 	}

// 	// Shutdown the runner and wait for all jobs to complete
// 	runner.ShutdownGracefully(cancel)

// 	// Check which jobs were executed successfully
// 	// for _, job := range jobs {
// 	// 	// t.log(job.String())
// 	// }

// 	jobFailed := 0

// 	for _, job := range jobs {
// 		if job.GetStatus() != StatusCompleted && job.GetStatus() != StatusPending {
// 			// add to the count of failed jobs
// 			jobFailed++
// 		}
// 	}

// 	jobsExpectToPass := totalJobs - cancelThreshold
// 	// t.log("Number of jobs failed: ", jobFailed)
// 	// t.log("Number of jobs expected to pass: ", jobsExpectToPass)

// 	// Verify that the number of executed jobs is within an acceptable range
// 	if jobFailed != totalJobs-(cancelThreshold) {
// 		t.Errorf("Unexpected number of jobs failed: %d/%d", jobFailed, totalJobs)
// 	}

// 	// Verify that correct canceled job returned the simulated error should be cancelthreshold +1
// 	for i, job := range jobs {
// 		if job.GetStatus() == StatusError && job.GetError() != ctx.Err() {
// 			// t.logf("Job %d did not return the expected context cancellation error: got %v, expected %v", i+1, job.GetError(), StatusError)
// 		}
// 		if job.GetStatus() != StatusCompleted && job.GetStatus() != StatusError && job.GetStatus() != StatusTimeout {
// 			// // t.log(job.String())
// 			t.Errorf("Job %d has an unexpected status got %v", i+1, job.GetStatus())
// 		}
// 	}
// }

func TestRunnerGracefulShutdownTimeout(t *testing.T) {
	totalJobs := 5
	workers := 3
	chanSize := 3

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callback := func(job Processable[Result]) {
		// t.logf("Callback invoked for job: %v, Status: %v", job, job.GetStatus())
	}

	runner := NewRunnerBuilder[Result]().WithStrategy(StrategyParallel{}).WithWorkers(workers).WithChanSize(chanSize).WithLogger(l).WithCallback(callback).Build()
	runner.Run(ctx)

	jobs := make([]*Job[Result], totalJobs)

	// Create jobs with long execution times
	for i := 0; i < totalJobs; i++ {
		jobID := i + 1
		job, err := NewJob(func(ctx context.Context, args ...any) (Result, error) {
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
	for i, job := range jobs {
		// // t.log("\n")
		// // t.log(job.String())
		if job.GetStatus() == StatusPending {
			t.Errorf("Job %d is still pending", i)
		}
	}
}

func BenchmarkRunnerMemoryUsage(b *testing.B) {
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)
	fmt.Printf("Memory Before: Alloc = %v KB, TotalAlloc = %v KB\n",
		memStatsBefore.Alloc/1024, memStatsBefore.TotalAlloc/1024)
	// Configure the runner
	r := NewRunnerBuilder[Result]().
		WithStrategy(StrategyParallel{}).
		WithWorkers(100).
		WithChanSize(100).
		WithMaxWaitForClose(20 * time.Second).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the runner
	r.Run(ctx)

	// Add jobs
	for i := 0; i < 10000; i++ {

		// Measure memory usage before adding jobs
    var memStatsDuring runtime.MemStats
	runtime.ReadMemStats(&memStatsDuring)
	fmt.Printf("Memory During: Alloc = %v KB, TotalAlloc = %v KB\n",
		memStatsDuring.Alloc/1024, memStatsBefore.TotalAlloc/1024)

		job, _ := NewJob(func(ctx context.Context, args ...any) (Result, error) {
			time.Sleep(10 * time.Millisecond) // Simulate job processing
			return Result{ID: i, Message: "Job completed"}, nil
		})
		r.AddJob(job)

	}
	// Shutdown the runner
	r.ShutdownGracefully(cancel)

	// Measure memory usage after adding jobs
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)
	fmt.Printf("Memory After: Alloc = %v KB, TotalAlloc = %v KB\n",
		memStatsAfter.Alloc/1024, memStatsAfter.TotalAlloc/1024)
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
		job, _ := NewJob(func(ctx context.Context, args ...any) (Result, error) {
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
		WithWorkers(500).
		WithChanSize(2000).
		WithMaxWaitForClose(30 * time.Second).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the runner
	r.Run(ctx)

	// Add a large number of jobs
	numJobs := 20000
	jobs := make([]*Job[Result], numJobs)
	for i := 0; i < numJobs; i++ {
		job, _ := NewJob(func(ctx context.Context, args ...any) (Result, error) {
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
	for _, job := range jobs {
		if job.GetStatus() != StatusCompleted {
			// // t.log(job.String())
			t.Errorf("Job %d did not complete successfully", job.GetID())
		}
	}
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
		job, _ := NewJob(func(ctx context.Context, args ...any) (Result, error) {
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
		// t.logf("Progress: %.2f%%", progress)
		if progress < 0 || progress > 100 {
			t.Errorf("Invalid progress value: %.2f%%", progress)
		}
	}

	// Shutdown the runner
	r.ShutdownGracefully(cancel)
}
