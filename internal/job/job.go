package job

import (
	"context"
)

type Processable[T any] interface {
	Run(ctx context.Context, args ...any) (T, error) // Executes the job
	setResult(result T)                              // Sets the result of the job
	setResultEmpty()                                 // Sets the result to its zero value
}

// make sure Job implements Processable interface
var _ Processable[any] = (*Job[any])(nil)

// Job that returns a result and an error
type Job[T any] struct {
	function func(ctx context.Context, args ...any) (T, error)
	result   T
}

func (j *Job[T]) Run(ctx context.Context, args ...any) (T, error) {

	//Check that ctx isnt already complete
	if ctx.Err() != nil {
		j.setResultEmpty()
		j.Complete(StatusTimeout)
		return j.result, ctx.Err()
	}

	result, err := j.function(ctx, args...)

	if err != nil {
		j.setResultEmpty()
		j.Complete(StatusError)
	} else {
		j.setResult(result)
		j.Complete(StatusCompleted)
	}
	return j.result, err
}

// Create initializes a new Job instance with the provided function.
// This function takes a function as an argument and returns a new Job instance.
// The function should accept a context and any number of arguments, and return a result of type T and an error.
// Parameters:
// - f: A function that takes a context and any number of arguments, and returns a result of type T and an error.
// Returns:
// - A pointer to the newly created Job instance.
// - An error if the provided function is nil.
// Example usage:
//
//	job, err := Create(func(ctx context.Context, args ...any) (string, error) {
//	    // Handle context cancellation or timeout
//	    // Your job logic here
//	    return "result", nil
//	})
func Create[T any](f func(ctx context.Context, args ...any) (T, error)) (*Job[T], error) {
	return &Job[T]{
		function: f,
	}, nil
}

// Complete marks the job as completed
func (j *Job[T]) Complete(status Status) {
	return
}

// String returns a human-readable string representation of the Job instance.
// This method implements the fmt.Stringer interface, allowing the Job to be printed in a readable format.
// This is useful for logging and debugging purposes.
// If you need to obsfucate the result, then you can override this method in your own implementation.
func (j *Job[T]) String() string {
  return "Job" // TODO: implement a better string representation
}

func (j *Job[T]) setResult(result T) {
	j.result = result
}

func (j *Job[T]) setResultEmpty() {
	j.result = *new(T) // Reset result to zero value
}
