package scheduler

import (
	"github.com/resonatehq/gocoro/pkg/io"
	"github.com/resonatehq/gocoro/pkg/promise"
	"github.com/resonatehq/gocoro/pkg/q"
)

/////////////////////////////////////////////////////////////////////
// Scheduler
/////////////////////////////////////////////////////////////////////

type Coroutine[I, O any] interface {
	Resume() (I, Promise[O], Coroutine[I, O], Completable, bool)
	SetTime(int64)
}

type Promise[T any] interface {
	promise.Promise[T]
	Complete(T, error)
}

type Completable interface {
	Pending() bool
	Completed() bool
}

type Scheduler[I, O any] struct {
	io io.IO[I, O]
	in chan Coroutine[I, O]

	runnable q.Queue[Coroutine[I, O]]
	awaiting q.Queue[*awaitingCoroutine[I, O]]

	closed bool

	// add the history
}

func New[I, O any](io io.IO[I, O], size int) *Scheduler[I, O] {
	return &Scheduler[I, O]{
		io: io,
		in: make(chan Coroutine[I, O], size),
	}
}

type awaitingCoroutine[I, O any] struct {
	coroutine Coroutine[I, O]
	on        Completable
}

func (s *Scheduler[I, O]) Add(c Coroutine[I, O]) bool {
	if s.closed {
		return false
	}

	select {
	case s.in <- c:
		return true
	default:
		return false
	}
}

func (s *Scheduler[I, O]) RunUntilBlocked(time int64, cqes []io.QE) {
	batch(s.in, len(s.in), func(c Coroutine[I, O]) {
		s.runnable.Enqueue(c)
	})

	for _, cqe := range cqes {
		cqe.Invoke()
	}

	// tick
	s.Tick(time)

	assert(len(s.runnable) == 0, "runnable should be empty")
}

func (s *Scheduler[I, O]) Tick(time int64) {
	// move unblocked coroutines from awaiting to runnable
	s.unblock()

	// coroutines
	for {
		if ok := s.Step(time); !ok {
			break
		}
	}
}

func (s *Scheduler[I, O]) Step(time int64) bool {
	coroutine, ok := s.runnable.Dequeue()
	if !ok {
		return false
	}

	// set the time
	coroutine.SetTime(time)

	// resume the coroutine
	value, promise, spawn, await, done := coroutine.Resume()

	if promise != nil {
		// enqueue sqe
		s.io.Enqueue(value, promise.Complete)

		// put on runnable
		s.runnable.Enqueue(coroutine)
	} else if spawn != nil {
		// put new coroutine on runnable
		s.runnable.Enqueue(spawn)

		// put old coroutine on runnable
		s.runnable.Enqueue(coroutine)
	} else if await != nil {
		// put on awaiting
		s.awaiting.Enqueue(&awaitingCoroutine[I, O]{
			coroutine: coroutine,
			on:        await,
		})
	} else if done {
		// move unblocked coroutines from awaiting to runnable
		s.unblock()
	} else {
		assert(false, "unreachable")
	}

	return true
}

func (s *Scheduler[I, O]) Size() int {
	return len(s.runnable) + len(s.awaiting) + len(s.in)
}

func (s *Scheduler[I, O]) Shutdown() {
	s.closed = true
	close(s.in)
}

func (s *Scheduler[I, O]) unblock() {
	i := 0
	for _, coroutine := range s.awaiting {
		if coroutine.on.Completed() {
			s.runnable.Enqueue(coroutine.coroutine)
		} else {
			s.awaiting[i] = coroutine
			i++
		}
	}

	s.awaiting = s.awaiting[:i]
}

/////////////////////////////////////////////////////////////////////
// Helpers
/////////////////////////////////////////////////////////////////////

func assert(cond bool, mesg string) {
	if !cond {
		panic(mesg)
	}
}

func batch[T any](c <-chan T, n int, f func(T)) {
	for i := 0; i < n; i++ {
		select {
		case e, ok := <-c:
			if !ok {
				return
			}
			f(e)
		default:
			return
		}
	}
}
