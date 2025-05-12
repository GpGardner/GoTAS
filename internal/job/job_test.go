package job

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

var timerPool = sync.Pool{
	New: func() interface{} {
		return time.NewTimer(0)
	},
}

func getTimer(d time.Duration) *time.Timer {
	t := timerPool.Get().(*time.Timer)
	if !t.Stop() {
		<-t.C // Drain the channel if it has a value
	}
	t.Reset(d)
	return t
}

func putTimer(t *time.Timer) {
	if !t.Stop() {
		<-t.C // Drain the channel if it has a value
	}
	timerPool.Put(t)
}

func TestJobWithError_Run_Success(t *testing.T) {
	j := &Job[any]{ // Using Job[any] to match the signature of the function
		function: func(ctx context.Context, args ...any) (any, error) {
			return nil, nil
		},
	}

	ctx := context.Background()
	_, err := j.Run(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

}

func TestJobWithError_Run_Error(t *testing.T) {
	expectedErr := ErrJobFailure
	j, err := Create(func(ctx context.Context, args ...any) (any, error) {
		return nil, expectedErr
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	ctx := context.Background()
	_, err = j.Run(ctx)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}
func TestJobWithError_Run_Timeout(t *testing.T) {
	j, err := Create(func(ctx context.Context, args ...any) (any, error) {
		timer := getTimer(2 * time.Second)
		defer putTimer(timer)
		<-timer.C
		return nil, nil
	})
	if err != nil {
		t.Error(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = j.Run(ctx)

	if err != context.DeadlineExceeded {
		t.Errorf("expected error %v, got %v", context.DeadlineExceeded, err)
	}
}

func TestJobWithResult_Run_Success(t *testing.T) {
	expectedResult := "success"
	j, err := Create(func(ctx context.Context, args ...any) (string, error) {
		return expectedResult, nil
	})
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	result, err := j.Run(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result != expectedResult {
		t.Errorf("expected result %v, got %v", expectedResult, result)
	}
}

func TestJobWithResult_Run_Error(t *testing.T) {
	expectedErr := errors.New("job failed")
	j, err := Create(func(ctx context.Context, args ...any) (string, error) {
		return "", expectedErr
	})
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	result, err := j.Run(ctx)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if result != "" {
		t.Errorf("expected result to be empty, got %v", result)
	}
}

func TestJobWithResult_Run_Timeout(t *testing.T) {
	j, err := Create(func(ctx context.Context, args ...any) (string, error) {
		time.Sleep(2 * time.Second)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "timeout", nil
	})
	if err != nil {
		t.Error(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := j.Run(ctx)

	if err != context.DeadlineExceeded {
		t.Errorf("expected error %v, got %v", context.DeadlineExceeded, err)
	}

	if result != "" {
		t.Errorf("expected result to be empty, got %v", result)
	}
}
