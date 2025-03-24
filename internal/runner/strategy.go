package runner

// Strategy represents the execution strategy for a job.
type Strategy string

const (
	// StrategySequential means jobs are executed one after the other in the order they were added.
	// Use this when jobs have dependencies or must run in a specific sequence.
	StrategySequential Strategy = "sequential"

	// StrategyParallel means jobs are executed concurrently in no particular order.
	// Use this when jobs are independent and can run in parallel for faster processing.
	StrategyParallel Strategy = "parallel"

	// StrategyBatch means jobs are executed in batches with a fixed size and order.
	// Use this when jobs can be grouped together for more efficient processing.
	StrategyBatch Strategy = "batch"

	// StrategyPriority means jobs are executed based on priority level.
	// Use this when jobs have different levels of importance or urgency.
	StrategyPriority Strategy = "priority"

	// StrategyRetry means jobs are retried a fixed number of times before being marked as failed.
	// Use this when jobs are expected to fail occasionally and can be retried automatically.
	StrategyRetry Strategy = "retry"

	// StrategyCron means jobs are executed at specific intervals or times based on a cron expression.
	// Use this when jobs need to run periodically or at scheduled times.
	StrategyCron Strategy = "cron"
)

// --- Utility Methods ---
// IsSequential checks if a strategy requires sequential execution of jobs.
func (s Strategy) IsSequential() bool {
	return s == StrategySequential
}

// IsParallel checks if a strategy allows parallel execution of
// jobs.
func (s Strategy) IsParallel() bool {
	return s == StrategyParallel
}

// IsBatch checks if a strategy groups jobs into batches for
// processing.
func (s Strategy) IsBatch() bool {
	return s == StrategyBatch
}

// IsPriority checks if a strategy prioritizes jobs based on
// importance or urgency.
func (s Strategy) IsPriority() bool {
	return s == StrategyPriority
}

// IsRetry checks if a strategy retries failed jobs automatically.
func (s Strategy) IsRetry() bool {
	return s == StrategyRetry
}

// IsCron checks if a strategy schedules jobs based on a cron
// expression.
func (s Strategy) IsCron() bool {
	return s == StrategyCron
}

// IsValid checks if a strategy is recognized.
func (s Strategy) IsValid() bool {
	switch s {
	case StrategySequential, StrategyParallel, StrategyBatch, StrategyPriority, StrategyRetry, StrategyCron:
		return true
	default:
		return false
	}
}
