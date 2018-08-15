package objectstore

import (
	"github.com/banzaicloud/pipeline/pkg/objectstore"
)

// IsAlreadyExistsError checks if an error indicates an already existing bucket.
func IsAlreadyExistsError(err error) bool {
	return objectstore.IsAlreadyExistsError(err)
}

// IsNotFoundError checks if an error indicates a missing bucket.
func IsNotFoundError(err error) bool {
	return objectstore.IsNotFoundError(err)
}
