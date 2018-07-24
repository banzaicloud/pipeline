package oci

import "fmt"

// EntityNotFoundError specific error for not found entities
type EntityNotFoundError struct {
	Type string
	Id   string
}

func (e *EntityNotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Type, e.Id)
}

// IsEntityNotFoundError returns false if the error is not EntityNotFoundError, otherwise true
func IsEntityNotFoundError(err error) (ok bool) {
	_, ok = err.(*EntityNotFoundError)
	return ok
}
