package job

import (
	"context"
	"fmt"
	"time"
)

// make sure jobWithError implements job
var _ Job[any] = (*JobWithError)(nil)

// Job that returns an error
type JobWithError struct {
	*jobBase
	function func(ctx context.Context, args ...any) error
}

// NewJobWithError creates a new JobWithError instance.
// It initializes the jobBase fields and sets the function that will be executed.
//
// Parameters:
// f: The function to be executed by the job. It takes a context and a variadic list of arguments
// and returns an error.
//
// args: A variadic list of arguments to be passed to the job.
//
// Returns:
// A pointer to the JobWithError instance and an error if the job creation fails.
// If the job creation is successful, the error will be nil.
func NewJobWithError(f func(ctx context.Context, args ...any) error) (*JobWithError, error) {
	j, err := NewJob()
	if err != nil {
		return nil, err
	}

	return &JobWithError{
		jobBase:  j,
		function: f,
	}, nil

}

// Run executes the job with the provided context and arguments.
// It delegates the execution to the internal execute method.
//
// Parameters:
//   - ctx: The context for managing the job's lifecycle.
//   - args: A variadic list of arguments to be passed to the job.
//
// Returns:
//   - An error if the job execution fails, otherwise nil.
func (j *JobWithError) Run(ctx context.Context, args ...any) (any, error) {
	return nil, j.execute(ctx, args...)
}

// Complete marks the JobWithError instance as completed by updating its status
// to StatusCompleted. It utilizes the completeJob function to perform the
// status update on the embedded jobBase field.
func (j *JobWithError) Complete(status Status) {
	completeJob(j.jobBase, status)
}

// GetStatus returns the current status of the JobWithError instance.
// It provides information about the state of the job.
func (j *JobWithError) GetStatus() Status {
	return j.status
}

// GetError returns the error associated with the JobWithError instance.
// If no error has occurred, it returns nil.
func (j *JobWithError) GetError() error {
	return j.error
}

// GetResult returns nil as JobWithError does not produce a result.
// This method satisfies the interface requirement but does not provide
// any meaningful output for this type of job.
func (j *JobWithError) GetResult() any {
	return nil // JobWithError does not produce a result
}

// GetDuration returns the duration of the job.
// It retrieves the time.Duration value associated with the JobWithError instance.
// Will return 0 if not complete
func (j *JobWithError) GetDuration() time.Duration {

	return j.duration
}

// CreatedAt returns the creation time of the JobWithError instance.
// If the creation time is not set, it returns the zero value of time.Time.
func (j *JobWithError) CreatedAt() time.Time {
	if j.createdAt != nil {
		return *j.createdAt
	}
	return time.Time{}
}

// CompletedAt returns the time when the job was completed.
// If the job has not been completed, it returns the zero value of time.Time.
func (j *JobWithError) CompletedAt() time.Time {
	if j.completedAt != nil {
		return *j.completedAt
	}
	return time.Time{}
}

func (j *JobWithError) execute(ctx context.Context, args ...any) error {
	// Channel for signaling job completion
	done := make(chan struct{})

	go func() {
		defer close(done) // Ensure channel is closed when job exits
		j.status = StatusRunning
		j.error = j.function(ctx, args)
	}()

	select {
	case <-ctx.Done(): // Context was canceled
		j.error = ctx.Err()
		completeJob(j.jobBase, StatusTimeout)
	case <-done: // Job finished normally
		if j.error != nil {
			completeJob(j.jobBase, StatusError)
		} else {
			completeJob(j.jobBase, StatusCompleted)
		}
	}

	return j.error
}

// String returns a human-readable string representation of the JobWithError instance.
func (j *JobWithError) String() string {
	return fmt.Sprintf(
		"JobWithError[ID=%d, Status=%s, CreatedAt=%s, CompletedAt=%s, Error=%v, Duration=%s]",
		j.id,
		j.GetStatus(),
		j.CreatedAt().Format(time.RFC3339),
		j.CompletedAt().Format(time.RFC3339),
		j.error,
		j.GetDuration(),
	)
}
