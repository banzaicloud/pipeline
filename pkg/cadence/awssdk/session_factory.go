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

package awssdk

import (
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

// SecretStore provides access to the Pipeline secret store.
type SecretStore interface {
	GetSecretValues(ctx context.Context, secretID string) (map[string]string, error)
}

// SessionFactory creates an AWS session.
type SessionFactory struct {
	secretStore SecretStore
}

// NewSessionFactory creates a new SessionFactory.
func NewSessionFactory(secretStore SecretStore) SessionFactory {
	return SessionFactory{
		secretStore: secretStore,
	}
}

func (s SessionFactory) Session(ctx context.Context) (*session.Session, error) {
	secretID, ok := SecretID(ctx)
	if !ok {
		return nil, errors.New("unable to extract secret ID from context")
	}

	secret, err := s.secretStore.GetSecretValues(ctx, secretID)
	if err != nil {
		return nil, err
	}

	region, ok := Region(ctx)
	if !ok {
		return nil, errors.New("unable to extract region from context")
	}

	return session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			secret["AWS_ACCESS_KEY_ID"],
			secret["AWS_SECRET_ACCESS_KEY"],
			"",
		),
	})
}
