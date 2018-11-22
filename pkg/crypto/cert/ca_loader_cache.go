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

package cert

import (
	"crypto"
	"crypto/x509"
	"sync"

	"github.com/pkg/errors"
)

// CACache caches certificate and signing key from a source and caches it.
type CACache struct {
	loader CALoader

	cert *x509.Certificate
	key  crypto.Signer

	mu sync.Mutex
}

// NewCACache returns a new CACache instance.
func NewCACache(loader CALoader) *CACache {
	return &CACache{
		loader: loader,
	}
}

func (s *CACache) Load() (*x509.Certificate, crypto.Signer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cert == nil || s.key == nil {
		cert, key, err := s.loader.Load()
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed to load CA bundle")
		}

		s.cert, s.key = cert, key
	}

	return s.cert, s.key, nil
}
