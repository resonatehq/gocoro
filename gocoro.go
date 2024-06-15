package gocoro

import (
	"github.com/resonatehq/gocoro/pkg/io"
)

/////////////////////////////////////////////////////////////////////
// Public API
/////////////////////////////////////////////////////////////////////

func New[I, O any](io io.IO[I, O], size int) *Scheduler[I, O] {
	return &Scheduler[I, O]{
		io:   io,
		in:   make(chan iCoroutine[I, O], size),
		done: make(chan interface{}),
	}
}

func Add[T, TNext, TReturn any](s *Scheduler[T, TNext], f CoroutineFunc[T, TNext, TReturn]) *Promise[TReturn] {
	coroutine := newCoroutine(f)
	s.add(coroutine)

	return coroutine.p
}

func Yield[T, TNext, TReturn any](c *Coroutine[T, TNext, TReturn], v T) *Promise[TNext] {
	p := newPromise[TNext]()

	c.c_o <- &yield[T, TNext, TReturn]{value: &value[T, TNext]{value: v, promise: p}, done: false}
	<-c.c_i

	return p
}

func Spawn[T, TNext, TReturn, R any](c *Coroutine[T, TNext, TReturn], f CoroutineFunc[T, TNext, R]) *Promise[R] {
	coroutine := newCoroutine(f)

	c.c_o <- &yield[T, TNext, TReturn]{spawn: coroutine, done: false}
	<-c.c_i

	return coroutine.p
}

func Await[T, TNext, TReturn, P any](c *Coroutine[T, TNext, TReturn], p *Promise[P]) (P, error) {
	if p.pending() {
		c.c_o <- &yield[T, TNext, TReturn]{await: p, done: false}
		<-c.c_i
	}

	return p.value, p.error
}
