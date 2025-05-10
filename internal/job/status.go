package job

import "fmt"

// Status represents the state of a job in the scheduler.
type Status int

const (

	// === Initial States ===
	StatusCreated Status = 0 // Job has been registered but hasn't started execution yet
	StatusPending Status = 1 // Job is waiting for dependencies to complete before running

	// === Active Execution ===
	StatusRunning Status = 2 // Job is currently being executed

	// === Successful Completion ===
	StatusCompleted Status = 3 // Job finished successfully

	// === Termination Due to Issues ===
	StatusFailed    Status = 4 // Job ran but encountered a system issue
	StatusError     Status = 5 // Job failed due to an error in the system or the job itself
	StatusCancelled Status = 6 // Job was manually stopped before completion
	StatusTimeout   Status = 7 // Job exceeded its allowed execution time and was forcefully stopped

	// === Invalid or Unclear States ===
	StatusUnknown Status = 8 // Job's state is unclear due to an unexpected condition
	StatusInvalid Status = 9 // Job was rejected due to invalid parameters or configuration
)

// --- Utility Methods ---

// IsExecuting checks if a status is in the execution phase (before completion or failure).
func (s Status) IsExecuting() bool {
	return s == StatusRunning
}

// IsPending checks if a status indicates the job is waiting for dependencies.
func (s Status) IsPending() bool {
	return s == StatusPending
}

// IsCreated checks if a status indicates the job is newly created.
func (s Status) IsCreated() bool {
	return s == StatusCreated
}

// IsCompleted checks if a status indicates successful job completion.
func (s Status) IsCompleted() bool {
	return s == StatusCompleted
}

// IsFailure checks if a status indicates job failure or termination.
func (s Status) IsFailure() bool {
	return s == StatusFailed || s == StatusCancelled || s == StatusTimeout || s == StatusUnknown || s == StatusInvalid
}

func (s Status) String() string {
  switch s {
  case StatusCreated:
    return "Created"
  case StatusPending:
    return "Pending"
  case StatusRunning:
    return "Running"
  case StatusCompleted:
    return "Completed"
  case StatusFailed:
    return "Failed"
  case StatusError:
    return "Error"
  case StatusCancelled:
    return "Cancelled"
  case StatusTimeout:
    return "Timeout"
  case StatusUnknown:
    return "Unknown"
  case StatusInvalid:
    return "Invalid"
  default:
    return fmt.Sprintf("Unknown status: %d", s)
  }
}
