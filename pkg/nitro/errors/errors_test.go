package errors

import (
	stdliberrors "errors"
	"fmt"
	"testing"
)

func TestUserError(t *testing.T) {
	orig := stdliberrors.New("something happened")
	ue := User(orig)
	if !IsUserError(ue) {
		t.Fatalf("should have been a user error")
	}
	if IsSystemError(ue) {
		t.Fatalf("should not have been a system error")
	}
	orig = fmt.Errorf("error in foo: %w", orig)
	orig = fmt.Errorf("error in bar: %w", orig)
	orig = fmt.Errorf("error in baz: %w", orig)
	t.Logf("orig: %v", orig)

	ue = User(orig)
	t.Logf("user err: %v", ue)

	if !IsUserError(ue) {
		t.Fatalf("should have been a user error")
	}
	if IsSystemError(ue) {
		t.Fatalf("should not have been a system error")
	}
	if IsUserError(stdliberrors.New("something else")) {
		t.Fatalf("standard error should not have been a user error")
	}

	if IsUserError(nil) {
		t.Fatalf("nil should have returned false")
	}
}

func TestSystemError(t *testing.T) {
	orig := stdliberrors.New("something happened")
	if IsUserError(orig) {
		t.Fatalf("should not have been a user error")
	}
	if !IsSystemError(orig) {
		t.Fatalf("should have been a system error")
	}
	orig = fmt.Errorf("error in foo: %w", orig)
	orig = fmt.Errorf("error in bar: %w", orig)
	orig = fmt.Errorf("error in baz: %w", orig)

	if IsUserError(orig) {
		t.Fatalf("should not have been a user error")
	}
	if !IsSystemError(orig) {
		t.Fatalf("should have been a system error")
	}
}

type customError struct {
	message string
}

func (ce customError) Error() string { return ce.message }

func TestSystemErrorUnwrapped(t *testing.T) {

	ce := customError{message: "custom error"}

	res := customError{}
	if ok := stdliberrors.As(ce, &res); !ok {
		t.Fatalf("expected error to be unwrapped properly and found by errors.As")
	}
}

func TestUserErrorUnwrapped(t *testing.T) {
	ce := customError{message: "custom error"}
	ue := User(ce)

	res := customError{}
	if ok := stdliberrors.As(ue, &res); !ok {
		t.Fatalf("expected error to be unwrapped properly and found by errors.As")
	}
}
