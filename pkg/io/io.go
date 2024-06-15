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

type FIO[I func() (O, error), O any] struct {
	sq chan *SQE[I, O]
	cq chan *CQE[O]
}

func NewFIO[O any](size int) *FIO[func() (O, error), O] {
	fio := &FIO[func() (O, error), O]{
		sq: make(chan *SQE[func() (O, error), O], size),
		cq: make(chan *CQE[O], size),
	}

	return fio
}

func (fio *FIO[I, O]) Enqueue(sqe *SQE[I, O]) {
	fio.sq <- sqe
}

func (fio *FIO[I, O]) Dequeue() <-chan *CQE[O] {
	return fio.cq
}

func (fio *FIO[I, O]) Shutdown() {
	close(fio.sq)
	close(fio.cq)
}

func (fio *FIO[I, O]) Worker() {
	for sqe := range fio.sq {
		v, e := sqe.Value()
		fio.cq <- &CQE[O]{
			Value:    v,
			Error:    e,
			Callback: sqe.Callback,
		}
	}
}
