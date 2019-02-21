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
	"context"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/pkg/errors"
)

const GenerateCertificatesActivityName = "pke-generate-certificates-activity"

type GenerateCertificatesActivity struct {
	secrets ClusterSecretStore
}

func NewGenerateCertificatesActivity(secrets ClusterSecretStore) *GenerateCertificatesActivity {
	return &GenerateCertificatesActivity{
		secrets: secrets,
	}
}

type ClusterSecretStore interface {
	// EnsureSecretExists creates a secret for an organization if it cannot be found and returns it's ID.
	EnsureSecretExists(ctx context.Context, clusterID uint, secret clustersecret.NewSecret) (string, error)
}

type GenerateCertificatesActivityInput struct {
	ClusterID uint
}

func (a *GenerateCertificatesActivity) Execute(ctx context.Context, input GenerateCertificatesActivityInput) error {
	req := clustersecret.NewSecret{
		Name:   "ca",
		Type:   pkgSecret.PKESecretType,
		Values: map[string]string{}, // Implicitly generate the necessary certificates
		Tags: []string{
			pkgSecret.TagBanzaiReadonly,
			pkgSecret.TagBanzaiHidden,
		},
	}

	_, err := a.secrets.EnsureSecretExists(ctx, input.ClusterID, req)

	return errors.Wrap(err, "failed to generate certificates")
}
