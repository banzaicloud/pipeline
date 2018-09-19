// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
