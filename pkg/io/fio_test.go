package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func greet(name string) func() (string, error) {
	return func() (string, error) {
		return "Hello " + name, nil

	}
}

func callbackThatAsserts(t *testing.T, expected string) func(string, error) {
	return func(s string, err error) {
		assert.Equal(t, s, expected, "Greeting value should be `%d`", expected)
	}
}

func TestFIO(t *testing.T) {
	fio := NewFIO[string](100)

	go fio.Worker()
	defer fio.Shutdown()

	names := []string{"A", "B", "C", "D"}
	expectedGreetings := []string{"Hello A", "Hello B", "Hello C", "Hello D"}

	for i := 0; i < len(names); i++ {
		fio.Dispatch(greet(names[i]), callbackThatAsserts(t, expectedGreetings[i]))
	}

	n := 0
	for n < len(names) {
		cqes := fio.Dequeue(1)
		if len(cqes) > 0 {
			cqes[0].Callback(cqes[0].Value, cqes[0].Error)
			n++
		}
	}
}
