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

	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

const GenerateCertificatesActivityName = "pke-generate-certificates-activity"

type GenerateCertificatesActivity struct {
	clusters Clusters
}

func NewGenerateCertificatesActivity(clusters Clusters) *GenerateCertificatesActivity {
	return &GenerateCertificatesActivity{
		clusters: clusters,
	}
}

type GenerateCertificatesActivityInput struct {
	ClusterID uint
}

func (a *GenerateCertificatesActivity) Execute(ctx context.Context, input GenerateCertificatesActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	// Generate certificates
	req := &secret.CreateSecretRequest{
		Name:   fmt.Sprintf("cluster-%d-ca", c.GetID()),
		Type:   pkgSecret.PKESecretType,
		Values: map[string]string{},
		Tags: []string{
			fmt.Sprintf("clusterUID:%s", c.GetUID()),
			fmt.Sprintf("clusterID:%d", c.GetID()),
			pkgSecret.TagBanzaiReadonly,
			pkgSecret.TagBanzaiHidden,
		},
	}
	_, err = secret.Store.GetOrCreate(c.GetOrganizationId(), req)
	if err != nil {
		return err
	}

	return nil
}
