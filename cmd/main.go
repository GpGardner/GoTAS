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
	JobID    int
	Duration time.Duration
	Success  bool
}

func work(ctx context.Context, id int) Result {
	duration := time.Second * time.Duration(rand.Intn(2)+1)
	time.Sleep(duration)
	return Result{JobID: id, Duration: duration, Success: true}
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
	totalJobs := 10 // Total number of jobs to create

	f, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Println("could not create CPU profile:", err)
		return
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	var jobs []*job.Job[Result]
	// Define empty type for the sake of example
	empty := Result{} // Empty type to satisfy the job's generic type

	for i := 0; i <= totalJobs; i++ {
		f := func(ctx context.Context, args ...any) (Result, error) {
			result := work(ctx, i)
			if !result.Success {
				return empty, fmt.Errorf("job %d failed", i)
			}
			return result, nil
		}

		if i == 6 { // Simulate a failure for job 653 to test error handling
			f = func(ctx context.Context, args ...any) (Result, error) {
				// Simulate a failure for job 653
				return empty, fmt.Errorf("simulated failure for job 653")
			}
		}

		job, err := job.NewJob(f)
		if err != nil {
			panic("Time to party")
		}
		jobs = append(jobs, job)
	}

	r := runner.NewRunner(runner.StrategyParallel, jobCallback)
	for i := 0; i <= totalJobs; i++ {
		r.AddJob(jobs[i])
	}

	defer cancel()
	r.Run(ctx)

	// TODO: Pull result from the jobs

	// Print the progress of the seq runner
	fmt.Println("Runner Progress:")
	for i := 0; i <= totalJobs; i++ {

		fmt.Println(jobs[i].String())
	}

	fmt.Println(r.CheckProgress())

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
