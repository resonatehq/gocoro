package scheduler

import (
	"sync"
	"time"

	"github.com/resonatehq/gocoro/pkg/io"
	"github.com/resonatehq/gocoro/pkg/promise"
	"github.com/resonatehq/gocoro/pkg/q"
)

/////////////////////////////////////////////////////////////////////
// Scheduler
/////////////////////////////////////////////////////////////////////

type Scheduler[I, O any] interface {
	Add(Coroutine[I, O])

	Run()
	RunUntilComplete()
	RunUntilBlocked()
	Shutdown()
}

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

type scheduler[I, O any] struct {
	lock sync.Mutex

	batchSize int

	io io.IO[I, O]
	in chan Coroutine[I, O]

	done chan interface{}

	runnable q.Queue[Coroutine[I, O]]
	awaiting q.Queue[*awaitingCoroutine[I, O]]

	// add the history
}

func New[I, O any](io io.IO[I, O], size int, batchSize int) *scheduler[I, O] {
	return &scheduler[I, O]{
		io:        io,
		in:        make(chan Coroutine[I, O], size),
		done:      make(chan interface{}),
		batchSize: batchSize,
	}
}

type awaitingCoroutine[I, O any] struct {
	coroutine Coroutine[I, O]
	on        Completable
}

func (s *scheduler[I, O]) Add(c Coroutine[I, O]) {
	s.in <- c
}

func (s *scheduler[I, O]) Run() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.run(false)
}

func (s *scheduler[I, O]) RunUntilComplete() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.run(true)
}

func (s *scheduler[I, O]) RunUntilBlocked() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.runUntilBlocked(time.Now().UnixMilli(), nil, nil)
}

func (s *scheduler[I, O]) Shutdown() {
	close(s.done)
}

func (s *scheduler[I, O]) run(breakOnComplete bool) {
	for {
		select {
		case crt := <-s.in:
			s.runUntilBlocked(time.Now().UnixMilli(), crt, nil)
		case cqe := <-s.io.Dequeue():
			s.runUntilBlocked(time.Now().UnixMilli(), nil, cqe)
		case <-s.done:
			return
		}

		assert(len(s.runnable) == 0, "runnable should be empty")

		if breakOnComplete && len(s.awaiting) == 0 {
			break
		}
	}
}

func (s *scheduler[I, O]) runUntilBlocked(time int64, crt Coroutine[I, O], cqe *io.CQE[O]) {
	assert(crt == nil || cqe == nil, "one or both of crt/cqe should be nil")

	if crt != nil {
		s.runnable.Enqueue(crt)
	}

	if cqe != nil {
		assert(cqe.Callback != nil, "callback should not be nil")
		cqe.Callback(cqe.Value, cqe.Error)
	}

	// exhaust in
	batch(s.in, s.batchSize, func(crt Coroutine[I, O]) {
		s.runnable.Enqueue(crt)
	})

	// exhaust cq
	batch(s.io.Dequeue(), s.batchSize, func(cqe *io.CQE[O]) {
		assert(cqe.Callback != nil, "callback should not be nil")
		cqe.Callback(cqe.Value, cqe.Error)
	})

	// tick
	s.Tick(time)

	assert(len(s.runnable) == 0, "runnable should be empty")
}

func (s *scheduler[I, O]) Tick(time int64) {
	// move unblocked coroutines from awaiting to runnable
	s.unblock()

	// coroutines
	for {
		if ok := s.Step(time); !ok {
			break
		}
	}
}

func (s *scheduler[I, O]) Step(time int64) bool {
	coroutine, ok := s.runnable.Dequeue()
	if !ok {
		return false
	}

	// set the time
	coroutine.SetTime(time)

	value, promise, spawn, await, done := coroutine.Resume()

	if promise != nil {
		s.io.Enqueue(&io.SQE[I, O]{
			Value:    value,
			Callback: promise.Complete,
		})

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

func (s *scheduler[I, O]) unblock() {
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
		case e := <-c:
			f(e)
		default:
			return
		}
	}
}
