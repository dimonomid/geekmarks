internal errors
===============

The package `interrors` is an addition to the
[`juju/errors`](https://github.com/juju/errors); `interrors` provides helpers
to handle internal errors which should not be reported to the user.

When building, say, a backend service, we often want to distinguish public
errors (e.g. caused by the invalid input received from a client) and internal
errors, like the failure to execute SQL query or something else. In the former
case, the client should be provided with the descriptive error message like
`handling bookmarks: saving bookmark "foo bar": URL can't be empty`. In the
latter case though, the client should only get `internal error`, and we want to
log the exact (and descriptive) internal error for us to examine.

Consider:

```go
errOrig := errors.New("specific error")

err := errOrig
err = errors.Annotatef(err, "some context")
err = errors.Annotatef(err, "some more context")

fmt.Println(err) // Prints "some more context: some context: specific error"
```
Now, if it was actually an internal error which we don't want to show to a
client, here's what we can do:

```go
internalServerError := errors.New("internal server error")
err = interrors.WrapInternalError(err, internalServerError)
```

So from now on, `err` can be further wrapped using `errors.Annotatef()` or
friends, but it will behave like the cause was `internalServerError`, not
`errOrig`.

```go
err = errors.Annotatef(err, "some public context")

fmt.Println(err)  // Prints "some public context: internal server error"
errors.Cause(err) // Returns internalServerError
errors.ErrorStack(err) // Returns stack up to internalServerError, not further
```

And, of course, there are helpers to get the details of an internal error back.

```go
fmt.Println(interrors.InternalErr(err))   // Prints "some more context: some context: specific error"
fmt.Println(interrors.InternalCause(err)) // Prints "specific error"
fmt.Println(interrors.ErrorStack(err))    // Prints full error stack, from "some public context" to "specific error"
```

The error stack printed by `interrors.ErrorStack(err)` would look like:

```
/path/to/file.go:46: specific error
/path/to/file.go:49: some context
/path/to/file.go:50: some more context
internal server error
/path/to/file.go:55: some public context
```
