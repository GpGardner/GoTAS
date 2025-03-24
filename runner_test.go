package main

import (
	"context"
	"testing"
	"time"
)

// MockJob is a mock implementation of the Job interface for testing.
type MockJob struct {
	ID          int
	Executed    bool
	createdAt   *time.Time
	completedAt *time.Time
	function    func(ctx context.Context, args ...any)
}

func (j *MockJob) NewMockJob() *MockJob {
	return &MockJob{
		createdAt: ptr(time.Now()),
	}
}

func (j *MockJob) GetStatus() Status {
	return StatusCompleted
}

func (j *MockJob) GetError() error {
	return nil
}

func (j *MockJob) GetResult() any {
	return nil
}

func (j *MockJob) GetDuration() time.Duration {
	return 10 * time.Millisecond
}

func (j *MockJob) CreatedAt() time.Time {
	return *j.createdAt
}

func (j *MockJob) CompletedAt() time.Time {
	if j.completedAt == nil {
		return time.Time{}
	}
	return *j.completedAt
}

func (j *MockJob) Complete() {
	j.completedAt = ptr(time.Now())
	j.Executed = true
}

func (j *MockJob) execute(ctx context.Context) {
	j.function(ctx, nil)
	j.Complete()
}

func TestRunnerParallel(t *testing.T) {
	ctx := context.Background()
	runner := NewRunner(StrategyParallel, nil)

	//Create mock functions
	functions := []func(ctx context.Context, args ...any){
		func(ctx context.Context, args ...any) { time.Sleep(10 * time.Millisecond) },
		func(ctx context.Context, args ...any) { time.Sleep(20 * time.Millisecond) },
		func(ctx context.Context, args ...any) { time.Sleep(30 * time.Millisecond) },
	}

	// Create mock jobs with functions and args
	jobs := []MockJob{
		{ID: 1, function: func(ctx context.Context, args ...any) { functions[0](ctx, args...) }},
		{ID: 2, function: func(ctx context.Context, args ...any) { functions[1](ctx, args...) }},
		{ID: 3, function: func(ctx context.Context, args ...any) { functions[2](ctx, args...) }},
	}

	// Add jobs to the runner
	for i := range jobs {
		runner.AddJob(&jobs[i])
	}

	// Run the jobs
	runner.Run(ctx)

	// Verify all jobs were executed
	for i, job := range jobs {

		if !job.Executed {
			t.Errorf("Job %d was not executed in parallel mode", i+1)
		}
	}
}

func TestRunnerProgress(t *testing.T) {
	ctx := context.Background()
	runner := NewRunner(StrategyParallel, nil)

	// Create 3 functions of different durations that I can pass to the mock jobs
	functions := []func(ctx context.Context, args ...any){
		func(ctx context.Context, args ...any) { time.Sleep(10 * time.Millisecond) },
		func(ctx context.Context, args ...any) { time.Sleep(20 * time.Millisecond) },
		func(ctx context.Context, args ...any) { time.Sleep(30 * time.Millisecond) },
	}

	// Create mock jobs
	jobs := []MockJob{
		{ID: 1, function: functions[0]}, {ID: 2, function: functions[1]}, {ID: 3, function: functions[2]},
		{ID: 4, function: functions[0]}, {ID: 5, function: functions[1]}, {ID: 6, function: functions[2]},
		{ID: 7, function: functions[0]}, {ID: 8, function: functions[1]}, {ID: 9, function: functions[2]},
		{ID: 10, function: functions[0]}, {ID: 11, function: functions[1]}, {ID: 12, function: functions[2]},
		{ID: 13, function: functions[2]}, {ID: 14, function: functions[1]}, {ID: 15, function: functions[2]},
		{ID: 16, function: functions[2]}, {ID: 17, function: functions[1]}, {ID: 18, function: functions[2]},
		{ID: 19, function: functions[0]}, {ID: 20, function: functions[1]},
	}

	// Add jobs to the runner
	for i := range jobs {
		runner.AddJob(&jobs[i])
	}

	// Run the jobs in a separate goroutine
	go runner.Run(ctx)

	// Wait a bit and check progress
	time.Sleep(15 * time.Millisecond)
	progress := runner.CheckProgress()
	t.Log(progress)

	if progress <= 0.0 || progress >= 100.0 {
		t.Errorf("Progress is out of bounds: %f", progress)
	}

	// Wait for all jobs to complete
	time.Sleep(50 * time.Millisecond)
	progress = runner.CheckProgress()

	if progress != 100.0 {
		t.Errorf("Expected progress to be 1.0, got %f", progress)
	}
}

func TestRunnerSequentialExecution(t *testing.T) {
	ctx := context.Background()
	runner := NewRunner(StrategySequential, nil)

	// Create mock jobs
	jobs := []MockJob{
		{ID: 1, function: func(ctx context.Context, args ...any) { time.Sleep(10 * time.Millisecond) }},
		{ID: 2, function: func(ctx context.Context, args ...any) { time.Sleep(20 * time.Millisecond) }},
		{ID: 3, function: func(ctx context.Context, args ...any) { time.Sleep(30 * time.Millisecond) }},
	}

	// Add jobs to the runner
	for i := range jobs {
		runner.AddJob(&jobs[i])
	}

	// Run the jobs
	runner.Run(ctx)

	// Verify sequential execution
	for i := 1; i < len(jobs); i++ {
		if jobs[i].CreatedAt().Before(jobs[i-1].CompletedAt()) {
			t.Errorf("Job %d started before Job %d finished", jobs[i].ID, jobs[i-1].ID)
		}
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := NewRunner(StrategyParallel, nil)

	// Create mock jobs
	jobs := []MockJob{
		{ID: 1, Executed: false, function: func(ctx context.Context, args ...any) { time.Sleep(100 * time.Millisecond) }},
		{ID: 2, Executed: false, function: func(ctx context.Context, args ...any) { time.Sleep(100 * time.Millisecond) }},
		{ID: 3, Executed: false, function: func(ctx context.Context, args ...any) { time.Sleep(100 * time.Millisecond) }},
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
