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

// MockJob is a mock implementation of the Job interface for testing.
type MockJob[T any] struct {
	ID        int
	Executed  bool
	StartChan chan struct{}                              // Signal when the job starts
	EndChan   chan struct{}                              // Signal when the job ends
	function  func(ctx context.Context, job *MockJob[T]) // The function to execute for the job
}

func NewMockJob(ID int, StartChan chan struct{}, Endchan chan struct{}, f func(ctx context.Context, job *MockJob[any])) Processable[any] {
	return &MockJob[any]{
		ID:        ID,
		Executed:  false,
		StartChan: StartChan,
		EndChan:   Endchan,
		function:  f,
	}
}

// Ensure MockJob implements the Processable interface
// var _ Processable[any] = (*MockJob[any])(nil)

func (j *MockJob[T]) Run(ctx context.Context, args ...any) (T, error) {
	j.execute(ctx)
	var zero T
	return zero, nil
}

func (j *MockJob[T]) Complete(status Status) {
	j.Executed = true
}

func (j *MockJob[T]) GetStatus() Status {
	return StatusCompleted
}

func (j *MockJob[T]) GetError() error {
	return nil
}

func (j *MockJob[T]) GetResult() any {
	return nil
}

func (j *MockJob[T]) GetDuration() time.Duration {
	return 10 * time.Millisecond
}

func (j *MockJob[T]) CreatedAt() time.Time {
	return time.Now()
}

func (j *MockJob[T]) CompletedAt() time.Time {
	return time.Now()
}

func (j *MockJob[T]) execute(ctx context.Context) {
	// Create a select block to handle context cancellation

	select {
	case <-ctx.Done():
		j.Executed = false // Mark as not executed if context is done
		return
	default:
		// If not cancelled, execute the function
		j.function(ctx, j)
		j.Complete(StatusCompleted)
	}
}

func TestRunnerParallelExecution(t *testing.T) {
	ctx := context.Background()
	runner := NewStaticRunner[any](StrategyParallel{}, nil)

	var wg sync.WaitGroup
	wg.Add(3) // Expect all 3 jobs to run concurrently

	// Create mock jobs
	jobs := []*MockJob[any]{
		NewMockJob(1, make(chan struct{}), make(chan struct{}), func(ctx context.Context, job *MockJob[any]) {
			defer wg.Done() // Signal that this job is done
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			log.Printf("Job 1 executed")
		}).(*MockJob[any]), // Ensure type matches Processable[any]
		NewMockJob(2, make(chan struct{}), make(chan struct{}), func(ctx context.Context, job *MockJob[any]) {
			defer wg.Done() // Signal that this job is done
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			log.Printf("Job 2 executed")
		}).(*MockJob[any]),
		NewMockJob(3, make(chan struct{}), make(chan struct{}), func(ctx context.Context, job *MockJob[any]) {
			defer wg.Done() // Signal that this job is done
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			log.Printf("Job 3 executed")
		}).(*MockJob[any]),
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
		for _, j := range jobs {
			if !j.Executed {
				t.Errorf("Jobs did not run in parallel")
			}
		}

	case <-time.After(50 * time.Millisecond):
		t.Errorf("Jobs did not run in parallel")
	}
}

