package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewJobBase(t *testing.T) {
	jb := NewJob()

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
	j := &JobWithError{
		jobBase: NewJob(),
		function: func(ctx context.Context, args ...any) error {
			return nil
		},
	}

	ctx := context.Background()
	err := j.Run(ctx)

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
	expectedErr := errors.New("job failed")
	j := &JobWithError{
		jobBase: NewJob(),
		function: func(ctx context.Context, args ...any) error {
			return expectedErr
		},
	}

	ctx := context.Background()
	err := j.Run(ctx)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if j.Status() != StatusError {
		t.Errorf("expected status to be StatusError, got %v", j.Status())
	}
}

func TestJobWithError_Run_Timeout(t *testing.T) {
	j := &JobWithError{
		jobBase: NewJob(),
		function: func(ctx context.Context, args ...any) error {
			time.Sleep(2 * time.Second)
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := j.Run(ctx)

	if err != context.DeadlineExceeded {
		t.Errorf("expected error %v, got %v", context.DeadlineExceeded, err)
	}

	if j.Status() != StatusTimeout {
		t.Errorf("expected status to be StatusTimeout, got %v", j.Status())
	}
}

func TestJobWithResult_Run_Success(t *testing.T) {
	expectedResult := "success"
	j := &JobWithResult[string]{
		jobBase: NewJob(),
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
	expectedErr := errors.New("job failed")
	j := &JobWithResult[string]{
		jobBase: NewJob(),
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
	j := &JobWithResult[string]{
		jobBase: NewJob(),
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
