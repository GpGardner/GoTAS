package job

// PriorityQueue implements a priority queue for jobs.
// It is a slice of PriorityJob[T] that uses the container/heap interface to manage jobs based on their priority.
// Jobs with higher priority values are dequeued first.
//
// This implementation assumes that PriorityJob[T] wraps a Processable job and provides a GetPriority() method.
// The priority queue is designed to be used with Go's container/heap package.
//
// Example Usage:
//
//	pq := &PriorityQueue[Result]{}
//	heap.Init(pq)
//	heap.Push(pq, PriorityJob[Result]{job: job1, priority: 10})
//	highestPriorityJob := heap.Pop(pq).(PriorityJob[Result])
type PriorityQueue[T any] []PriorityJob[T]

// Len returns the number of jobs in the priority queue.
// This method is required by the container/heap interface.
//
// Returns:
// - The number of jobs in the priority queue.
func (pq PriorityQueue[T]) Len() int { return len(pq) }

// Less compares the priority of two jobs in the priority queue.
// Jobs with higher priority values are considered "less" to ensure they are dequeued first.
//
// Parameters:
// - i: The index of the first job to compare.
// - j: The index of the second job to compare.
//
// Returns:
// - true if the job at index i has a higher priority than the job at index j.
// - false otherwise.
func (pq PriorityQueue[T]) Less(i, j int) bool {
	return pq[i].GetPriority() > pq[j].GetPriority() // Higher priority first
}

// Swap swaps the positions of two jobs in the priority queue.
// This method is required by the container/heap interface.
//
// Parameters:
// - i: The index of the first job to swap.
// - j: The index of the second job to swap.
func (pq PriorityQueue[T]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

// Push adds a new job to the priority queue.
// This method is required by the container/heap interface.
//
// Parameters:
// - x: The job to add to the priority queue. Must be of type PriorityJob[T].
func (pq *PriorityQueue[T]) Push(x *PriorityJob[T]) {
	job := x
	*pq = append(*pq, *job)
}

// Pop removes and returns the job with the highest priority from the priority queue.
// This method is required by the container/heap interface.
//
// Returns:
// - The job with the highest priority, of type PriorityJob[T].
func (pq *PriorityQueue[T]) Pop() *PriorityJob[T] {
	old := *pq
	n := len(old)
	item := old[n-1] // Get the last item (highest priority due to heap ordering)
	*pq = old[0 : n-1]
	return &item
}
