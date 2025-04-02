package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewJobBase(t *testing.T) {
	jb, err := NewJobBase()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if jb.ID() == uuid.Nil {
		t.Errorf("expected a valid UUID, got nil")
	}

	if jb.Status() != StatusPending {
		t.Errorf("expected status to be StatusPending, got %v", jb.Status())
	}

	if jb.CreatedAt().IsZero() {
		t.Errorf("expected createdAt to be set, got zero value")
	}
}

func TestJobWithError_Run_Success(t *testing.T) {

	jb, err := NewJobBase()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	j := &Job[any]{ // Using Job[any] to match the signature of the function
		jobBase: jb,
		function: func(ctx context.Context, args ...any) (any, error) {
			return nil, nil
		},
	}

	ctx := context.Background()
	_, err = j.Run(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if j.Status() != StatusCompleted {
		t.Errorf("expected status to be StatusCompleted, got %v", j.Status())
	}

	if j.CompletedAt().IsZero() {
		t.Errorf("expected completedAt to be set, got zero value")
	}
}

func TestJobWithError_Run_Error(t *testing.T) {
	jb, err := NewJobBase()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	expectedErr := ErrJobFailure
	j := &Job[any]{
		jobBase: jb,
		function: func(ctx context.Context, args ...any) (any, error) {
			return nil, expectedErr
		},
	}

	ctx := context.Background()
	_, err = j.Run(ctx)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if j.Status() != StatusError {
		t.Errorf("expected status to be StatusError, got %v", j.Status())
	}
}

func TestJobWithError_Run_Timeout(t *testing.T) {
	jb, err := NewJobBase()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	j := &Job[any]{
		jobBase: jb,
		function: func(ctx context.Context, args ...any) (any, error) {
			time.Sleep(2 * time.Second)
			return nil, nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = j.Run(ctx)

	if err != context.DeadlineExceeded {
		t.Errorf("expected error %v, got %v", context.DeadlineExceeded, err)
	}

	if j.Status() != StatusTimeout {
		t.Errorf("expected status to be StatusTimeout, got %v", j.Status())
	}
}

func TestJobWithResult_Run_Success(t *testing.T) {
	jb, err := NewJobBase()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	expectedResult := "success"
	j := &Job[string]{
		jobBase: jb,
		function: func(ctx context.Context, args ...any) (string, error) {
			return expectedResult, nil
		},
	}

	ctx := context.Background()
	result, err := j.Run(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result != expectedResult {
		t.Errorf("expected result %v, got %v", expectedResult, result)
	}

	if j.Status() != StatusCompleted {
		t.Errorf("expected status to be StatusCompleted, got %v", j.Status())
	}
}

func TestJobWithResult_Run_Error(t *testing.T) {
	jb, err := NewJobBase()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	expectedErr := errors.New("job failed")
	j := &Job[string]{
		jobBase: jb,
		function: func(ctx context.Context, args ...any) (string, error) {
			return "", expectedErr
		},
	}

	ctx := context.Background()
	result, err := j.Run(ctx)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if result != "" {
		t.Errorf("expected result to be empty, got %v", result)
	}

	if j.Status() != StatusError {
		t.Errorf("expected status to be StatusError, got %v", j.Status())
	}
}

func TestJobWithResult_Run_Timeout(t *testing.T) {
	jb, err := NewJobBase()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	j := &Job[string]{
		jobBase: jb,
		function: func(ctx context.Context, args ...any) (string, error) {
			time.Sleep(2 * time.Second)
			return "timeout", nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := j.Run(ctx)

	if err != context.DeadlineExceeded {
		t.Errorf("expected error %v, got %v", context.DeadlineExceeded, err)
	}

	if result != "" {
		t.Errorf("expected result to be empty, got %v", result)
	}

	if j.Status() != StatusTimeout {
		t.Errorf("expected status to be StatusTimeout, got %v", j.Status())
	}
}
