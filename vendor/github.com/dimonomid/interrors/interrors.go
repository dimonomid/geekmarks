// Copyright 2018 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the MIT, see LICENSE file for details.

/*
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

Currently the `interror` package depends on a couple of implementation details
of `errors`: namely, it calls the `Underlying` method however the documentation
says that clients should never call this method; and also internal error
wrappers satisfies the unexported interface `errors.causer`.

See README.md for more details.
*/

package interrors

import "github.com/juju/errors"

// WrapInternalError wraps two errors: internal intError and external pubError.
// All standard operations on the resulting error will behave as if pubError
// was the original cause, thus hiding the internal error, which is only
// retrievable with the accessor functions below.
func WrapInternalError(
	intError, pubError error,
) error {
	if !IsInternalError(intError) {
		return &internalErrorWrapper{
			pubError: pubError,
			intError: intError,
		}
	} else {
		return errors.Annotate(intError, pubError.Error())
	}
}

// WrapInternalErrorf takes an internal error, creates a new (public) error
// with the formatted message, and calls WrapInternalError with those two
// errors.
func WrapInternalErrorf(
	intError error, pubMessageFormat string, args ...interface{},
) error {
	return WrapInternalError(intError, errors.Errorf(pubMessageFormat, args...))
}

// InternalErr takes an error, and if it's an internal error wrapper, returns
// the underlying internal error (which might still be wrapped with
// errors.Trace() or some such). Otherwise, just returns the given err back.
//
// err is an "internal error wrapper" if it's a value returned from
// WrapInternalError, which can be further wrapped into errors.Trace() or
// friends.
func InternalErr(err error) error {
	ierr, _ := internalErrCheck(err)
	return ierr
}

// InternalCause is like InternalErr, but it returns the original cause of the
// error.
func InternalCause(err error) error {
	return errors.Cause(InternalErr(err))
}

// IsInternalError reports whether the given err is an internal error wrapper.
func IsInternalError(err error) bool {
	_, ok := internalErrCheck(err)
	return ok
}

// ErrorStack is similar to errors.ErrorStack, but it also includes the stack
// of the internal error, if given err is an internal error wrapper.
func ErrorStack(err error) string {
	st := ""
	if intError, ok := internalErrCheck(err); ok {
		st += errors.ErrorStack(intError) + "\n"
	}
	st += errors.ErrorStack(err)
	return st
}

func internalErrCheck(err error) (ierr error, ok bool) {
	cerr := err

	for {
		if cerr == nil {
			return err, false
		}

		if ieWrapper, ok := cerr.(*internalErrorWrapper); ok {
			return ieWrapper.intError, true
		}

		cerr2, ok := cerr.(*errors.Err)
		if !ok {
			return err, false
		}

		cerr = cerr2.Underlying()
	}
}
