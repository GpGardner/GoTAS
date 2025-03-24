package main

import "fmt"

// JobError represents a custom error type for the job scheduler.
type JobError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// ErrorCode represents various error codes that can occur in the job scheduler.
type ErrorCode string

const (
	ErrorInvalidStatus   ErrorCode = "invalid_status"
	ErrorJobTimeout      ErrorCode = "job_timeout"
	ErrorJobFailure      ErrorCode = "job_failure"
	ErrorJobCancelled    ErrorCode = "job_cancelled"
	ErrorJobNotFound     ErrorCode = "job_not_found"
	ErrorInvalidFunction ErrorCode = "invalid_function"
)

// NewJobError creates a new JobError instance.
func NewJobError(code ErrorCode, message string, err error) *JobError {
	return &JobError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Error implements the error interface for JobError.
func (e *JobError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// --- Example Errors ---

var (
	ErrInvalidFunction  = NewJobError(ErrorInvalidFunction, "The function provided is invalid", nil)
	ErrInvalidJobStatus = NewJobError(ErrorInvalidStatus, "The job status is invalid for this operation", nil)
	ErrJobTimeout       = NewJobError(ErrorJobTimeout, "The job has timed out", nil)
	ErrJobFailure       = NewJobError(ErrorJobFailure, "The job has failed due to an error", nil)
	ErrJobCancelled     = NewJobError(ErrorJobCancelled, "The job has been manually cancelled", nil)
	ErrJobNotFound      = NewJobError(ErrorJobNotFound, "The job could not be found", nil)
)
