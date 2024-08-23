package io

// SQE / CQE

type sqe[I func() (O, error), O any] struct {
	Value    I
	Callback func(O, error)
}

type cqe[O any] struct {
	Value    O
	Error    error
	Callback func(O, error)
}

// FIO

type FIO[I func() (O, error), O any] struct {
	sq chan *sqe[I, O]
	cq chan *cqe[O]
}

func NewFIO[O any](size int) *FIO[func() (O, error), O] {
	fio := &FIO[func() (O, error), O]{
		sq: make(chan *sqe[func() (O, error), O], size),
		cq: make(chan *cqe[O], size),
	}

	return fio
}

func (fio *FIO[I, O]) Dispatch(value I, callback func(O, error)) {
	fio.sq <- &sqe[I, O]{
		Value:    value,
		Callback: callback,
	}
}

func (fio *FIO[I, O]) Enqueue(cqe *cqe[O]) {
	fio.cq <- cqe
}

func (fio *FIO[I, O]) Dequeue(n int) []*cqe[O] {
	cqes := []*cqe[O]{}
	for i := 0; i < n; i++ {
		select {
		case cqe, ok := <-fio.cq:
			if !ok {
				return cqes
			}
			cqes = append(cqes, cqe)
		default:
			break
		}
	}

	return cqes
}

func (fio *FIO[I, O]) Shutdown() {
	close(fio.sq)
	close(fio.cq)
}

func (fio *FIO[I, O]) Worker() {
	for sqe := range fio.sq {
		v, e := sqe.Value()
		fio.Enqueue(&cqe[O]{
			Value:    v,
			Error:    e,
			Callback: sqe.Callback,
		})
	}
}
