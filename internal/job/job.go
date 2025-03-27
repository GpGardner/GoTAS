package job

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"GOTAS/internal/utils"
)

type Job[T any] interface {
	Run(ctx context.Context, args ...any) (T, error) // Executes the job and returns result or error
	Complete(Status)                                 // Marks the job as completed
	GetStatus() Status                               // Returns current job status
	GetError() error                                 // Returns job error (if any)
	GetResult() any                                  // Returns job result (if applicable)
	GetDuration() time.Duration                      // Returns execution time
	CreatedAt() time.Time                            // Returns the time the job was created
	CompletedAt() time.Time                          // Returns the time the job was completed
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

func NewJob() (*jobBase, error) {
	id, err := GenerateID()
	if err != nil {
		return nil, err
	}

	t, err := GenerateCreatedAt()
	if err != nil {
		return nil, err
	}

	return &jobBase{
		id:        id,
		status:    StatusPending,
		createdAt: t,
	}, nil
}

func completeJob(j *jobBase, status Status) {
	j.status = status
	j.completedAt = utils.Ptr(time.Now())
	j.duration = time.Since(*j.createdAt)
}

func GenerateID() (uuid.UUID, error) {
	return uuid.NewUUID()
}

func GenerateCreatedAt() (*time.Time, error) {
	t := time.Now()
	return &t, nil
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

func (j *jobBase) GetDuration() time.Duration {
	return j.duration
}

func (j *jobBase) Error() error {
	return j.error
}
