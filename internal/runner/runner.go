package runner

import (
	. "GOTAS/internal/job"
	"context"
)

// Package runner provides a flexible job runner that supports multiple execution strategies.
// It allows jobs to be executed sequentially, in parallel, or based on custom strategies like priority or fail-fast.
// The runner is designed to handle job execution efficiently while providing progress tracking and optional callbacks.

type Runnable[T any] interface {
	Run(ctx context.Context)
	AddJob(j Processable[T]) error
	CheckProgress() float64
}
