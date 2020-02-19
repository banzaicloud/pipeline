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

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

// IntegratedServiceManager implements the Vault integrated service manager
type IntegratedServicesManager struct {
	integratedservices.PassthroughIntegratedServiceSpecPreparer

	clusterGetter integratedserviceadapter.ClusterGetter
	secretStore   services.SecretStore
	config        Config
	logger        services.Logger
}

// MakeIntegratedServiceManager builds a new integrated service manager component
func MakeIntegratedServiceManager(
	clusterGetter integratedserviceadapter.ClusterGetter,
	secretStore services.SecretStore,
	config Config,
	logger services.Logger,
) IntegratedServicesManager {
	return IntegratedServicesManager{
		clusterGetter: clusterGetter,
		secretStore:   secretStore,
		config:        config,
		logger:        logger,
	}
}

// Name returns the integrated service' name
func (m IntegratedServicesManager) Name() string {
	return integratedServiceName
}

// GetOutput returns the Vault integrated service' output
func (m IntegratedServicesManager) GetOutput(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceOutput, error) {
	boundSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return nil, integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               err.Error(),
		}
	}

	cluster, err := m.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.New("failed to get cluster")
	}
	orgID := cluster.GetOrganizationId()

	// get token from vault
	var token string
	if boundSpec.CustomVault.Enabled && boundSpec.CustomVault.SecretID != "" {
		tokenValues, err := m.secretStore.GetSecretValues(ctx, boundSpec.CustomVault.SecretID)
		if err != nil {
			return nil, errors.WrapIf(err, "failed get token from Vault")
		}
		token = tokenValues[vaultTokenKey]
	}

	// create Vault manager
	vaultManager, err := newVaultManager(boundSpec, orgID, clusterID, token, m.logger)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create Vault manager")
	}

	defer vaultManager.close()

	chartVersion := m.config.Charts.Webhook.Version

	vaultOutput, err := getVaultOutput(*vaultManager, orgID, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get Vault output")
	}

	out := map[string]interface{}{
		"vault": vaultOutput,
		"webhook": map[string]interface{}{
			"version": chartVersion,
		},
	}

	return out, nil
}

func getVaultOutput(m vaultManager, orgID, clusterID uint) (map[string]interface{}, error) {
	// get Vault version
	vaultVersion, err := m.getVaultVersion()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get Vault version")
	}

	out := map[string]interface{}{
		"authMethodPath": getAuthMethodPath(orgID, clusterID),
		"role":           getRoleName(m.customVault),
		"version":        vaultVersion,
	}
	if !m.customVault {
		out["policy"] = getDefaultPolicy(orgID)
	}

	return out, nil
}

// ValidateSpec validates a Vault integrated service specification
func (m IntegratedServicesManager) ValidateSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) error {
	vaultSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return err
	}

	if !m.config.Managed.Enabled && !vaultSpec.CustomVault.Enabled {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               "Pipeline's managed Vault service is not available, configure a custom Vault instance",
		}
	}

	if err := vaultSpec.Validate(); err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               err.Error(),
		}
	}

	return nil
}
