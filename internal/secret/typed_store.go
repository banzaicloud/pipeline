package secret

import (
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
)

// typedStore validates that a secret item is of a specific type.
type typedStore struct {
	store store
	t     string
}

// NewTypedStore returns a secret store that checks the secret type.
func NewTypedStore(store store, t string) *typedStore {
	return &typedStore{
		store: store,
		t:     t,
	}
}

// Get returns the requested secret of the organization.
func (s *typedStore) Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error) {
	secretItem, err := s.store.Get(organizationID, secretID)
	if err != nil {
		return nil, errorWithSecretContext(emperror.With(err, "type", s.t), organizationID, secretID)
	}

	err = secretItem.ValidateSecretType(s.t)
	if err != nil {
		return nil, errorWithSecretContext(
			emperror.With(
				emperror.Wrap(err, "invalid secretItem type"),
				"type", s.t,
			),
			organizationID,
			secretID,
		)
	}

	return secretItem, nil
}
