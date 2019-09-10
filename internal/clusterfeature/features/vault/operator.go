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
	"encoding/json"
	"fmt"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/hashicorp/vault/api"
	"github.com/prometheus/common/log"
	"github.com/spf13/viper"
)

// FeatureOperator implements the Vault feature operator
type FeatureOperator struct {
	clusterGetter  clusterfeatureadapter.ClusterGetter
	clusterService clusterfeature.ClusterService
	helmService    features.HelmService
	logger         common.Logger
}

// MakeFeatureOperator returns a Vault feature operator
func MakeFeatureOperator(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	logger common.Logger,
) FeatureOperator {
	return FeatureOperator{
		clusterGetter:  clusterGetter,
		clusterService: clusterService,
		helmService:    helmService,
		logger:         logger,
	}
}

// Name returns the name of the Vault feature
func (op FeatureOperator) Name() string {
	return featureName
}

// Apply applies the provided specification to the cluster feature
func (op FeatureOperator) Apply(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": featureName})

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	if err := op.configureVault(ctx, logger, clusterID, boundSpec); err != nil {
		return errors.WrapIf(err, "failed to configure Vault")
	}

	return nil
}

func (op FeatureOperator) configureVault(
	ctx context.Context,
	logger common.Logger,
	clusterID uint,
	boundSpec vaultFeatureSpec,
) error {
	// install vault-secrets-webhook
	if err := op.installOrUpdateWebhook(ctx, logger, clusterID, boundSpec); err != nil {
		return errors.WrapIf(err, "failed to deploy feature")
	}

	if !boundSpec.CustomVault.Enabled || (boundSpec.CustomVault.Enabled && len(boundSpec.CustomVault.Token) != 0) {
		// custom Vault with token or CP's vault
		logger.Debug("start to setup Vault")

		// get orgID to create policy rule
		orgID, err := getOrgID(ctx, op.clusterGetter, clusterID)
		if err != nil {
			return errors.New("failed to get organization ID from context")
		}

		// create vault client
		vaultManager, err := newVaultManager(boundSpec, orgID, clusterID)
		if err != nil {
			return errors.WrapIf(err, "failed to create Vault client")
		}

		// create policy
		if err := vaultManager.createPolicy(orgID, clusterID); err != nil {
			return errors.WrapIf(err, "failed to create policy")
		}
		logger.Info("policy created successfully")

		// enable auth method
		if err := vaultManager.enableAuth(getAuthMethodPath(orgID, clusterID), authMethodType); err != nil {
			return errors.WrapIf(err, fmt.Sprintf("failed to enabling %s auth method for vault", authMethodType))
		}
		logger.Info(fmt.Sprintf("auth method %q enabled for vault", authMethodType))

		// create role
		if _, err := vaultManager.createRole(orgID, clusterID, boundSpec.Settings.ServiceAccounts, boundSpec.Settings.Namespaces); err != nil {
			return errors.WrapIf(err, fmt.Sprintf("failed to create role in the auth method %q", authMethodType))
		}
		logger.Info(fmt.Sprintf("role created in auth method %q for vault", authMethodType))

	}

	return nil
}

func (m *vaultManager) enableAuth(path, authType string) error {

	mounts, err := m.vaultClient.RawClient().Sys().ListAuth()
	if err != nil {
		return errors.WrapIf(err, "failed to list auth")
	}

	if _, ok := mounts[fmt.Sprintf("%s/", path)]; ok {
		log.Debugf("%s auth path is already in use", path)
		return nil
	}

	return m.vaultClient.RawClient().Sys().EnableAuthWithOptions(
		path,
		&api.EnableAuthOptions{
			Type: authType,
		})
}

func getPolicyName(orgID, clusterID uint) string {
	return fmt.Sprintf("%s_%d_%d", policyNamePrefix, orgID, clusterID)
}

func (op FeatureOperator) installOrUpdateWebhook(
	ctx context.Context,
	logger common.Logger,
	clusterID uint,
	spec vaultFeatureSpec,
) error {
	// create chart values
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)
	var chartValues = &webhookValues{
		NamespaceSelector: namespaceSelector{
			MatchExpressions: []matchExpressions{
				{
					Key:      "name",
					Operator: "NotIn",
					Values: []string{
						kubeSysNamespace,
						pipelineSystemNamespace,
					},
				},
			},
		},
	}
	valuesBytes, err := json.Marshal(chartValues)
	if err != nil {
		logger.Debug("failed to marshal chartValues")
		return errors.WrapIf(err, "failed to decode chartValues")
	}

	chartName, chartVersion := getChartParams()

	return op.helmService.ApplyDeployment(
		ctx,
		clusterID,
		pipelineSystemNamespace,
		chartName,
		vaultWebhookReleaseName,
		valuesBytes,
		chartVersion,
	)
}

func getChartParams() (name string, version string) {
	name = viper.GetString(config.VaultWebhookChartKey)
	version = viper.GetString(config.VaultWebhookChartVersionKey)
	return
}

// Deactivate deactivates the cluster feature
func (op FeatureOperator) Deactivate(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": featureName})

	// delete deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, vaultWebhookReleaseName); err != nil {
		logger.Info("failed to delete feature deployment")

		return errors.WrapIf(err, "failed to uninstall feature")
	}

	logger.Info("vault webhook deployment deleted successfully")

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	orgID, err := getOrgID(ctx, op.clusterGetter, clusterID)
	if err != nil {
		return errors.New("failed to get organization ID from context")
	}

	// create Vault client
	vaultManager, err := newVaultManager(boundSpec, orgID, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to create Vault client")
	}

	// delete role
	if _, err := vaultManager.deleteRole(orgID, clusterID); err != nil {
		return errors.WrapIf(err, "failed to delete role")
	}
	logger.Info("role deleted successfully")

	// disable auth method
	if err := vaultManager.disableAuth(getAuthMethodPath(orgID, clusterID)); err != nil {
		return errors.WrapIf(err, fmt.Sprintf("failed to disabling %s auth method for vault", authMethodType))
	}
	logger.Info(fmt.Sprintf("auth method %q for vault deactivated successfully", authMethodType))

	// delete policy
	if err := vaultManager.deletePolicy(orgID, clusterID); err != nil {
		return errors.WrapIf(err, fmt.Sprintf("failed to delete policy"))
	}
	logger.Info("policy deleted successfully")

	return nil
}
