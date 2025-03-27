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
func jobCallback(job *job.Job[any]) {
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

	f, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Println("could not create CPU profile:", err)
		return
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	var jobs []*job.JobWithError

	for i := 0; i <= 1000; i++ {
		f := func(ctx context.Context, args ...any) error {
			result := work(ctx, i)
			if !result.Success {
				return fmt.Errorf("job %d failed", i)
			}
			return nil
		}
		job, err := job.NewJobWithError(f)
		if err != nil {
			panic("Time to party")
		}
		jobs = append(jobs, job)
	}

	runner := runner.NewRunner(runner.StrategyParallel, jobCallback)
	for i := 0; i <= 1000; i++ {
		runner.AddJob(jobs[i])
	}
	defer cancel()
	runner.Run(ctx)
	

	for i := 0; i <= 1000; i++ {

		fmt.Println(jobs[i].String())
	}

	fmt.Println(runner.CheckProgress())

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
