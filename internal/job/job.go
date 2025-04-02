package job

import (
	"context"
	"fmt"
	"time"
)

type Processable[T any] interface {
	Run(ctx context.Context, args ...any) (any, error) // Executes the job and returns result or error
	Complete(Status)                                   // Marks the job as completed
	GetStatus() Status                                 // Returns current job status
	GetError() error                                   // Returns job error (if any)
	GetResult() any                                    // Returns job result (if applicable)
	GetDuration() time.Duration                        // Returns execution time
	CreatedAt() time.Time                              // Returns the time the job was created
	CompletedAt() time.Time                            // Returns the time the job was completed
}

// make sure Job implements Processable interface
var _ Processable[any] = (*Job[any])(nil)

// Job that returns a result and an error
type Job[T any] struct {
	*jobBase
	function func(ctx context.Context, args ...any) (T, error)
	result   T
}

func (j *Job[T]) Run(ctx context.Context, args ...any) (any, error) {
	return j.execute(ctx, args...)
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
func NewJob[T any](f func(ctx context.Context, args ...any) (T, error)) (*Job[T], error) {
	j, err := NewJobBase()
	if err != nil {
		return nil, err
	}

	return &Job[T]{
		jobBase:  j,
		function: f,
	}, nil
}

// Complete marks the job as completed
func (j *Job[T]) Complete(status Status) {
	completeJob(j.jobBase, status)
}

// GetStatus returns the current job status
func (j *Job[T]) GetStatus() Status {
	return j.status
}

// GetError returns the job error (if any)
func (j *Job[T]) GetError() error {
	return j.error
}

// GetResult returns the job result (if applicable)
func (j *Job[T]) GetResult() any {
	return j.result
}

// GetDuration returns the execution time
func (j *Job[T]) GetDuration() time.Duration {
	return j.duration
}

// CreatedAt returns the time the job was created
// default time if nil
func (j *Job[T]) CreatedAt() time.Time {
	return j.jobBase.CreatedAt()
}

// StartedAt returns the time the job was started
// default time if nil
func (j *Job[T]) StartedAt() time.Time {
	return j.jobBase.StartedAt()
}

// CompletedAt returns the time the job was completed
// default time if nil
func (j *Job[T]) CompletedAt() time.Time {
	return j.jobBase.CompletedAt()
}

func (j *Job[T]) execute(ctx context.Context, args ...any) (any, error) {
	// Channel for signaling job completion
	done := make(chan struct{})

	go func() {
		defer close(done) // Ensure channel is closed when job exits
		j.jobBase.start() // Mark the job as started
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

// String returns a human-readable string representation of the Job instance.
// This method implements the fmt.Stringer interface, allowing the Job to be printed in a readable format.
// This is useful for logging and debugging purposes.
// If you need to obsfucate the result, then you can override this method in your own implementation.
func (j *Job[T]) String() string {
	return fmt.Sprintf(
		"Job[ID=%d, Status=%s, CreatedAt=%s, Start=%s, CompletedAt=%s, Error=%v, Duration=%s]",
		j.jobBase.ID(),
		j.GetStatus(),
		j.CreatedAt().Format(time.RFC3339),
		j.StartedAt().Format(time.RFC3339),
		j.CompletedAt().Format(time.RFC3339),
		j.GetError(), // This will be nil if no error occurred
		j.GetDuration(),
	)
}
