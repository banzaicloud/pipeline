package providers

import (
	"github.com/banzaicloud/pipeline/secret"
	"github.com/pkg/errors"
)

type secretStore interface {
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
}

type secretValidator struct {
	secrets secretStore
}

// NewSecretValidator returns a struct which validates that a secret belongs to a cloud provider.
func NewSecretValidator(secrets secretStore) *secretValidator {
	return &secretValidator{secrets}
}

// ValidateSecretType validates that a secret belongs to a cloud provider.
func (v *secretValidator) ValidateSecretType(organizationID uint, secretID string, provider string) error {
	s, err := v.secrets.Get(organizationID, secretID)
	if err == secret.ErrSecretNotExists {
		return errors.Wrap(err, "error during secret validation")
	} else if err != nil {
		return errors.WithMessage(err, "error during secret validation")
	}

	return s.ValidateSecretType(provider)
}
