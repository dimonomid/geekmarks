// +build all_tests unit_tests

package interror

import (
	"testing"

	"github.com/juju/errors"
)

func TestInternalError(t *testing.T) {
	errOrig := errors.Errorf("some internal error: %s", "foo")
	err := errors.Annotatef(errOrig, "annotation 1")
	err = errors.Annotatef(err, "annotation 2")

	errPub := errors.Errorf("my public error: %s", "bar")
	errPubWrap := WrapInternalError(err, errPub)
	err2 := errors.Annotatef(errPubWrap, "pub annotation 1")
	err2 = errors.Annotatef(err2, "pub annotation 2")

	exp := "my public error: bar"
	if errPub.Error() != exp {
		t.Errorf("errPub.Error(): expected %q, got %q", exp, errPub.Error())
	}

	gotCause := errors.Cause(err2)
	if gotCause != errPub {
		t.Errorf("errPub is wrong: want: %q, got %q", errPub, gotCause)
	}

	{
		gotIntCause := InternalCause(err2)
		if gotIntCause != errOrig {
			t.Errorf("InternalCause(%v): want: %q, got %q", err2, errOrig, gotIntCause)
		}
	}

	{
		gotIntCause := InternalCause(err)
		if gotIntCause != errOrig {
			t.Errorf("InternalCause(%v): want: %q, got %q", err, errOrig, gotIntCause)
		}
	}

	{
		gotIntErr := InternalErr(err2)
		if gotIntErr != err {
			t.Errorf("InternalErr(%v): want: %q, got %q", err2, err, gotIntErr)
		}
	}

	{
		gotIntErr := InternalErr(err)
		if gotIntErr != err {
			t.Errorf("InternalErr(%v): want: %q, got %q", err, err, gotIntErr)
		}
	}

	if !IsInternalError(err2) {
		t.Errorf("IsInternalError(%v) should be true, got false", err2)
	}

	if !IsInternalError(errPubWrap) {
		t.Errorf("IsInternalError(%v) should be true, got false", errPubWrap)
	}

	if IsInternalError(err) {
		t.Errorf("IsInternalError(%v) should be false, got true", err)
	}
}

func TestDoubleInternal(t *testing.T) {
	errOrig := errors.Errorf("some internal error: %s", "foo")
	err := errors.Annotatef(errOrig, "annotation 1")
	err = errors.Annotatef(err, "annotation 2")

	errPub := errors.Errorf("my public error: %s", "bar")
	errPubWrap := WrapInternalError(err, errPub)
	err2 := errors.Annotatef(errPubWrap, "pub annotation 1")
	err2 = errors.Annotatef(err2, "pub annotation 2")

	errPub2 := WrapInternalErrorf(err2, "my public error2: %s", "baz")
	err3 := errors.Annotatef(errPub2, "pub2 annotation 1")
	err3 = errors.Annotatef(err3, "pub2 annotation 2")

	gotCause := errors.Cause(err3)
	if gotCause != errPub {
		t.Errorf("errPub is wrong: want: %q, got %q", errPub, gotCause)
	}

	gotIntErr := InternalErr(err3)
	if gotIntErr != err {
		t.Errorf("InternalErr is wrong: want: %q, got %q", err, gotIntErr)
	}
}
