package gocoro

/////////////////////////////////////////////////////////////////////
// Coroutine
/////////////////////////////////////////////////////////////////////

type iCoroutine[I, O any] interface {
	resume() (*value[I, O], iCoroutine[I, O], iCompletable, bool)
}

type iCompletable interface {
	completed() bool
}

type CoroutineFunc[T, TNext, TReturn any] func(*Coroutine[T, TNext, TReturn]) (TReturn, error)

type Coroutine[T, TNext, TReturn any] struct {
	f CoroutineFunc[T, TNext, TReturn]
	p *Promise[TReturn]

	c_i chan interface{}
	c_o chan *yield[T, TNext, TReturn]
}

type yield[T, TNext, TReturn any] struct {
	value *value[T, TNext]
	spawn iCoroutine[T, TNext]
	await iCompletable
	done  bool
}

type value[T, TNext any] struct {
	value   T
	promise *Promise[TNext]
}

func newCoroutine[T, TNext, TReturn any](f func(*Coroutine[T, TNext, TReturn]) (TReturn, error)) *Coroutine[T, TNext, TReturn] {
	c := &Coroutine[T, TNext, TReturn]{
		f:   f,
		p:   newPromise[TReturn](),
		c_i: make(chan interface{}),
		c_o: make(chan *yield[T, TNext, TReturn]),
	}

	go func() {
		<-c.c_i

		v, e := c.f(c)
		close(c.c_i)

		c.p.complete(v, e)

		c.c_o <- &yield[T, TNext, TReturn]{done: true}
		close(c.c_o)
	}()

	return c
}

func (c *Coroutine[T, TNext, TReturn]) resume() (*value[T, TNext], iCoroutine[T, TNext], iCompletable, bool) {
	c.c_i <- nil

	o := <-c.c_o
	return o.value, o.spawn, o.await, o.done
}
