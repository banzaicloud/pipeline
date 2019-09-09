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
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common"
)

// FeatureManager implements the Vault feature manager
type FeatureManager struct {
	clusterGetter clusterfeatureadapter.ClusterGetter

	logger common.Logger
}

// NewVaultFeatureManager builds a new feature manager component
func MakeFeatureManager(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	logger common.Logger,
) FeatureManager {
	return FeatureManager{
		clusterGetter: clusterGetter,
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

	// create Vault client
	vaultManager, err := newVaultManager(boundSpec)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create Vault client")
	}

	// get vault version
	vaultVersion, err := vaultManager.getVaultVersion()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get Vault version")
	}

	_, chartVersion := getChartParams()

	out := map[string]interface{}{
		"vault": map[string]interface{}{
			"version": vaultVersion,
		},
		"wehhook": map[string]interface{}{
			"version": chartVersion,
		},
	}

	return out, nil
}

// ValidateSpec validates a Vault feature specification
func (m FeatureManager) ValidateSpec(ctx context.Context, spec clusterfeature.FeatureSpec) error {
	vaultSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return err
	}

	if vaultSpec.CustomVault.Enabled {

		// address is required in case of custom vault
		if len(vaultSpec.CustomVault.Address) == 0 {
			return errors.New("address field is required in case of custom vault")
		}
	}

	if len(vaultSpec.Settings.Namespaces) == 1 && vaultSpec.Settings.Namespaces[0] == "*" &&
		len(vaultSpec.Settings.ServiceAccounts) == 1 && vaultSpec.Settings.ServiceAccounts[0] == "*" {
		return errors.New("both namespaces and service accounts can not be \"*\"")
	}

	return nil
}

// PrepareSpec makes certain preparations to the spec before it's sent to be applied
func (m FeatureManager) PrepareSpec(ctx context.Context, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureSpec, error) {
	return spec, nil
}
