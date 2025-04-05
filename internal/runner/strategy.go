package runner

// Strategy represents the execution strategy for a job.
type Strategy string

type StaticStrategy interface {
	isStaticCompatible()
}

type DynamicStrategy interface {
	isDynamicCompatible()
}

// StrategySequential represents a strategy where jobs are executed sequentially.
type StrategySequential struct{}
type StrategyFIFO = StrategySequential // Alias FIFO to StrategySequential for clarity in usage

// StrategyParallel represents a strategy where jobs are executed in parallel.
type StrategyParallel struct{}

// StrategyFailFast represents a strategy where execution stops on the first failure.
type StrategyFailFast struct{}

// StrategyRetry represents a strategy where failed jobs are retried automatically.
type StrategyRetry struct{}

// StrategyPriority means jobs are executed based on priority level.
// Use this when jobs have different levels of importance or urgency.
type StrategyPriority struct{}

func (s StrategyParallel) isDynamicCompatible()   {}
func (s StrategySequential) isDynamicCompatible() {}
func (s StrategyFailFast) isDynamicCompatible()   {}
func (s StrategyRetry) isDynamicCompatible()      {}
func (s StrategyPriority) isDynamicCompatible()   {}

func (s StrategyParallel) isStaticCompatible()   {}
func (s StrategySequential) isStaticCompatible() {}
func (s StrategyFailFast) isStaticCompatible()   {}
func (s StrategyRetry) isStaticCompatible()      {}
func (s StrategyPriority) isStaticCompatible()   {}
