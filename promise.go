package gocoro

/////////////////////////////////////////////////////////////////////
// Promise
/////////////////////////////////////////////////////////////////////

// public

type Promise[T any] struct {
	state int
	value T
	error error

	// notify outside world
	done chan interface{}
}

func newPromise[T any]() *Promise[T] {
	return &Promise[T]{
		done: make(chan interface{}, 1),
	}
}

func (p *Promise[T]) Await() (T, error) {
	if p.pending() {
		<-p.done
	}

	return p.value, p.error
}

// private

func (p *Promise[T]) resolve(v T) {
	p.state = 1
	p.value = v
	close(p.done)
}

func (p *Promise[T]) reject(e error) {
	p.state = 2
	p.error = e
	close(p.done)
}

func (p *Promise[T]) complete(v T, e error) {
	if e != nil {
		p.reject(e)
	} else {
		p.resolve(v)
	}
}

func (p *Promise[T]) pending() bool {
	return p.state == 0
}

func (p *Promise[T]) completed() bool {
	return !p.pending()
}
