package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime/pprof"

	_ "net/http/pprof"
	"time"

	"GOTAS/internal/job"
	"GOTAS/internal/runner"
)

type Result struct {
	JobID   int
	Success bool
}

func (r Result) String() string {
	return fmt.Sprintf("JobID: %d, Success: %t", r.JobID, r.Success)
}

func work(ctx context.Context, id int) Result {
	duration := time.Second * time.Duration(rand.Intn(2)+1)
	time.Sleep(duration)
	return Result{JobID: id, Success: true}
}

// Inner runner job function
func innerRunnerJob(ctx context.Context, outerJobID int, innerJobID int) (Result, error) {
	result := work(ctx, innerJobID)
	if !result.Success {
		return Result{JobID: innerJobID, Success: false}, fmt.Errorf("inner job %d failed", innerJobID)
	}
	return result, nil
}

// Outer runner job function
func outerRunnerJob(ctx context.Context, outerJobID int) (Result, error) {
	// Create an inner runner
	innerRunner := runner.NewRunner(runner.StrategyPriority, nil)

	// Add 10 jobs to the inner runner
	var innerJobs []*job.Job[Result]
	for j := 0; j < 10; j++ {
		innerJobFunc := func(ctx context.Context, args ...any) (Result, error) {
			return innerRunnerJob(ctx, outerJobID, j)
		}

		innerJob, err := job.NewJob(innerJobFunc)
		if err != nil {
			return Result{JobID: outerJobID, Success: false}, fmt.Errorf("failed to create inner job: %v", err)
		}
		innerJobs = append(innerJobs, innerJob)
		innerRunner.AddJob(innerJob)
	}

	// Run the inner runner
	innerRunner.Run(ctx)

	// Collect results from the inner runner
	fmt.Printf("Results for Outer Job %d:\n", outerJobID)
	for _, innerJob := range innerJobs {
		fmt.Println(innerJob.String())
	}

	// Return a successful result for the outer job
	return Result{JobID: outerJobID, Success: true}, nil
}

// Callback function example
func jobCallback(job *job.Processable[any]) {
	// Log the job completion
	j := *job
	j.GetDuration()

	log.Printf("Status: %s, Duration: %s\n", j.GetStatus(), j.GetDuration())

	// // Additional actions can be taken here, like:
	// // - Saving results to a database
	// // - Sending notifications or alerts
	// // - Triggering subsequent jobs based on the result
	// if job.Status == StatusFailed {
	// 	log.Printf("Job %s failed with error: %v", job.ID, job.Error)
	// 	// Take action for failure, like retries or alerts
	// }
}

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start CPU profiling
	f, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Println("could not create CPU profile:", err)
		return
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// Create the outer runner
	outerRunner := runner.NewRunner(runner.StrategyParallel, nil)

	// Add 3 jobs to the outer runner
	var outerJobs []*job.Job[Result]
	for i := 0; i < 3; i++ {
		outerJobFunc := func(ctx context.Context, args ...any) (Result, error) {
			return outerRunnerJob(ctx, i)
		}

		outerJob, err := job.NewJob(outerJobFunc)
		if err != nil {
			log.Fatalf("Failed to create outer job: %v", err)
		}
		outerJobs = append(outerJobs, outerJob)
		outerRunner.AddJob(outerJob)
	}

	// Run the outer runner
	outerRunner.Run(ctx)

	// Collect results from the outer runner
	fmt.Println("Outer Runner Results:")
	for _, outerJob := range outerJobs {
		result := outerJob.GetResult().(Result)
		fmt.Println(outerJob.String() + " - " + result.String())
	}

	// Print progress
	fmt.Println("Final Progress:", outerRunner.CheckProgress(), "%")

	// Start memory profiling
	f, err = os.Create("mem.prof")
	if err != nil {
		fmt.Println("could not create memory profile:", err)
		return
	}
	pprof.WriteHeapProfile(f)
	f.Close()

	log.Println("Starting server on localhost:6060...")
	log.Fatal(http.ListenAndServe("localhost:6060", nil))
}
