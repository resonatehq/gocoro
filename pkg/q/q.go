package q

// Queue[T] is a generic FIFO queue implementation
type Queue[T any] []T

// Enqueue adds an item to the end of the queue
func (q *Queue[T]) Enqueue(item T) {
	*q = append(*q, item)
}

// Dequeue removes and returns the item at the front of the queue
func (q *Queue[T]) Dequeue() (T, bool) {
	if len(*q) == 0 {
		var zero T
		return zero, false // return zero value of type T if queue is empty
	}
	item := (*q)[0]
	*q = (*q)[1:]
	return item, true
}
