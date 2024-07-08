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

	for i, name := range names {
		fio.Enqueue(&SQE[func() (string, error), string]{greet(name), callbackThatAsserts(t, expectedGreetings[i])})
	}

	for range names {
		cqe := <-fio.Dequeue()
		cqe.Callback(cqe.Value, nil)
	}
}