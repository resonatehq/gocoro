package io

// SQE / CQE

type sqe[I func() (O, error), O any] struct {
	value    I
	callback func(O, error)
}

type cqe[O any] struct {
	value    O
	error    error
	callback func(O, error)
}

func (cqe *cqe[O]) Invoke() {
	cqe.callback(cqe.value, cqe.error)
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

func (fio *FIO[I, O]) Enqueue(value I, callback func(O, error)) {
	fio.sq <- &sqe[I, O]{
		value:    value,
		callback: callback,
	}
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
		v, e := sqe.value()
		fio.cq <- &cqe[O]{
			value:    v,
			error:    e,
			callback: sqe.callback,
		}
	}
}
