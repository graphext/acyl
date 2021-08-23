package errors

import "errors"

// A UserError wraps an underlying error to annotate it as being caused by
// user error. The underlying error string is returned directly, meaning the
// annotation can be detected using errors.As without the annotation affecting
// how the errors are communicated.
type UserError struct {
	error
}

// Error returns err's underlying error string.
func (err UserError) Error() string {
	return err.error.Error()
}

// Unwrap returns err's underlying error.
func (err UserError) Unwrap() error {
	return err.error
}

// User annotates err as a user error.
func User(err error) error {
	if err == nil {
		return nil
	}
	return UserError{err}
}

// A CancelledError wraps an underlying error to annotite at as being caused by
// the cancellation of a context. CancelledErrors are also annotated as
// UserErrors.
type CancelledError struct {
	error
}

// Error returns err's underlying error string.
func (err CancelledError) Error() string {
	return err.error.Error()
}

// Unwrap returns err's underlying error.
func (err CancelledError) Unwrap() error {
	return err.error
}

// User annotates err as a context cancelled error.
func Cancelled(err error) error {
	if err == nil {
		return nil
	}
	return CancelledError{User(err)}
}

// IsUserError returns whether err is annotated as a user error.
func IsUserError(err error) bool {
	return errors.As(err, &UserError{})
}

// IsCancelledError returns whether err is annotated as a context cancelled error.
func IsCancelledError(err error) bool {
	return errors.As(err, &CancelledError{})
}

// IsSystemError returns whether err is not annotated as a user error.
func IsSystemError(err error) bool {
	return !errors.As(err, &UserError{})
}
