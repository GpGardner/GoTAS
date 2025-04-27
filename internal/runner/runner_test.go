package runner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	. "GOTAS/internal/job"
)

type Result struct {
	ID      int
	Message string
}

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

			t.Log(j.String())
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

	go func() {
		for progress := range progressChan {
			t.Logf("Progress update: %f%%", progress)
		}
	}()

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

	for _, job := range jobs {
		t.Log(job.String())
	}
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

		t.Log(job.String())
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

		t.Log(job.String())
	}
}

func TestDynamicRunner(t *testing.T) {
	totalJobs := 100
	workers := 4
	chanSize := 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewDynamicRunner[Result](StrategyParallel{}, nil, workers, chanSize)
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
	runner.Shutdown()

	// Verify all jobs were processed
	for _, job := range jobs {
		t.Log(job.String())
		if !job.GetStatus().IsCompleted() {
			t.Errorf("Job %d was not completed", job.GetResult().ID)
		}
	}
}

func TestDynamicRunnerCancel(t *testing.T) {
	totalJobs := 1000
	workers := 100
	chanSize := 5000
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

	runner := NewDynamicRunner[Result](StrategyParallel{}, callback, workers, chanSize)
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
	runner.Shutdown()

	// Check which jobs were executed successfully
	for _, job := range jobs {
		t.Log(job.String())
	}

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

	jobsExpectToPass := totalJobs - cancelThreshold
	t.Log("Number of jobs failed: ", jobFailed)
	t.Log("Number of jobs expected to pass: ", jobsExpectToPass)

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

func TestDynamicRunnerFailFast(t *testing.T) {

	totalJobs := 200
	workers := 200
	chanSize := 100
	cancelThreshold := 15

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Counter to track completed jobs
	var completedJobs int
	var mu sync.Mutex // Protect access to the counter

	runner := NewDynamicRunner[Result](StrategyFailFast{}, nil, workers, chanSize)
	runner.Run(ctx)

	jobs := make([]*Job[Result], totalJobs)

	// Create jobs
	for i := 0; i < totalJobs; i++ {
		jobID := i + 1
		job, err := NewJob(func(ctx context.Context, args ...any) (Result, error) {
			var shouldFail bool
			mu.Lock()
			if completedJobs >= cancelThreshold {
				shouldFail = true
			} else {
				completedJobs++
			}
			mu.Unlock()

			if shouldFail {
				// Simulate a failure after the threshold is reached
				return Result{}, fmt.Errorf("job %d simulated error", jobID)
			}

			// Simulate successful job completion
			return Result{
				ID:      jobID,
				Message: fmt.Sprintf("Job %d completed successfully", jobID),
			}, nil
		})
		if err != nil {
			t.Fatalf("Failed to create job %d: %v", jobID, err)
		}
		jobs[i] = job
		runner.AddJob(job)
	}

	// Shutdown the runner and wait for all jobs to complete
	runner.Shutdown()

	// Check which jobs were executed successfully
	for _, job := range jobs {
		t.Log(job.String())
	}

	jobFailed := 0

	for _, job := range jobs {
		if job.GetStatus() != StatusCompleted {
			// add to the count of failed jobs
			jobFailed++
		}
	}

	jobsExpectToPass := totalJobs - cancelThreshold
	t.Log("Number of jobs failed: ", jobFailed)
	t.Log("Number of jobs expected: ", jobsExpectToPass)

	// Verify that the number of executed jobs is within an acceptable range
	if jobFailed != totalJobs-(cancelThreshold) {

		t.Errorf("Unexpected number of jobs failed: %d/%d", jobFailed, totalJobs)
	}

	// Verify that canceled jobs returned the correct error
	for i, job := range jobs {
		if job.GetStatus() == StatusError && job.GetError() != ctx.Err() {
			t.Logf("Job %d did not return the expected context cancellation error: got %v, expected %v", i+1, job.GetError(), ctx.Err())
		}
	}
}
