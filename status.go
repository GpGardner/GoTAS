package main

// Status represents the state of a job in the scheduler.
type Status string

const (

	// === Initial States ===

	// StatusCreated means the job has been registered but hasn't started execution yet.
	// Use this when a job is added to the queue.
	StatusCreated Status = "created"

	// StatusPending means the job is waiting for dependencies to complete before running.
	// Use this when a job is blocked by another job in a sequential execution.
	StatusPending Status = "pending"

	// === Active Execution ===

	// StatusRunning means the job is currently being executed.
	// Use this when a job is actively processing its task.
	StatusRunning Status = "running"

	// === Successful Completion ===

	// StatusCompleted means the job finished successfully.
	// Use this when a job has executed without errors and produced a valid result.
	StatusCompleted Status = "completed"

	// === Termination Due to Issues ===

	// StatusFailed means the job ran but encountered an system issue.
	// Use this when a job fails due to an internal issue like a panic.
	StatusFailed Status = "failed"

	// StatusError means the job failed due to an error in the system or the job itself.
	// Use this when a job encounters an error that prevents it from running or completing.
	StatusError Status = "error"

	// StatusCancelled means the job was manually stopped before completion.
	// Use this when a job is aborted by the user or the system.
	StatusCancelled Status = "cancelled"

	// StatusTimeout means the job exceeded its allowed execution time and was forcefully stopped.
	// Use this when implementing execution time limits for long-running tasks.
	StatusTimeout Status = "timeout"

	// === Invalid or Unclear States ===
	
	// StatusUnknown means the job's state is unclear due to an unexpected condition.
	// Use this when a job's status cannot be determined, possibly due to system failure.
	StatusUnknown Status = "unknown"

	// StatusInvalid means the job was rejected due to invalid parameters or configuration.
	// Use this when validation fails before the job enters the queue.
	StatusInvalid Status = "invalid"
)

// --- Utility Methods ---

// IsExecution checks if a status is in the execution phase (before completion or failure).
func (s Status) IsExecution() bool {
	return s == StatusCreated || s == StatusPending || s == StatusRunning
}

// IsCompletion checks if a status indicates successful job completion.
func (s Status) IsCompletion() bool {
	return s == StatusCompleted
}

// IsFailure checks if a status indicates job failure or termination.
func (s Status) IsFailure() bool {
	return s == StatusFailed || s == StatusCancelled || s == StatusTimeout || s == StatusUnknown || s == StatusInvalid
}
