package secret

import "github.com/goph/emperror"

// errorWithSecretContext appends the the secret context to the error.
func errorWithSecretContext(err error, organizationID uint, secretID string) error {
	return emperror.With(
		err,
		"organization-id", organizationID,
		"secret-id", secretID,
	)
}
