package job

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Processable[T any] interface {
	Run(ctx context.Context, args ...any) (T, error) // Executes the job and returns result or error
	Complete(Status)                                 // Marks the job as completed
	GetStatus() Status                               // Returns current job status
	GetError() error                                 // Returns job error (if any)
	GetResult() T                                    // Returns job result (if applicable)
	GetDuration() time.Duration                      // Returns execution time
	CreatedAt() time.Time                            // Returns the time the job was created
	CompletedAt() time.Time                          // Returns the time the job was completed
	GetID() uuid.UUID                                // Returns the unique identifier of the job
}

// make sure Job implements Processable interface
var _ Processable[any] = (*Job[any])(nil)

// Job that returns a result and an error
type Job[T any] struct {
	*jobBase
	function func(ctx context.Context, args ...any) (T, error)
	result   T
}

func (j *Job[T]) Run(ctx context.Context, args ...any) (T, error) {
	j.start()

	//Check that ctx isnt already complete
	if ctx.Err() != nil {
		j.setError(ctx.Err())
		j.result = *new(T) // Reset result to zero value
		j.Complete(StatusTimeout)
		return j.result, j.error
	}

	j.result, j.error = j.function(ctx, args...)
	if ctx.Err() != nil {
		j.setError(ctx.Err())
		j.result = *new(T)
		j.Complete(StatusTimeout)
	} else if j.error != nil {
		j.Complete(StatusError)
	} else {
		j.Complete(StatusCompleted)
	}
	return j.result, j.error
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
	j.jobBase.completeJob(status)
}

// GetStatus returns the current job status
func (j *Job[T]) GetStatus() Status {
	return j.jobBase.getStatus()
}

// GetError returns the job error (if any)
func (j *Job[T]) GetError() error {
	return j.jobBase.getError()
}

// GetResult returns the job result (if applicable)
func (j *Job[T]) GetResult() T {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.result
}

// GetDuration returns the execution time
func (j *Job[T]) GetDuration() time.Duration {
	return j.jobBase.getDuration()
}

// CreatedAt returns the time the job was created
// default time if nil
func (j *Job[T]) CreatedAt() time.Time {
	if j.jobBase.getCreatedAt() == nil {
		return time.Time{}
	}
	return *j.jobBase.getCreatedAt()
}

// StartedAt returns the time the job was started
// default time if nil
func (j *Job[T]) StartedAt() time.Time {
	// If the job has not started yet, set default time
	if j.jobBase.getStartedAt() == nil {
		return time.Time{}
	}
	return *j.jobBase.getStartedAt()
}

// CompletedAt returns the time the job was completed
// default time if nil
func (j *Job[T]) CompletedAt() time.Time {
	if j.jobBase.getCompletedAt() == nil {
		return time.Time{}
	}
	return *j.jobBase.getCompletedAt()
}

func (j *Job[T]) GetID() uuid.UUID {
	return j.jobBase.getID()
}

// String returns a human-readable string representation of the Job instance.
// This method implements the fmt.Stringer interface, allowing the Job to be printed in a readable format.
// This is useful for logging and debugging purposes.
// If you need to obsfucate the result, then you can override this method in your own implementation.
func (j *Job[T]) String() string {
	return fmt.Sprintf(
		"\nJob[ID=%d,\n Status=%s,\n CreatedAt=%s,\n Start=%s,\n CompletedAt=%s,\n Error=%v,\n Duration=%s\n]",
		j.jobBase.getID(),
		j.GetStatus(),
		j.CreatedAt().Format(time.RFC3339),
		j.StartedAt().Format(time.RFC3339),
		j.CompletedAt().Format(time.RFC3339),
		j.GetError(), // This will be nil if no error occurred
		j.GetDuration(),
	)
}
