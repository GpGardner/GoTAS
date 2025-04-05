package runner

import (
	"context"
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
		// If context is cancelled, return early
		// log.Printf("Job %d was cancelled", j.ID)
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
