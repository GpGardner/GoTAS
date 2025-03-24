package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Job interface {
	execute(ctx context.Context) // Runs the job with context for cancellation
	Complete()                   // Marks the job as completed
	GetStatus() Status           // Returns current job status
	GetError() error             // Returns job error (if any)
	GetResult() any              // Returns job result (if applicable)
	GetDuration() time.Duration  // Returns execution time
	CreatedAt() time.Time        // Returns the time the job was created
	CompletedAt() time.Time      // Returns the time the job was completed
}

type JobBase struct {
	id          uuid.UUID     `json:"id" bson:"id"`                     // Unique identifier of the job
	status      Status        `json:"status" bson:"status"`             // Status of the job
	createdAt   *time.Time    `json:"created_at" bson:"created_at"`     // Time the job was created
	completedAt *time.Time    `json:"completed_at" bson:"completed_at"` // Time the job was completed
	duration    time.Duration `json:"duration" bson:"duration"`         // Duration of the job execution
	error       error         `json:"error" bson:"error"`               // Error message of the job (converted to string for serialization)
}

func NewJobBase() JobBase {
	return JobBase{
		id:        uuid.New(),
		status:    StatusPending,
		createdAt: ptr(time.Now()),
	}
}

// Job that returns an error
type JobWithError struct {
	JobBase
	function func(ctx context.Context, args ...any) error
}

// Job that returns a result and an error
type JobWithResult[T any] struct {
	JobBase
	function func(ctx context.Context, args ...any) (T, error)
	result   T
}

func (j *JobWithError) execute(ctx context.Context, args ...any) error {
	// Check if the context is already canceled
	if ctx.Err() != nil {
		j.error = ctx.Err()
		completeJob(&j.JobBase, StatusTimeout)
		return j.error
	}

	j.status = StatusRunning

	defer func() {
		if r := recover(); r != nil {
			j.error = fmt.Errorf("panic: %v", r)
			completeJob(&j.JobBase, StatusFailed)
			return
		}
		if j.error != nil {
			completeJob(&j.JobBase, StatusError)
			return
		}
		completeJob(&j.JobBase, StatusCompleted)
	}()

	// Execute the job function
	j.error = j.function(ctx, args)

	return j.error
}

func (j *JobWithResult[T]) execute(ctx context.Context, args ...any) (T, error) {
	// Check if the context is already canceled
	if ctx.Err() != nil {
		var empty T // Declare a zero value of type T
		j.error = ctx.Err()
		completeJob(&j.JobBase, StatusTimeout)
		return empty, j.error
	}
	j.status = StatusRunning

	defer func() {
		if r := recover(); r != nil {
			j.error = fmt.Errorf("panic: %v", r)
			completeJob(&j.JobBase, StatusFailed)
			return
		}
		if j.error != nil {
			completeJob(&j.JobBase, StatusError)
			return
		}
		completeJob(&j.JobBase, StatusCompleted)
	}()

	j.result, j.error = j.function(ctx, args)

	return j.result, j.error
}

func completeJob(j *JobBase, status Status) {
	j.status = status
	j.completedAt = ptr(time.Now())
	j.duration = time.Since(*j.createdAt)
}

func (j *JobBase) ID() uuid.UUID {
	return j.id
}

func (j *JobBase) Status() Status {
	return j.status
}

// CreatedAt returns the time the job was created or nil if not set
func (j *JobBase) CreatedAt() time.Time {
	if j.createdAt == nil {
		return time.Time{}
	}
	return *j.createdAt
}

// CompletedAt returns the time the job was completed or nil if not set
func (j *JobBase) CompletedAt() time.Time {
	if j.completedAt == nil {
		return time.Time{}
	}
	return *j.completedAt
}

func (j *JobBase) Duration() time.Duration {
	return j.duration
}

func (j *JobBase) Error() error {
	return j.error
}

func ptr[T any](v T) *T {
	return &v
}
