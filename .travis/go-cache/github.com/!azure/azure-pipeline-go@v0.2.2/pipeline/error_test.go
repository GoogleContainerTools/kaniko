package pipeline

import (
	"testing"

	"github.com/pkg/errors"
)

func TestErrorWithCause(t *testing.T) {
	rootErr := errors.New("root cause error")
	pipeErr := NewError(rootErr, "pipeline wrapper error")
	wrapErr := errors.Wrap(pipeErr, "wrap with stack trace")
	causeErr := errors.Cause(wrapErr)
	if causeErr == nil {
		t.Fatal("cause error should not be nil")
	}
	if causeErr != rootErr {
		t.Fatal("cause error should be the same as root error")
	}
}

func TestErrorWithoutCause(t *testing.T) {
	pipeErr := NewError(nil, "pipeline error without cause")
	wrapErr := errors.Wrap(pipeErr, "wrap with stack trace")
	causeErr := errors.Cause(wrapErr)
	if causeErr == nil {
		t.Fatal("cause error should not be nil")
	}
	if causeErr != pipeErr {
		t.Fatal("cause error should be the same as pipeline error")
	}
}
