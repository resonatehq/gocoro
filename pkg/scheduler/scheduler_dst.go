package scheduler

import (
	"github.com/resonatehq/gocoro/pkg/io"
	"github.com/resonatehq/gocoro/pkg/q"
)

/////////////////////////////////////////////////////////////////////
// Scheduler
/////////////////////////////////////////////////////////////////////

type dstScheduler[I, O any] struct {
	batchSize int

	io io.IO[I, O]
	in []Coroutine[I, O]

	runnable q.Queue[Coroutine[I, O]]
	awaiting q.Queue[*awaitingCoroutine[I, O]]

	// add the history
}

func NewDST[I, O any](io io.IO[I, O], batchSize int) *dstScheduler[I, O] {
	return &dstScheduler[I, O]{
		io:        io,
		in:        []Coroutine[I, O]{},
		batchSize: batchSize,
	}
}

func (s *dstScheduler[I, O]) Add(c Coroutine[I, O]) {
	s.in = append(s.in, c)
}

func (s *dstScheduler[I, O]) Run() {
	panic("not implemented in dst")
}

func (s *dstScheduler[I, O]) RunUntilComplete() {
	panic("not implemented in dst")
}

func (s *dstScheduler[I, O]) RunUntilBlocked() {
	panic("not implemented in dst")
}

func (s *dstScheduler[I, O]) Shutdown() {}

func (s *dstScheduler[I, O]) Tick(time int64) {
	// exhaust in
	s.runnable = append(s.runnable, s.in[:s.batchSize]...)

	// exhaust cq
	batch(s.io.Dequeue(), s.batchSize, func(cqe *io.CQE[O]) {
		assert(cqe.Callback != nil, "callback should not be nil")
		cqe.Callback(cqe.Value, cqe.Error)
	})

	// move unblocked coroutines from awaiting to runnable
	s.unblock()

	// coroutines
	for {
		if ok := s.Step(time); !ok {
			break
		}
	}
}

func (s *dstScheduler[I, O]) Step(time int64) bool {
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

func (s *dstScheduler[I, O]) unblock() {
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
