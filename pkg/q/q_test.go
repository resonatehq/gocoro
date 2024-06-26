package q

import "testing"

func TestQueue(t *testing.T) {
	q := Queue[int]{}
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	qe, _ := q.Dequeue()
	if qe != 1 {
		t.Errorf("expected 1, got %d", qe)
	}

	expected := 2
	for qe := range q.Pop() {
		if qe != expected {
			t.Errorf("expected %d, got %d", expected, qe)
		}
		expected++
	}
}
