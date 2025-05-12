package job

import (
	"context"
)

// PriorityJob is a wrapper around a Processable job that adds priority functionality.
// It allows jobs to be assigned a priority level (0-255), where higher values indicate higher priority.
// This wrapper forwards all Processable methods to the wrapped job while adding priority-specific behavior.
type PriorityJob[T any] struct {
	job      Processable[T] // The wrapped job that implements the Processable interface.
	priority uint8          // Priority of the job (0-255). Higher values indicate higher priority.
}

// Ensure PriorityJob implements the Processable interface.
// This guarantees that PriorityJob can be used wherever a Processable is expected.
var _ Processable[any] = (*PriorityJob[any])(nil)

// NewPriorityJob creates a new PriorityJob instance with the given job and priority.
// This function wraps an existing Processable job and assigns it a priority level.
//
// Parameters:
// - job: The Processable job to wrap. Must not be nil.
// - priority: The priority level of the job (0-255). Higher values indicate higher priority.
//
// Returns:
// - A pointer to the newly created PriorityJob instance.
// - An error if the provided job is nil.
//
// Example:
//
//	job := &Job[Result]{ID: 1}
//	priorityJob, err := NewPriorityJob(job, 10)
//	if err != nil {
//	    log.Fatalf("Failed to create PriorityJob: %v", err)
//	}
func NewPriorityJob[T any](job Processable[T], priority uint8) (*PriorityJob[T], error) {
	if job == nil {
		return nil, ErrInvalidJobStatus // or return an error based on your requirements
	}

	// Create and return a new PriorityJob instance
	return &PriorityJob[T]{
		job:      job,
		priority: priority,
	}, nil
}

// Run executes the wrapped job's Run method.
// This method forwards the call to the wrapped job, ensuring that the job's logic is executed.
//
// Parameters:
// - ctx: The context for managing cancellation and timeouts.
// - args: Additional arguments required by the job.
//
// Returns:
// - The result of the job execution.
// - An error if the job fails.
func (p *PriorityJob[T]) Run(ctx context.Context, args ...any) (T, error) {
	return p.job.Run(ctx)
}

// Complete marks the wrapped job as complete with the given status.
// This method forwards the call to the wrapped job.
//
// Parameters:
// - status: The completion status of the job.
func (p *PriorityJob[T]) Complete(status Status) {
	return
}

// GetPriority retrieves the priority level of the PriorityJob. Used for sorting or prioritizing jobs in a queue.
// This method is specific to PriorityJob and does not exist in the wrapped job.
//
// Returns:
// - The priority level of the job (0-255). Higher values indicate higher priority.
func (p *PriorityJob[T]) GetPriority() uint8 {
	return p.priority
}

func (j *Job[T]) WithPriority(i uint8) *PriorityJob[T] {
	// This method allows a Processable to be wrapped in a PriorityJob
	if j == nil {
		return nil
	}
	job, _ := NewPriorityJob(j, i)
	return job
}

// setResult sets the result of the job to the provided value.
// This method is used when the job completes successfully.
// It ensures that the result is stored in the job's state, allowing it to be retrieved later.
// This wraps the setResult method of the wrapped job.
func (j *PriorityJob[T]) setResult(result T) {
	j.job.setResult(result)
}

// setResultEmpty sets the result of the job to its zero value.
// This method is used when the job fails or encounters an error.
// It ensures that the result is reset to a known state, preventing any stale data from being returned.
// This wraps the setResultEmpty method of the wrapped job.
func (j *PriorityJob[T]) setResultEmpty() {
	j.job.setResultEmpty()
}
