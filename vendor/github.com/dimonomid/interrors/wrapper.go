// Copyright 2018 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the MIT, see LICENSE file for details.

package interrors

type internalErrorWrapper struct {
	pubError error
	intError error
}

func (e *internalErrorWrapper) Error() string {
	return e.pubError.Error()
}

// internalErrorWrapper must satisfy non-exported interface errors.causer
func (e *internalErrorWrapper) Cause() error {
	return e.pubError
}