func TestRunnerProgress(t *testing.T) {
	ctx := context.Background()
	runner := NewStaticRunner[any](StrategySequential{}, nil)

	// Create jobs, each with a CompleteChan to control when they finish
	jobs := []MockJob[any]{
		{
			ID:        1,
			StartChan: make(chan struct{}),
			EndChan:   make(chan struct{}),
			function: func(ctx context.Context, job *MockJob[any]) {
				// Wait for the signal to start
				<-job.StartChan // Receive to unblock the send in test
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
				job.EndChan <- struct{}{}
			},
		},
		{
			ID:        2,
			StartChan: make(chan struct{}),
			EndChan:   make(chan struct{}),
			function: func(ctx context.Context, job *MockJob[any]) {
				// Wait for the signal to start
				<-job.StartChan // Receive to unblock the send in test
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
				job.EndChan <- struct{}{}
			},
		},
		{
			ID:        3,
			StartChan: make(chan struct{}),
			EndChan:   make(chan struct{}),
			function: func(ctx context.Context, job *MockJob[any]) {
				// Wait for the signal to start
				<-job.StartChan // Receive to unblock the send in test
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
				job.EndChan <- struct{}{}
			},
		},
	}

	// Add jobs to the runner
	for i := range jobs {
		runner.AddJob(&jobs[i])
	}

	// Run the jobs in a separate goroutine
	go runner.Run(ctx)

	// Verify sequential execution and check progress after each job completes
	for i := 0; i < len(jobs); i++ {

		// Signal the current job to complete
		jobs[i].StartChan <- struct{}{}

		// Wait for the current job to finish and check progress
		<-jobs[i].EndChan

		progress := runner.CheckProgress()

		t.Logf("Progress after job %d: %f%%", jobs[i].ID, progress)

		if progress <= float64(i*33) || progress > float64((i+1)*33+1) {
			t.Errorf("Progress is out of bounds after job %d: %f", jobs[i].ID, progress)
		}
	}

	progress := runner.CheckProgress()

	if progress != 100.0 {
		t.Errorf("Expected progress to be 100.0, got %f", progress)
	}
}

func TestRunnerSequentialExecution(t *testing.T) {
	ctx := context.Background()
	runner := NewStaticRunner[any](StrategySequential{}, nil)

	// Create mock jobs and give them a function
	jobs := []MockJob[any]{
		{
			ID:        1,
			StartChan: make(chan struct{}),
			EndChan:   make(chan struct{}),
			function: func(ctx context.Context, job *MockJob[any]) {
				// Wait for the signal to start
				<-job.StartChan // Receive to unblock the send in test
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
				job.EndChan <- struct{}{}
			},
		},
		{
			ID:        2,
			StartChan: make(chan struct{}),
			EndChan:   make(chan struct{}),
			function: func(ctx context.Context, job *MockJob[any]) {
				// Wait for the signal to start
				<-job.StartChan // Receive to unblock the send in test
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
				job.EndChan <- struct{}{}
			},
		},
		{
			ID:        3,
			StartChan: make(chan struct{}),
			EndChan:   make(chan struct{}),
			function: func(ctx context.Context, job *MockJob[any]) {
				// Wait for the signal to start
				<-job.StartChan // Receive to unblock the send in test
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
				job.EndChan <- struct{}{}
			},
		},
	}

	// Add jobs to the runner
	for i := range jobs {
		runner.AddJob(&jobs[i])
	}

	// Run the jobs
	go runner.Run(ctx)

	// Verify sequential execution
	for i := 0; i < len(jobs); i++ {
		log.Printf("Waiting for Job %d to finish...", jobs[i].ID)
		jobs[i].StartChan <- struct{}{}
		// Wait for the current job to finish and check progress
		<-jobs[i].EndChan
		if i < len(jobs)-1 {
			log.Printf("Job %d finished, checking if Job %d starts...", jobs[i].ID, jobs[i+1].ID)
		}
		//check if jobs[i] completed, i+1 false
		if i > 1 {
			if jobs[i-1].Executed == false && jobs[i].Executed {
				t.Error("Jobs finished out of order")
			}
		}
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := NewStaticRunner[any](StrategyParallel{}, nil)

	// Create mock jobs
	jobs := []MockJob[any]{
		{ID: 1, function: func(ctx context.Context, job *MockJob[any]) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}},
		{ID: 2, function: func(ctx context.Context, job *MockJob[any]) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}},
		{ID: 3, function: func(ctx context.Context, job *MockJob[any]) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}},
	}

	// Add jobs to the runner
	for i := range jobs {
		runner.AddJob(&jobs[i])
	}

	// Cancel the context before running
	cancel()
	runner.Run(ctx)

	// Verify that no jobs were executed
	for _, job := range jobs {
		if job.Executed {
			t.Errorf("Job %d was executed despite context cancellation", job.ID)
		}
	}
}

