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
	id          uuid.UUID     // Unique identifier of the job
	status      Status        // Status of the job
	createdAt   *time.Time    // Time the job was created
	completedAt *time.Time    // Time the job was completed
	duration    time.Duration // Duration of the job execution
	error       error         // Error message of the job
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
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		j.status = StatusRunning
		start := time.Now()

		done := make(chan struct{})
		go func() {
			defer func() {
				if r := recover(); r != nil {
					j.error = fmt.Errorf("panic: %v", r)
					j.status = StatusFailed
				}
				close(done)
			}()
			j.error = j.function(ctx, args)
		}()

		select {
		case <-ctx.Done():
			j.status = StatusCancelled
			j.error = ctx.Err()
		case <-done:
			if j.error != nil {
				j.status = StatusFailed
			} else {
				j.status = StatusCompleted
			}
		}

		completeJob(&j.JobBase, start)
		return nil
	}
}

func (j *JobWithResult[T]) execute(ctx context.Context, args ...any) (T, error) {
	select {
	case <-ctx.Done():
		var empty T // Context was canceled, stop execution
		return empty, ctx.Err()
	default:
		j.status = StatusRunning
		start := time.Now()

		done := make(chan struct{})
		go func() {
			j.result, j.error = j.function(ctx, args)
			close(done)
		}()

		if j.error != nil {
			j.status = StatusFailed
		} else {
			j.status = StatusCompleted
		}

		completeJob(&j.JobBase, start)

		return j.result, j.error
	}
}

func completeJob(j *JobBase, start time.Time) {
	endTime := time.Now()
	j.completedAt = &endTime
	j.duration = time.Since(start)
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

func (j *JobBase) Complete() {
	j.completedAt = ptr(time.Now())
	j.duration = time.Since(*j.createdAt)
	j.status = StatusCompleted
}

func ptr[T any](v T) *T {
	return &v
}
