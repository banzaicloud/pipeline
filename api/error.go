package api

// isNotFound checks whether an error is about a resource not being found.
func isNotFound(err error) bool {
	if e, ok := err.(interface {
		NotFound() bool
	}); ok {
		return e.NotFound()
	}

	return false
}