func TestDynamicRunner(t *testing.T) {
	totalJobs := 100
	failAt := 50
	totalSuccess := 0

	workers := 4
	chanSize := 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewDynamicRunner[any](StrategyParallel{}, nil, workers, chanSize)
	jobs := []*MockJob[any]{}

	startChan := make(chan struct{}, totalJobs) // Channel to signal job start
	endChan := make(chan struct{}, totalJobs)   // Channel to signal job end

	// Run the runner in a separate goroutine
	go runner.Run(ctx)

	// Create a WaitGroup to track job execution
	var wg sync.WaitGroup

	// Add jobs to the runner
	for i := 0; i < totalJobs; i++ {
		job := NewMockJob(i, startChan, endChan, func(ctx context.Context, job *MockJob[any]) {
			defer wg.Done() // Mark job as done when it finishes
			select {
			case <-ctx.Done():
				// Context canceled, job will not execute
				fmt.Printf("Job %d canceled\n", job.ID)
				return
			default:
				// Simulate job execution
				<-job.StartChan // Receive to unblock the send in test
				fmt.Printf("Job %d executed\n", job.ID)
				job.Executed = true
			}
		})
		jobs = append(jobs, job.(*MockJob[any])) // Ensure type matches Processable[any]
		wg.Add(1)                                // Increment WaitGroup counter
		runner.AddJob(job)
		startChan <- struct{}{} // Signal that the job has started (for testing purposes)
	}

	// Cancel the context after a short delay
	time.AfterFunc(50*time.Millisecond, cancel)

	// Wait for all jobs to complete
	wg.Wait()

	// Check which jobs were executed
	for _, job := range jobs {
		if job.Executed {
			totalSuccess++
		}
	}

	fmt.Printf("Total jobs executed: %d/%d\n", totalSuccess, totalJobs)

	// Verify that the number of executed jobs is within an acceptable range
	if totalSuccess > totalJobs || totalSuccess < failAt {
		t.Errorf("Unexpected number of jobs executed: %d/%d", totalSuccess, totalJobs)
	}
}

func TestDynamicRunnerCtx(t *testing.T) {
	totalJobs := 2
	failAt := 1
	totalSuccess := 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewDynamicRunner[any](StrategyParallel{}, nil, 1, 500)
	jobs := []*MockJob[any]{}

	// Create a WaitGroup to track job execution
	var wg sync.WaitGroup

	// Run the runner in a separate goroutine
	go runner.Run(ctx)

	// Add jobs to the runner
	for i := 0; i < totalJobs; i++ {
		startChan := make(chan struct{}, 1) // Channel to signal job start
		endChan := make(chan struct{}, 1)   // Channel to signal job end
		job := NewMockJob(i, startChan, endChan, func(ctx context.Context, job *MockJob[any]) {
			defer wg.Done() // Mark job as done when it finishes
			select {
			case <-ctx.Done():
				// Context canceled, job will not execute
				fmt.Printf("Job %d canceled\n", job.ID)
				return
			default:
				// Wait for the signal to start
				fmt.Println("Waiting for job start signal...")
				<-job.StartChan // Receive to unblock the send in test
				fmt.Printf("Job %d executing...\n", job.ID)
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
				job.EndChan <- struct{}{}
				job.Executed = true
			}
		})
		jobs = append(jobs, job.(*MockJob[any])) // Ensure type matches Processable[any]

		// if i == failAt {
		// 	cancel() // Cancel the context when we reach the failAt threshold
		// }

		wg.Add(1) // Increment WaitGroup counter
		go func() {
			runner.AddJob(job)
		}()

	}

	// Cancel the context after a short delay

	for i, r := range jobs {
		if i == failAt {
			// Cancel the context to simulate a failure at this point
			fmt.Println("Canceling context at job index:", i)
			cancel()
		}
		r.StartChan <- struct{}{} // Signal that the job has started (for testing purposes)
	}
	// Wait for all jobs to complete
	runner.Shutdown()
	wg.Wait()

	// Check which jobs were executed
	for _, job := range jobs {
		if job.Executed {
			totalSuccess++
		}
	}

	fmt.Printf("Total jobs executed: %d/%d\n", totalSuccess, totalJobs)

	// Verify that the number of executed jobs is within an acceptable range
	if totalSuccess > failAt {
		t.Errorf("Unexpected number of jobs executed: %d/%d expected %d", totalSuccess, totalJobs, failAt)
	}
}
