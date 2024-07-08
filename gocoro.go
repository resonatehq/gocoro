package gocoro

import (
	"github.com/resonatehq/gocoro/pkg/io"
	"github.com/resonatehq/gocoro/pkg/promise"
	"github.com/resonatehq/gocoro/pkg/scheduler"
)

/////////////////////////////////////////////////////////////////////
// Public API
/////////////////////////////////////////////////////////////////////

// TODO: add comment about generic type stuff

type Scheduler[I, O any] interface {
	Add(scheduler.Coroutine[I, O])

	Run()
	RunUntilComplete()
	RunUntilBlocked()
	Shutdown()
	Done() bool

	// lower level apis
	Tick(int64)
	Step(int64) bool
}

func New[I, O any](io io.IO[I, O], size int, batchSize int) Scheduler[I, O] {
	return scheduler.New(io, size, batchSize)
}

func NewDST[I, O any](io io.IO[I, O], batchSize int) Scheduler[I, O] {
	return scheduler.NewDST(io, batchSize)
}

func Add[T, TNext, TReturn any](s Scheduler[T, TNext], f CoroutineFunc[T, TNext, TReturn]) promise.Promise[TReturn] {
	coroutine := newCoroutine(f)
	s.Add(coroutine)

	return coroutine.p
}

func Yield[T, TNext, TReturn any](c Coroutine[T, TNext, TReturn], v T) promise.Awaitable[TNext] {
	p := promise.New[TNext]()
	c.yieldAndAwait(&yield[T, TNext, TReturn]{value: v, promise: p, done: false})

	return p
}

func Spawn[T, TNext, TReturn, R any](c Coroutine[T, TNext, TReturn], f CoroutineFunc[T, TNext, R]) promise.Awaitable[R] {
	coroutine := newCoroutine(f)
	c.yieldAndAwait(&yield[T, TNext, TReturn]{spawn: coroutine, done: false})

	return coroutine.p
}

func Await[T, TNext, TReturn, P any](c Coroutine[T, TNext, TReturn], p promise.Awaitable[P]) (P, error) {
	if p.Pending() {
		c.yieldAndAwait(&yield[T, TNext, TReturn]{await: p, done: false})
	}

	assert(p.Completed(), "promise must be completed")

	return p.Value(), p.Error()
}

/////////////////////////////////////////////////////////////////////
// Coroutine
/////////////////////////////////////////////////////////////////////

type Coroutine[T, TNext, TReturn any] interface {
	Time() int64
	yieldAndAwait(*yield[T, TNext, TReturn])
}

type CoroutineFunc[T, TNext, TReturn any] func(Coroutine[T, TNext, TReturn]) (TReturn, error)

type coroutine[T, TNext, TReturn any] struct {
	f CoroutineFunc[T, TNext, TReturn]
	p scheduler.Promise[TReturn]
	t int64

	c_i chan interface{}
	c_o chan *yield[T, TNext, TReturn]
}

type yield[T, TNext, TReturn any] struct {
	value   T
	promise scheduler.Promise[TNext]
	spawn   scheduler.Coroutine[T, TNext]
	await   scheduler.Completable
	done    bool
}

func newCoroutine[T, TNext, TReturn any](f CoroutineFunc[T, TNext, TReturn]) *coroutine[T, TNext, TReturn] {
	c := &coroutine[T, TNext, TReturn]{
		f:   f,
		p:   promise.New[TReturn](),
		c_i: make(chan interface{}),
		c_o: make(chan *yield[T, TNext, TReturn]),
	}

	go func() {
		<-c.c_i

		v, e := c.f(c)
		c.p.Complete(v, e)
		close(c.c_i)

		c.c_o <- &yield[T, TNext, TReturn]{done: true}
		close(c.c_o)
	}()

	return c
}

func (c *coroutine[T, TNext, TReturn]) Resume() (T, scheduler.Promise[TNext], scheduler.Coroutine[T, TNext], scheduler.Completable, bool) {
	c.c_i <- nil

	o := <-c.c_o
	return o.value, o.promise, o.spawn, o.await, o.done
}

func (c *coroutine[T, TNext, TReturn]) SetTime(t int64) {
	c.t = t
}

func (c *coroutine[T, TNext, TReturn]) Time() int64 {
	return c.t
}

// nolint:unused
func (c *coroutine[T, TNext, TReturn]) yieldAndAwait(y *yield[T, TNext, TReturn]) {
	c.c_o <- y
	<-c.c_i
}

func assert(cond bool, mesg string) {
	if !cond {
		panic(mesg)
	}
}
