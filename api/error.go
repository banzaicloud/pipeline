package api

import (
	"github.com/banzaicloud/pipeline/secret"
	"github.com/pkg/errors"
)

// isNotFound checks whether an error is about a resource not being found.
func isNotFound(err error) bool {
	// Check the root cause error.
	err = errors.Cause(err)

	if e, ok := err.(interface {
		NotFound() bool
	}); ok {
		return e.NotFound()
	}

	return false
}

// isInvalid checks whether an error is about a resource not being found.
func isInvalid(err error) bool {
	// Check the root cause error.
	err = errors.Cause(err)

	if e, ok := err.(interface {
		IsInvalid() bool
	}); ok {
		return e.IsInvalid()
	}

	switch err {
	case secret.ErrSecretNotExists:
		return true
	}

	switch err.(type) {
	case secret.MissmatchError:
		return true
	}

	return false
}
