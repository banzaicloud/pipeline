package objectstore

import "github.com/pkg/errors"

type errAlreadyExists interface {
	AlreadyExists() bool
}

// IsAlreadyExistsError checks if an error indicates an already existing bucket.
func IsAlreadyExistsError(err error) bool {
	err = errors.Cause(err)

	if err, ok := err.(errAlreadyExists); ok {
		return err.AlreadyExists()
	}

	return false
}

type errNotFound interface {
	NotFound() bool
}

// IsNotFoundError checks if an error indicates a missing bucket.
func IsNotFoundError(err error) bool {
	err = errors.Cause(err)

	if err, ok := err.(errNotFound); ok {
		return err.NotFound()
	}

	return false
}
