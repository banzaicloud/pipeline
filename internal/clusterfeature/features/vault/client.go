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
	"fmt"
	"os"

	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	vaultapi "github.com/hashicorp/vault/api"
)

type vaultManager struct {
	vaultClient *vault.Client
}

func newVaultManager(spec vaultFeatureSpec, orgID, clusterID uint) (*vaultManager, error) {
	var vaultAddress string
	if spec.CustomVault.Enabled {
		vaultAddress = spec.CustomVault.Address
	} else {
		vaultAddress = os.Getenv(vaultAddressEnvKey)
	}

	clientConfig := vaultapi.DefaultConfig()
	clientConfig.Address = vaultAddress

	var clientOptions = []vault.ClientOption{
		vault.ClientRole(roleName),
		vault.ClientAuthPath(getAuthMethodPath(orgID, clusterID)),
	}
	if len(spec.CustomVault.Token) != 0 {
		clientOptions = append(clientOptions, vault.ClientToken(spec.CustomVault.Token))
	}

	client, err := vault.NewClientFromConfig(
		clientConfig,
		clientOptions...,
	)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create Vault client")
	}

	return &vaultManager{
		vaultClient: client,
	}, nil
}

func (m vaultManager) getVaultVersion() (string, error) {
	status, err := m.vaultClient.RawClient().Sys().SealStatus()
	if err != nil {
		return "", errors.WrapIf(err, "failed to get Vault status")
	}

	return status.Version, nil
}

func (m *vaultManager) disableAuth(path string) error {
	return m.vaultClient.RawClient().Sys().DisableAuth(path)
}

func (m *vaultManager) createRole(orgID, clusterID uint, serviceAccounts, namespaces []string) (*vaultapi.Secret, error) {
	roleData := map[string]interface{}{
		"bound_service_account_names":      serviceAccounts,
		"bound_service_account_namespaces": namespaces,
		"policies":                         []string{getPolicyName(orgID, clusterID)},
	}
	return m.vaultClient.RawClient().Logical().Write(getRolePath(orgID, clusterID), roleData)
}

func (m *vaultManager) deleteRole(orgID, clusterID uint) (*vaultapi.Secret, error) {
	return m.vaultClient.RawClient().Logical().Delete(getRolePath(orgID, clusterID))
}

func (m *vaultManager) createPolicy(orgID, clusterID uint, policy string) error {
	return m.vaultClient.RawClient().Sys().PutPolicy(getPolicyName(orgID, clusterID), policy)
}

func (m *vaultManager) deletePolicy(orgID, clusterID uint) error {
	return m.vaultClient.RawClient().Sys().DeletePolicy(getPolicyName(orgID, clusterID))
}

func getAuthMethodPath(orgID, clusterID uint) string {
	return fmt.Sprintf("%s/%d/%d", authMethodPathPrefix, orgID, clusterID)
}

func getRolePath(orgID, clusterID uint) string {
	return fmt.Sprintf("auth/%s/role/%s", getAuthMethodPath(orgID, clusterID), roleName)
}

func getDefaultPolicy(orgID uint) string {
	return fmt.Sprintf(`
			path "secret/org/%d/*" {
				capabilities = [ "read", "list" ]
			}`, orgID)
}
