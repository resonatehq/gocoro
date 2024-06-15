package gocoro

import (
	"sync"

	"github.com/resonatehq/gocoro/pkg/io"
	"github.com/resonatehq/gocoro/pkg/q"
)

/////////////////////////////////////////////////////////////////////
// Scheduler
/////////////////////////////////////////////////////////////////////

type Scheduler[I, O any] struct {
	sync.Mutex

	io io.IO[I, O]
	in chan iCoroutine[I, O]

	done chan interface{}

	runnable q.Queue[iCoroutine[I, O]]
	awaiting q.Queue[*awaitingCoroutine[I, O]]
}

type awaitingCoroutine[I, O any] struct {
	coroutine iCoroutine[I, O]
	on        iCompletable
}

func (s *Scheduler[I, O]) add(c iCoroutine[I, O]) {
	s.in <- c
}

func (s *Scheduler[I, O]) Run() {
	s.Lock()
	defer s.Unlock()

	s.run(false)
}

func (s *Scheduler[I, O]) RunUntilComplete() {
	s.Lock()
	defer s.Unlock()

	s.run(true)
}

func (s *Scheduler[I, O]) RunUntilBlocked() {
	s.Lock()
	defer s.Unlock()

	s.runUntilBlocked(nil, nil)
}

func (s *Scheduler[I, O]) Shutdown() {
	close(s.done)
}

func (s *Scheduler[I, O]) run(breakOnComplete bool) {
	for {
		select {
		case crt := <-s.in:
			s.runUntilBlocked(crt, nil)
		case cqe := <-s.io.Dequeue():
			s.runUntilBlocked(nil, cqe)
		case <-s.done:
			return
		}

		assert(len(s.runnable) == 0, "runnable should be empty")

		if breakOnComplete && len(s.awaiting) == 0 {
			break
		}
	}
}

func (s *Scheduler[I, O]) runUntilBlocked(crt iCoroutine[I, O], cqe *io.CQE[O]) {
	assert(crt == nil || cqe == nil, "one or both of crt/cqe should be nil")

	var cqes []*io.CQE[O]

	if crt != nil {
		s.runnable.Enqueue(crt)
	}

	if cqe != nil {
		cqes = append(cqes, cqe)
	}

	// exhaust in
	batch(s.in, 10, func(crt iCoroutine[I, O]) {
		s.runnable.Enqueue(crt)
	})

	// exhaust cq
	batch(s.io.Dequeue(), 10, func(cqe *io.CQE[O]) {
		cqes = append(cqes, cqe)
	})

	// tick
	for _, sqe := range s.Tick(cqes) {
		s.io.Enqueue(sqe)
	}

	assert(len(s.runnable) == 0, "runnable should be empty")
}

func (s *Scheduler[I, O]) Tick(cqes []*io.CQE[O]) []*io.SQE[I, O] {
	var sqes []*io.SQE[I, O]

	// cqes
	for _, cqe := range cqes {
		assert(cqe.Callback != nil, "callback should not be nil")
		cqe.Callback(cqe.Value, cqe.Error)
	}

	// move unblocked coroutines from awaiting to runnable
	s.unblock()

	// coroutines
	for coroutine := range s.runnable.Pop() {
		value, spawn, await, done := coroutine.resume()

		if value != nil {
			sqes = append(sqes, &io.SQE[I, O]{
				Value:    value.value,
				Callback: value.promise.complete,
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
	}

	return sqes
}

func (s *Scheduler[I, O]) unblock() {
	i := 0
	for _, coroutine := range s.awaiting {
		if coroutine.on.completed() {
			s.runnable.Enqueue(coroutine.coroutine)
		} else {
			s.awaiting[i] = coroutine
			i++
		}
	}

	s.awaiting = s.awaiting[:i]
}

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
