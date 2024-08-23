package main

import (
	"fmt"

	"github.com/resonatehq/gocoro"
	"github.com/resonatehq/gocoro/pkg/io"
)

func coroutine(n int) gocoro.CoroutineFunc[func() (string, error), string, string] {
	return func(c gocoro.Coroutine[func() (string, error), string, string]) (string, error) {
		fmt.Println("coroutine:", n)

		if n == 0 {
			return "", nil
		}

		fooPromise := gocoro.Yield(c, func() (string, error) {
			return fmt.Sprintf("foo.%d", n), nil
		})

		barPromise := gocoro.Yield(c, func() (string, error) {
			return fmt.Sprintf("bar.%d", n), nil
		})

		bazPromise := gocoro.Spawn(c, coroutine(n-1))

		foo, _ := gocoro.Await(c, fooPromise)
		bar, _ := gocoro.Await(c, barPromise)
		baz, _ := gocoro.Await(c, bazPromise)

		return fmt.Sprintf("%s:%s:%s", foo, bar, baz), nil
	}
}

func main() {
	// instantiate io
	fio := io.NewFIO[string](100)

	// start io worker on a goroutine
	go fio.Worker()

	// instantiate scheduler
	scheduler := gocoro.New(fio, 100)

	// add coroutine to scheduler
	promise, _ := gocoro.Add(scheduler, coroutine(3))

	// run scheduler until blocked repeatedly
	for scheduler.Size() > 0 {
		for _, cqe := range fio.Dequeue(3) {
			cqe.Callback(cqe.Value, cqe.Error)
		}
		scheduler.RunUntilBlocked(0)
	}

	// shutdown scheduler
	scheduler.Shutdown()

	// await and print result
	if v, e := promise.Await(); e != nil {
		fmt.Println("error:", e)
	} else {
		fmt.Println("value:", v)
	}
}
