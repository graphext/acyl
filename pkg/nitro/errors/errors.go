package errors

import (
	"strings"

	pkgerrors "github.com/pkg/errors"
)

type operationError struct {
	inner        error
	user, system bool
}

func (oe operationError) Error() string {
	return oe.inner.Error()
}

// causer is from github.com/pkg/errors
type causer interface {
	Cause() error
}

// causeAndWraps unwraps err to the root cause, returning it as well as all wrapped error strings, in reverse order
func causeAndWraps(err error) ([]string, error) {
	wraps := []string{}

	for err != nil {
		errmsg := err.Error()
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
		// a wrapped error actually consists of two nested error values, WithMessage() and WithStack()
		// if the cause has the same error string as the outer error, ignore it
		if err.Error() == errmsg {
			continue
		}
		// remove the inner error message from the outer results in just the outer
		wraps = append(wraps, strings.Replace(errmsg, ": "+err.Error(), "", 1))
		errmsg = err.Error()

	}
	return wraps, err
}

func unwrapIfNeeded(err error, oe operationError) error {
	if err == nil {
		return nil
	}
	_, ok := err.(causer)
	if ok {
		wraps, inner := causeAndWraps(err)
		oe.inner = inner
		err = oe
		// rewrap the inner error in reverse order
		for i := len(wraps) - 1; i >= 0; i-- {
			msg := wraps[i]
			err = pkgerrors.WithMessage(err, msg)
		}
		return err
	}
	oe.inner = err
	return oe
}

// UserError annotates err in such a way that IsUserError() can be used further up in the callstack.
// If err is a wrapped error, the root error will be unwrapped, annotated and rewrapped instead. The root error stack trace is preserved but any middle ones are lost.
func UserError(err error) error {
	return unwrapIfNeeded(err, operationError{user: true})
}

// SystemError annotates err in such a way that IsSystemError() can be used further up in the callstack.
// If err is a wrapped error, the root error will be unwrapped, annotated and rewrapped instead. The root error stack trace is preserved but any middle ones are lost.
func SystemError(err error) error {
	return unwrapIfNeeded(err, operationError{system: true})
}

// IsUserError will unwrap err to the innermost error value and return whether it is a UserError.
func IsUserError(err error) bool {
	switch v := pkgerrors.Cause(err).(type) {
	case operationError:
		return v.user
	default:
		return false
	}
}

// IsSystemError will unwrap err to the innermost error value and return whether it is a SystemError.
func IsSystemError(err error) bool {
	switch v := pkgerrors.Cause(err).(type) {
	case operationError:
		return v.system
	default:
		return false
	}
}
