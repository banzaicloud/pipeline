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
	"fmt"

	"github.com/banzaicloud/pipeline/pkg/auth"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

const GenerateCertificatesActivityName = "pke-generate-certificates-activity"

type GenerateCertificatesActivity struct {
	secrets SecretStore
}

func NewGenerateCertificatesActivity(secrets SecretStore) *GenerateCertificatesActivity {
	return &GenerateCertificatesActivity{
		secrets: secrets,
	}
}

type SecretStore interface {
	GetOrCreate(organizationID auth.OrganizationID, value *secret.CreateSecretRequest) (pkgSecret.SecretID, error)
}

type GenerateCertificatesActivityInput struct {
	OrganizationID uint
	ClusterID      uint
	ClusterUID     string
}

func (a *GenerateCertificatesActivity) Execute(ctx context.Context, input GenerateCertificatesActivityInput) error {
	req := &secret.CreateSecretRequest{
		Name:   fmt.Sprintf("cluster-%d-ca", input.ClusterID),
		Type:   pkgSecret.PKESecretType,
		Values: map[string]string{}, // Implicitly generate the necessary certificates
		Tags: []string{
			fmt.Sprintf("clusterUID:%s", input.ClusterUID),
			fmt.Sprintf("clusterID:%d", input.ClusterID),
			pkgSecret.TagBanzaiReadonly,
			pkgSecret.TagBanzaiHidden,
		},
	}

	_, err := a.secrets.GetOrCreate(auth.OrganizationID(input.OrganizationID), req)
	if err != nil {
		return err
	}

	return nil
}
