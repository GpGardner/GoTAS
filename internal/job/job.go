package job

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"GOTAS/internal/utils"
)

type Job interface {
	Run(ctx context.Context)    //
	Complete()                  // Marks the job as completed
	GetStatus() Status          // Returns current job status
	GetError() error            // Returns job error (if any)
	GetResult() any             // Returns job result (if applicable)
	GetDuration() time.Duration // Returns execution time
	CreatedAt() time.Time       // Returns the time the job was created
	CompletedAt() time.Time     // Returns the time the job was completed
}

type jobBase struct {
	id          uuid.UUID     // Unique identifier of the job
	status      Status        // Status of the job
	createdAt   *time.Time    // Time the job was created
	completedAt *time.Time    // Time the job was completed
	duration    time.Duration // Duration of the job execution
	error       error         // Error message of the job (converted to string for serialization)
}

// MarshalJSON implements custom JSON serialization for JobBase
func (j *jobBase) MarshalJSON() ([]byte, error) {
	var errMsg string
	if j.error != nil {
		errMsg = j.error.Error()
	}

	return json.Marshal(struct {
		ID          string        `json:"id" bson:"_id"`
		Status      Status        `json:"status" bson:"status"`
		CreatedAt   *time.Time    `json:"created_at,omitempty" bson:"created_at,omitempty"`
		CompletedAt *time.Time    `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
		Duration    time.Duration `json:"duration" bson:"duration"`
		Error       string        `json:"error,omitempty" bson:"error,omitempty"`
	}{
		ID:          j.id.String(),
		Status:      j.status,
		CreatedAt:   j.createdAt,
		CompletedAt: j.completedAt,
		Duration:    j.duration,
		Error:       errMsg,
	})
}

func NewJob() jobBase {
	return jobBase{
		id:        uuid.New(),
		status:    StatusPending,
		createdAt: utils.Ptr(time.Now()),
	}
}

// Job that returns an error
type JobWithError struct {
	jobBase
	function func(ctx context.Context, args ...any) error
}

// Job that returns a result and an error
type JobWithResult[T any] struct {
	jobBase
	function func(ctx context.Context, args ...any) (T, error)
	result   T
}

func (j *JobWithError) Run(ctx context.Context, args ...any) error {
	return j.execute(ctx)
}

func (j *JobWithResult[T]) Run(ctx context.Context, args ...any) (T, error) {
	return j.execute(ctx)
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
		completeJob(&j.jobBase, StatusTimeout)
	case <-done: // Job finished normally
		if j.error != nil {
			completeJob(&j.jobBase, StatusError)
		} else {
			completeJob(&j.jobBase, StatusCompleted)
		}
	}

	return j.error
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
		completeJob(&j.jobBase, StatusTimeout)
	case <-done: // Job finished normally
		if j.error != nil {
			j.result = empty
			completeJob(&j.jobBase, StatusError)
		} else {
			completeJob(&j.jobBase, StatusCompleted)
		}
	}

	return j.result, j.error
}

func completeJob(j *jobBase, status Status) {
	j.status = status
	j.completedAt = utils.Ptr(time.Now())
	j.duration = time.Since(*j.createdAt)
}

func (j *jobBase) ID() uuid.UUID {
	return j.id
}

func (j *jobBase) Status() Status {
	return j.status
}

// CreatedAt returns the time the job was created or nil if not set
func (j *jobBase) CreatedAt() time.Time {
	if j.createdAt == nil {
		return time.Time{}
	}
	return *j.createdAt
}

// CompletedAt returns the time the job was completed or nil if not set
func (j *jobBase) CompletedAt() time.Time {
	if j.completedAt == nil {
		return time.Time{}
	}
	return *j.completedAt
}

func (j *jobBase) Duration() time.Duration {
	return j.duration
}

func (j *jobBase) Error() error {
	return j.error
}
