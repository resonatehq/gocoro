package io

/////////////////////////////////////////////////////////////////////
// IO
/////////////////////////////////////////////////////////////////////

type IO[I, O any] interface {
	Enqueue(*SQE[I, O])
	Dequeue() <-chan *CQE[O]
	Shutdown()
}

type SQE[I, O any] struct {
	Value    I
	Callback func(O, error)
}

type CQE[O any] struct {
	Value    O
	Error    error
	Callback func(O, error)
}
