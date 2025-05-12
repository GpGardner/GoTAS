package runner

import (
	"context"
	"fmt"
	"sync"
	"testing"

	. "GOTAS/internal/job"
)

func TestNewStaticRunner(t *testing.T) {
	runner := NewStaticBuilder[Result]().
		WithStrategy(StrategySequential{}).
		Build()
	if runner == nil {
		t.Fatal("Expected runner to be created, got nil")
	}
	if runner.strategy == nil {
		t.Fatal("Expected strategy to be set, got nil")
	}
}

func TestAddJob(t *testing.T) {
	runner := NewStaticBuilder[Result]().
		WithStrategy(StrategySequential{}).
		Build()
	job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
		return Result{}, nil
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}
	err = runner.AddJob(job)
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}
	if len(runner.jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(runner.jobs))
	}
}

func TestCheckProgress(t *testing.T) {
	runner := NewStaticBuilder[Result]().
		WithStrategy(StrategySequential{}).
		Build()
	if progress := runner.CheckProgress(); progress != 0.0 {
		t.Fatalf("Expected progress to be 0.0, got %.2f", progress)
	}

	runner.totalJobs = 10
	runner.completedJobs = 5
	if progress := runner.CheckProgress(); progress != 50.0 {
		t.Fatalf("Expected progress to be 50.0, got %.2f", progress)
	}
}

func TestRunSequential(t *testing.T) {
	runner := NewStaticBuilder[Result]().
		WithStrategy(StrategySequential{}).
		Build()
	job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
		return Result{ID: 1}, nil
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}
	runner.AddJob(job)

	ctx := context.Background()
	runner.Run(ctx)

	if runner.completedJobs != 1 {
		t.Fatalf("Expected 1 completed job, got %d", runner.completedJobs)
	}
}

func TestRunParallel(t *testing.T) {
	runner := NewStaticBuilder[Result]().
		WithStrategy(StrategyParallel{}).
		Build()
	var mu sync.Mutex
	executedJobs := 0

	for i := 0; i < 5; i++ {
		job, err := Create(func(ctx context.Context, args ...any) (Result, error) {
			mu.Lock()
			executedJobs++
			mu.Unlock()
			return Result{ID: i}, nil
		})
		if err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}
		runner.AddJob(job)
	}

	ctx := context.Background()
	runner.Run(ctx)

	if executedJobs != 5 {
		t.Fatalf("Expected 5 executed jobs, got %d", executedJobs)
	}
}

func TestRunFailFast(t *testing.T) {
	runner := NewStaticBuilder[Result]().
		WithStrategy(StrategyFailFast{}).
		Build()
	job1, err := Create(func(ctx context.Context, args ...any) (Result, error) {
		return Result{ID: 1}, nil
	})
	if err != nil {
		t.Fatalf("Failed to create job1: %v", err)
	}
	job2, err := Create(func(ctx context.Context, args ...any) (Result, error) {
		return Result{}, fmt.Errorf("job failed")
	})
	if err != nil {
		t.Fatalf("Failed to create job2: %v", err)
	}
	job3, err := Create(func(ctx context.Context, args ...any) (Result, error) {
		t.Fatal("This job should not have been executed")
		return Result{ID: 3}, nil
	})
	if err != nil {
		t.Fatalf("Failed to create job3: %v", err)
	}

	runner.AddJob(job1)
	runner.AddJob(job2)
	runner.AddJob(job3)

	ctx := context.Background()
	runner.Run(ctx)

	if runner.completedJobs != 2 {
		t.Fatalf("Expected 2 completed jobs, got %d", runner.completedJobs)
	}
}

func TestRunPriority(t *testing.T) {
	runner := NewStaticBuilder[Result]().
		WithStrategy(StrategyPriority{}).
		Build()

	job1, _ := Create(func(ctx context.Context, args ...any) (Result, error) {
		return Result{ID: 1}, nil
	})
	job2, _ := Create(func(ctx context.Context, args ...any) (Result, error) {
		return Result{ID: 2}, nil
	})
	job3, _ := Create(func(ctx context.Context, args ...any) (Result, error) {
		return Result{ID: 3}, nil
	})

	// Create PriorityJob as values, not pointers
	priorityJob1, _ := NewPriorityJob(job1, 1)
	priorityJob2, _ := NewPriorityJob(job2, 3)
	priorityJob3, _ := NewPriorityJob(job3, 2)

	runner.AddJob(priorityJob1)
	runner.AddJob(priorityJob2)
	runner.AddJob(priorityJob3)

	ctx := context.Background()
	runner.Run(ctx)

	if runner.completedJobs != 3 {
		t.Fatalf("Expected 3 completed jobs, got %d", runner.completedJobs)
	}
}

func TestIncrementCompletedJobs(t *testing.T) {
	runner := NewStaticBuilder[Result]().
		WithStrategy(StrategySequential{}).
		Build()
	runner.incrementCompletedJobs()
	if runner.completedJobs != 1 {
		t.Fatalf("Expected 1 completed job, got %d", runner.completedJobs)
	}
}

func TestIncrementTotalJobs(t *testing.T) {
	runner := NewStaticBuilder[Result]().
		WithStrategy(StrategySequential{}).
		Build()
	runner.incrementTotalJobs()
	if runner.totalJobs != 1 {
		t.Fatalf("Expected 1 total job, got %d", runner.totalJobs)
	}
}
