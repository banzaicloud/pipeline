// Copyright © 2020 Banzai Cloud
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

package auth

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"emperror.dev/emperror"
	"emperror.dev/errors"
)

func TLSConfigForClientAuth(caCertFile string) (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to read CA certificate for client authentication")
	}

	clientCertCAs := x509.NewCertPool()
	if !clientCertCAs.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to append CA certificate")
	}

	config := tls.Config{
		ClientAuth: tls.VerifyClientCertIfGiven,
		ClientCAs:  clientCertCAs,
	}

	return &config, nil
}
