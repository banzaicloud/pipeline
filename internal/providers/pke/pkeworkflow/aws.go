// Copyright © 2019 Banzai Cloud
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

package pkeworkflow

import (
	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	"github.com/banzaicloud/pipeline/src/secret/verify"
)

type AWSActivityInput struct {
	OrganizationID uint
	SecretID       string
	Region         string
}

// AWSClientFactory creates a new AWS client.
type AWSClientFactory struct {
	secrets SecretStore
}

// NewAWSClientFactory returns a new AWS client factory.
func NewAWSClientFactory(secrets SecretStore) *AWSClientFactory {
	return &AWSClientFactory{secrets: secrets}
}

// SecretStore accesses secrets.
type SecretStore interface {
	// GetSecret returns a secret from an organization.
	GetSecret(organizationID uint, secretID string) (Secret, error)
}

// Secret represents an item in the secret store.
type Secret interface {
	// GetValues returns the secret values.
	GetValues() map[string]string

	// ValidateSecretType checks that the secret is of a certain type.
	ValidateSecretType(t string) error
}

// New creates a new AWS client.
func (f *AWSClientFactory) New(organizationID uint, secretID string, region string) (*session.Session, error) {
	s, err := f.secrets.GetSecret(organizationID, secretID)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get AWS secret")
	}

	err = s.ValidateSecretType(amazon.Provider)
	if err != nil {
		return nil, err
	}

	awsCred := verify.CreateAWSCredentials(s.GetValues())

	return session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: awsCred,
	})
}
