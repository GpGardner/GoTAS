package main

// //make a simple runner example
// import (
// 	"GOTAS/internal/job"
// 	"GOTAS/internal/runner"
// 	"context"
// 	"fmt"
// 	"log"
// 	"math/rand"
// 	"net/http"
// 	"os"
// 	"runtime/pprof"
// 	"time"
// )

// func work(ctx context.Context, id int) Result {
// 	duration := rand.Intn(2) + 1 // Random duration between 1 and 2 seconds
// 	// Simulate work by sleeping
// 	select {
// 	case <-ctx.Done():
// 		return Result(fmt.Sprintf("Job %d cancelled", id))
// 	case <-time.After(time.Duration(duration) * time.Second):
// 		return Result(fmt.Sprintf("Job %d completed", id))
// 	}
// }

// type Result string // Define a simple type for the result, can be string or any other type

// func main() {
// 	// Enable pprof for profiling
// 	go func() {
// 		log.Println(http.ListenAndServe("localhost:6060", nil))
// 	}()

// 	// Create a context
// 	ctx := context.Background()

// 	r := runner.NewStaticRunner(runner.StrategyParallel{}, func(p *job.Processable[Result]) {
// 		fmt.Println("Runner initialized with strategy:", "StrategyParallel")
// 	})

// 	// Simulate running 5 jobs
// 	for i := 0; i < 5; i++ {
// 		jobInstance, err := job.NewJob(func(ctx context.Context, args ...any) (Result, error) {
// 			result := work(ctx, i)
// 			return result, nil
// 		})
// 		if err != nil {
// 			log.Fatalf("Failed to create job: %v", err)
// 		}
// 		r.AddJob(jobInstance)
// 	}

// 	r.Run(ctx)

// 	fmt.Println("Complete percentage: ", r.CheckProgress())

// 	if f, err := os.Create("cpu.prof"); err == nil {
// 		defer f.Close()
// 		if err := pprof.StartCPUProfile(f); err != nil {
// 			log.Fatal(err)
// 		}
// 		defer pprof.StopCPUProfile()
// 	}
// }
