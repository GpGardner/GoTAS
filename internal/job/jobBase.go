package job

import (
	"GOTAS/internal/utils"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type jobBase struct {
	id          uuid.UUID     // Unique identifier of the job
	status      Status        // Status of the job
	createdAt   *time.Time    // Time the job was created
	startedAt   *time.Time    // Time the job execution started
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
		CreatedAt   *time.Time    `json:"created_at" bson:"created_at"`
		StartedAt   *time.Time    `json:"started_at" bson:"started_at"`
		CompletedAt *time.Time    `json:"completed_at" bson:"completed_at"`
		Duration    time.Duration `json:"duration" bson:"duration"`
		Error       string        `json:"error" bson:"error"`
	}{
		ID:          j.id.String(),
		Status:      j.status,
		CreatedAt:   j.createdAt,
		StartedAt:   j.startedAt,
		CompletedAt: j.completedAt,
		Duration:    j.duration,
		Error:       errMsg,
	})
}

func NewJobBase() (*jobBase, error) {
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
	j.duration = time.Since(*j.startedAt)
}

func GenerateID() (uuid.UUID, error) {
	return uuid.NewUUID()
}

func GenerateCreatedAt() (*time.Time, error) {
	t := time.Now()
	return &t, nil
}

// Start marks the job as started and sets the started time
func (j *jobBase) Start() {
	// Mark the job as started
	now := time.Now()
	j.status = StatusRunning
	j.startedAt = utils.Ptr(now)
}

// ID returns the unique identifier of the job
func (j *jobBase) ID() uuid.UUID {
	return j.id
}

// Status returns the current status of the job
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

func (j *jobBase) StartedAt() time.Time {
	if j.startedAt == nil {
		return time.Time{}
	}
	return *j.startedAt
}

func (j *jobBase) GetDuration() time.Duration {
	return j.duration
}

func (j *jobBase) Error() error {
	return j.error
}
