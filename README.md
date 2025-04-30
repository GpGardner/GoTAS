# GoTAS

GoTAS (Go Task Automation System) is a flexible and efficient task runner built in Go. It provides dynamic strategies for executing jobs, including parallel, sequential, retry-based, and priority-based execution. The system is designed to handle high concurrency and offers customizable configurations for developers.

---

## Features

- **Dynamic Strategies**: Supports multiple execution strategies:
  - Parallel execution
  - Sequential execution
  - Retry-based execution
  - Fail-fast execution
  - Priority-based execution
- **Customizable Configuration**: Adjust worker count, channel size, retry logic, and more using a builder pattern.
- **Graceful Shutdown**: Ensures all workers shut down cleanly, even with long-running jobs.
- **Job Tracking**: Tracks completed, failed, and total jobs.
- **Callback Support**: Allows consumers to define custom logic after job execution.

---

# TODO: Complete the rest of this. Code is moderatly documented but will include more examples in the README. runner_test has a few good examples

## Usage
### Example: Creating a Dynamic Runner
```go
package main

import (
    "context"
    "fmt"
    "time"
    "GoTAS/internal/runner"
)

func main() {
    runner := runner.NewRunnerBuilder[Result]().
        WithStrategy(runner.StrategyParallel{}).
        WithWorkers(5).
        WithChanSize(100).
        WithBackoff(2 * time.Second).
        WithMaxRetries(3).
        Build()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start the runner
    runner.Run(ctx)

    // Add jobs asynchronously
    go func() {
        for i := 0; i < 1000; i++ { // Adjust the number of jobs as needed
            job, _ := runner.NewJob(func(ctx context.Context, args ...any) (Result, error) {
                time.Sleep(1 * time.Second) // Simulate job processing
                return Result{ID: i, Message: "Job completed"}, nil
            })
            runner.AddJob(job)
        }
    }()

    // Shutdown the runner
    runner.Shutdown()
}
```

## License

This code is proprietary and not currently open for any use, modification, or extension without explicit permission from the copyright holder. 

If you have questions regarding usage or licensing, please contact [gpgardner@yahoo.com].

## Contact
For questions or support, please contact [gpgardner@yahoo.com]