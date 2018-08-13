package cluster

import "errors"

var ErrInvalidClusterInstance = errors.New("invalid cluster instance")

type invalidError struct {
	err error
}

func (e *invalidError) Error() string {
	return e.err.Error()
}

func (invalidError) IsInvalid() bool {
	return true
}
