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
	"fmt"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/mitchellh/mapstructure"
)

// FeatureManager implements the Vault feature manager
type FeatureManager struct {
	clusterGetter clusterfeatureadapter.ClusterGetter
	secretStore   features.SecretStore
	logger        common.Logger
}

// NewVaultFeatureManager builds a new feature manager component
func MakeFeatureManager(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	secretStore features.SecretStore,
	logger common.Logger,
) FeatureManager {
	return FeatureManager{
		clusterGetter: clusterGetter,
		secretStore:   secretStore,
		logger:        logger,
	}
}

// Name returns the feature's name
func (m FeatureManager) Name() string {
	return featureName
}

// GetOutput returns the Vault feature's output
func (m FeatureManager) GetOutput(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureOutput, error) {
	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return nil, clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	cluster, err := m.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.New("failed to get cluster")
	}
	orgID := cluster.GetOrganizationId()

	// get token from vault
	tokenValues, err := m.secretStore.GetSecretValues(ctx, boundSpec.CustomVault.TokenSecretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed get token from Vault")
	}

	token := tokenValues[vaultTokenKey]

	// create Vault client
	vaultManager, err := newVaultManager(boundSpec, orgID, clusterID, token)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create Vault client")
	}

	_, chartVersion := getChartParams()

	vaultOutput, err := getVaultOutput(vaultManager, orgID, clusterID, boundSpec.CustomVault.Enabled)
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

func getVaultOutput(m *vaultManager, orgID, clusterID uint, isCustomVault bool) (map[string]interface{}, error) {
	// get Vault version
	vaultVersion, err := m.getVaultVersion()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get Vault version")
	}

	out := map[string]interface{}{
		"authMethodPath": getAuthMethodPath(orgID, clusterID),
		"rolePath":       getRolePath(orgID, clusterID),
		"version":        vaultVersion,
	}
	if !isCustomVault {
		out["policy"] = getDefaultPolicy(orgID)
	}

	return out, nil
}

// ValidateSpec validates a Vault feature specification
func (m FeatureManager) ValidateSpec(ctx context.Context, spec clusterfeature.FeatureSpec) error {
	vaultSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return err
	}

	if err := vaultSpec.Validate(); err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	return nil
}

func (m FeatureManager) BeforeSave(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureSpec, error) {
	vaultSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return nil, err
	}

	if vaultSpec.CustomVault.Enabled && len(vaultSpec.CustomVault.Token) != 0 {
		createSecretRequest := &secret.CreateSecretRequest{
			Name: fmt.Sprintf("vault-token-%d-cluster", clusterID),
			Type: pkgSecret.GenericSecret,
			Values: map[string]string{
				vaultTokenKey: vaultSpec.CustomVault.Token,
			},
			Tags: []string{
				pkgSecret.TagBanzaiReadonly,
				featureSecretTag,
			},
		}
		secretID, err := m.secretStore.Store(ctx, createSecretRequest)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to store token in Vault")
		}

		vaultSpec.CustomVault.Token = ""
		vaultSpec.CustomVault.TokenSecretID = secretID

		var res clusterfeature.FeatureSpec
		if err := mapstructure.Decode(vaultSpec, &res); err != nil {
			return nil, errors.WrapIf(err, "failed to decode feature spec")
		}
		return res, nil
	}
	return spec, nil
}

// PrepareSpec makes certain preparations to the spec before it's sent to be applied
func (m FeatureManager) PrepareSpec(ctx context.Context, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureSpec, error) {
	return spec, nil
}
