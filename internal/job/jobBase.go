package job

import (
	"GOTAS/internal/utils"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

//Consider using a sync.Pool to manage timers for better performance
// This pool can be used to reuse timers instead of creating new ones every time.
// This can help reduce memory allocation and improve performance in high-load scenarios.

// var timerPool = sync.Pool{
//     New: func() interface{} {
//         return time.NewTimer(0)
//     },
// }

// func getTimer(d time.Duration) *time.Timer {
//     t := timerPool.Get().(*time.Timer)
//     if !t.Stop() {
//         <-t.C
//     }
//     t.Reset(d)
//     return t
// }

// func putTimer(t *time.Timer) {
//     if !t.Stop() {
//         <-t.C
//     }
//     timerPool.Put(t)
// }

type jobBase struct {
	mu          sync.Mutex    // Mutex to protect concurrent access
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

	j.mu.Lock()
	defer j.mu.Unlock()

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
	id, err := generateID()
	if err != nil {
		return nil, err
	}

	t, err := generateCreatedAt()
	if err != nil {
		return nil, err
	}

	return &jobBase{
		id:        id,
		status:    StatusPending,
		createdAt: t,
	}, nil
}

func (j *jobBase) completeJob(status Status) {
	j.setStatus(status)
	j.complete()
	j.calculateDuration()
}

func generateID() (uuid.UUID, error) {
	return uuid.NewUUID()
}

func generateCreatedAt() (*time.Time, error) {
	t := time.Now()
	return &t, nil
}

// start marks the job as started and sets the started time
func (j *jobBase) start() {
	// Mark the job as started
	j.setStatus(StatusRunning)
	j.setStartedAt(time.Now())
}

func (j *jobBase) setStartedAt(t time.Time) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.startedAt = utils.Ptr(t)
}

// ID returns the unique identifier of the job'
// ID cannot be nil, if it is nil, generate a new one
func (j *jobBase) getID() uuid.UUID {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.id == uuid.Nil {
		id, err := generateID()
		if err != nil {
			return uuid.Nil
		}
		j.id = id
	}
	return j.id
}

// Status returns the current status of the job
func (j *jobBase) getStatus() Status {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.status
}

// Status returns the current status of the job
func (j *jobBase) setStatus(s Status) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status = s
}

// CreatedAt returns the time the job was created or nil if not set
func (j *jobBase) getCreatedAt() *time.Time {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.createdAt
}

// CompletedAt returns the time the job was completed or nil if not set
func (j *jobBase) getCompletedAt() *time.Time {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.completedAt
}

func (j *jobBase) complete() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.completedAt = utils.Ptr(time.Now())
}

func (j *jobBase) getStartedAt() *time.Time {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.startedAt
}

func (j *jobBase) getDuration() time.Duration {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.duration
}

func (j *jobBase) calculateDuration() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.startedAt != nil {
		j.duration = time.Since(*j.startedAt)
	} else {
		j.duration = 0
	}
}

func (j *jobBase) getError() error {
	return j.error
}

func (j *jobBase) setError(e error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.error = e
}

func (j *Job[T]) setResult(result T) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.result = result
}

func (j *Job[T]) setResultEmpty() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.result = *new(T) // Reset result to zero value
}
