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

	qe1, _ := q.Dequeue()
	assert.Equal(t, qe1, 1, "Value should be 1")

	qe2, _ := q.Dequeue()
	assert.Equal(t, qe2, 2, "Value should be 2")

	qe3, _ := q.Dequeue()
	assert.Equal(t, qe3, 3, "Value should be 3")
}
