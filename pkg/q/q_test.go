package q

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueue(t *testing.T) {
	q := Queue[int]{}
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	qe, _ := q.Dequeue()
	assert.Equal(t, qe, 1, "Value should be 1")
	expected := 2
	for qe := range q.Pop() {
		assert.Equal(t, qe, expected, "expected %d, got %d", expected, qe)
		expected++
	}
}
