package promise

type Promise[T any] interface {
	Awaitable[T]
	Await() (T, error)
}

type Awaitable[T any] interface {
	Value() T
	Error() error
	Pending() bool
	Completed() bool
}

type promise[T any] struct {
	state int
	value T
	error error
	done  chan interface{}
}

func New[T any]() *promise[T] {
	return &promise[T]{
		done: make(chan interface{}, 1),
	}
}

func (p *promise[T]) Await() (T, error) {
	if p.Pending() {
		<-p.done
	}

	return p.value, p.error
}

func (p *promise[T]) Resolve(v T) {
	p.state = 1
	p.value = v
	close(p.done)
}

func (p *promise[T]) Reject(e error) {
	p.state = 2
	p.error = e
	close(p.done)
}

func (p *promise[T]) Complete(v T, e error) {
	if e != nil {
		p.Reject(e)
	} else {
		p.Resolve(v)
	}
}

func (p *promise[T]) Pending() bool {
	return p.state == 0
}

func (p *promise[T]) Completed() bool {
	return !p.Pending()
}

func (p *promise[T]) Value() T {
	return p.value
}

func (p *promise[T]) Error() error {
	return p.error
}
