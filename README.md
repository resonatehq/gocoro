# gocoro

A coroutine library for go that supports async await semantics.

## Usage

```go
return func(c *gocoro.Coroutine[func() (string, error), string, string]) (string, error) {

  // yield a function
  p := gocoro.Yield(c, func() (string, error) {
    return "foo", nil
  })

  // yield a coroutine
  c := gocoro.Spawn(c, func(c *gocoro.Coroutine[func() (string, error), string, string]) (string, error) {
    return "bar", nil
  })

  // await function
  foo, _ := gocoro.Await(c, p)

  // await coroutine
  bar, _ := gocoro.Await(c, c)

  return foo + bar, nil
}
```

See the [example](example/main.go) for complete usage details.
