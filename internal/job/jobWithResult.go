package job

import (
	"context"
	"time"
)

// make sure jobwithresult implements job
var _ Job[any] = (*JobWithResult[any])(nil)

// Job that returns a result and an error
type JobWithResult[T any] struct {
	*jobBase
	function func(ctx context.Context, args ...any) (T, error)
	result   T
}

func (j *JobWithResult[T]) Run(ctx context.Context, args ...any) (T, error) {
	return j.execute(ctx)
}

// NewJobWithResult creates a new JobWithResult instance.
// It initializes the jobBase fields and sets the function that will be executed.
//
// Parameters:
// f: The function to be executed by the job. It takes a context and a variadic list of arguments
// and returns a result and an error.
//
// args: A variadic list of arguments to be passed to the job.
//
// Returns:
// A pointer to the JobWithResult instance and an error if the job creation fails.
// If the job creation is successful, the error will be nil.
func NewJobWithResult[T any](f func(ctx context.Context, args ...any) (T, error)) (*JobWithResult[T], error) {
	j, err := NewJob()
	if err != nil {
		return nil, err
	}

	return &JobWithResult[T]{
		jobBase:  j,
		function: f,
	}, nil
}

// Complete marks the job as completed
func (j *JobWithResult[T]) Complete(status Status) {
	completeJob(j.jobBase, status)
}

// GetStatus returns the current job status
func (j *JobWithResult[T]) GetStatus() Status {
	return j.status
}

// GetError returns the job error (if any)
func (j *JobWithResult[T]) GetError() error {
	return j.error
}

// GetResult returns the job result (if applicable)
func (j *JobWithResult[T]) GetResult() T {
	return j.result
}

// GetDuration returns the execution time
func (j *JobWithResult[T]) GetDuration() time.Duration {
	return j.duration
}

// CreatedAt returns the time the job was created
func (j *JobWithResult[T]) CreatedAt() time.Time {
	return *j.createdAt
}

// CompletedAt returns the time the job was completed
func (j *JobWithResult[T]) CompletedAt() time.Time {
	return *j.completedAt
}

func (j *JobWithResult[T]) execute(ctx context.Context, args ...any) (T, error) {
	// Channel for signaling job completion
	done := make(chan struct{})

	go func() {
		defer close(done) // Ensure channel is closed when job exits
		j.status = StatusRunning
		j.result, j.error = j.function(ctx, args)
	}()
	var empty T
	select {
	case <-ctx.Done(): // Context was canceled
		j.result = empty
		j.error = ctx.Err()
		completeJob(j.jobBase, StatusTimeout)
	case <-done: // Job finished normally
		if j.error != nil {
			j.result = empty
			completeJob(j.jobBase, StatusError)
		} else {
			completeJob(j.jobBase, StatusCompleted)
		}
	}

	return j.result, j.error
}
