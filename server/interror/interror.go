// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package interror // import "dmitryfrank.com/geekmarks/server/interror"

import "github.com/juju/errors"

type InternalErrorWrapper struct {
	pubError error
	intError error
}

func (e *InternalErrorWrapper) Error() string {
	return e.pubError.Error()
}

func (e *InternalErrorWrapper) Cause() error {
	return e.pubError
}

func WrapInternalError(
	intError, pubError error,
) error {
	if !IsInternalError(intError) {
		return &InternalErrorWrapper{
			pubError: pubError,
			intError: intError,
		}
	} else {
		return errors.Annotate(intError, pubError.Error())
	}
}

func WrapInternalErrorf(
	intError error, pubMessageFormat string, args ...interface{},
) error {
	return WrapInternalError(intError, errors.Errorf(pubMessageFormat, args...))
}

func InternalErrCheck(err error) (ierr error, ok bool) {
	cerr := err

	for {
		if cerr == nil {
			return err, false
		}

		if ieWrapper, ok := cerr.(*InternalErrorWrapper); ok {
			return ieWrapper.intError, true
		}

		cerr2, ok := cerr.(*errors.Err)
		if !ok {
			return err, false
		}

		cerr = cerr2.Underlying()
	}
}

func InternalErr(err error) error {
	ierr, _ := InternalErrCheck(err)
	return ierr
}

func InternalCause(err error) error {
	return errors.Cause(InternalErr(err))
}

func IsInternalError(err error) bool {
	_, ok := InternalErrCheck(err)
	return ok
}

func ErrorStack(err error) string {
	st := ""
	if intError, ok := InternalErrCheck(err); ok {
		st += errors.ErrorStack(intError) + "\n"
	}
	st += errors.ErrorStack(err)
	return st
}
