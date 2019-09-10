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

package vault

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
)

const (
	featureName             = "vault"
	vaultWebhookReleaseName = "vault-secrets-webhook"
	kubeSysNamespace        = "kube-system"
	vaultAddressEnvKey      = "VAULT_ADDR"
	roleName                = "pipeline-webhook"
	authMethodType          = "kubernetes"
	authMethodPathPrefix    = "kubernetes"
	policyNamePrefix        = "allow_cluster_secrets"
)

func getOrgID(ctx context.Context, clusterGetter clusterfeatureadapter.ClusterGetter, clusterID uint) (uint, error) {
	cl, err := clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return 0, errors.WrapIf(err, "failed to get cluster by ID")
	}
	org, err := auth.GetOrganizationById(cl.GetOrganizationId())
	if err != nil {
		return 0, errors.WrapIf(err, "failed to get organization by ID")
	}
	return org.ID, nil
}
