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
	"io/ioutil"

	"github.com/pkg/errors"
)

// FileCALoader loads a CA bundle from file.
type FileCALoader struct {
	certPath string
	keyPath  string
}

// NewFileCALoader returns a new FileCALoader.
func NewFileCALoader(certPath string, keyPath string) *FileCALoader {
	return &FileCALoader{
		certPath: certPath,
		keyPath:  keyPath,
	}
}

func (s *FileCALoader) Load() (*x509.Certificate, crypto.Signer, error) {
	certBytes, err := ioutil.ReadFile(s.certPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read cert file")
	}

	keyBytes, err := ioutil.ReadFile(s.keyPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read key file")
	}

	return parseCABundle(certBytes, keyBytes)
}
