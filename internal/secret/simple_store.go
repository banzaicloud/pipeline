package secret

// simpleStore implements a simplified interface for the secret store.
type simpleStore struct {
	store store
}

// NewSimpleStore returns a secret store with a simplified interface.
func NewSimpleStore(store store) *simpleStore {
	return &simpleStore{
		store: store,
	}
}

// Get returns the requested secret of the organization.
func (s *simpleStore) Get(organizationID uint, secretID string) (map[string]string, error) {
	secret, err := s.store.Get(organizationID, secretID)
	if err != nil {
		return nil, errorWithSecretContext(err, organizationID, secretID)
	}

	return secret.Values, nil
}
