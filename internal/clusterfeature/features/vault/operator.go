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
	"os"

	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	"github.com/banzaicloud/pipeline/auth"
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
	vaultClient    *vault.Client
}

// MakeFeatureOperator returns a Vault feature operator
func MakeFeatureOperator(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	logger common.Logger,
	vaultClient *vault.Client,
) FeatureOperator {
	return FeatureOperator{
		clusterGetter:  clusterGetter,
		clusterService: clusterService,
		helmService:    helmService,
		logger:         logger,
		vaultClient:    vaultClient,
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
		return err
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
		orgID, err := op.getOrgID(ctx, clusterID)
		if err != nil {
			return errors.New("failed to get organization ID from context")
		}

		// create policy
		if err := op.createPolicy(orgID, clusterID); err != nil {
			return errors.WrapIf(err, "failed to create policy")
		}
		logger.Info("policy created successfully")

		// enable auth method
		if err := op.enableAuth(authMethodPath, authMethodType); err != nil {
			return errors.WrapIf(err, fmt.Sprintf("failed to enabling %s auth method for vault", authMethodType))
		}
		logger.Info(fmt.Sprintf("auth method %q enabled for vault", authMethodType))

		// create role
		if _, err := op.createRole(orgID, clusterID, boundSpec.Settings.ServiceAccounts, boundSpec.Settings.Namespaces); err != nil {
			return errors.WrapIf(err, fmt.Sprintf("failed to create role in the auth method %q", authMethodType))
		}
		logger.Info(fmt.Sprintf("role created in auth method %q for vault", authMethodType))

	}

	return nil
}

func (op *FeatureOperator) enableAuth(path, authType string) error {

	mounts, err := op.vaultClient.RawClient().Sys().ListAuth()
	if err != nil {
		return errors.WrapIf(err, "failed to list auth")
	}

	if _, ok := mounts[fmt.Sprintf("%s/", authMethodPath)]; ok {
		log.Debugf("%s auth path is already in use", authMethodPath)
		return nil
	}

	return op.vaultClient.RawClient().Sys().EnableAuthWithOptions(
		path,
		&api.EnableAuthOptions{
			Type: authType,
		})
}

func (op *FeatureOperator) disableAuth(path string) error {
	return op.vaultClient.RawClient().Sys().DisableAuth(path)
}

func (op *FeatureOperator) createRole(orgID, clusterID uint, serviceAccounts, namespaces []string) (*api.Secret, error) {
	roleData := map[string]interface{}{
		"bound_service_account_names":      serviceAccounts,
		"bound_service_account_namespaces": namespaces,
		"policies":                         []string{getPolicyName(orgID, clusterID)},
	}
	return op.vaultClient.RawClient().Logical().Write(rolePath, roleData)
}

func (op *FeatureOperator) deleteRole() (*api.Secret, error) {
	return op.vaultClient.RawClient().Logical().Delete(rolePath)
}

func (op *FeatureOperator) createPolicy(orgID, clusterID uint) error {
	return op.vaultClient.RawClient().Sys().PutPolicy(
		getPolicyName(orgID, clusterID),
		fmt.Sprintf(`
			path "secret/org/%d/*" {
				capabilities = [ "read", "list" ]
			}`, orgID),
	)
}

func (op *FeatureOperator) deletePolicy(orgID, clusterID uint) error {
	return op.vaultClient.RawClient().Sys().DeletePolicy(getPolicyName(orgID, clusterID))
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
	var vaultAddress string
	if spec.CustomVault.Enabled {
		vaultAddress = spec.CustomVault.Address
	} else {
		vaultAddress = os.Getenv(vaultAddressEnvKey)
	}

	// create chart values
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)
	var chartValues = &webhookValues{
		Env: map[string]string{
			vaultAddressEnvKey: vaultAddress,
		},
		NamespaceSelector: namespaceSelector{
			MatchExpressions: []matchExpressions{
				{
					Key:      "name",
					Operator: "NotIn",
					Values: []string{
						kubeSysNamesapce,
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
func (op FeatureOperator) Deactivate(ctx context.Context, clusterID uint) error {
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

	// delete role
	if _, err := op.deleteRole(); err != nil {
		return errors.WrapIf(err, "failed to delete role")
	}
	logger.Info("role deleted successfully")

	// disable auth method
	if err := op.disableAuth(authMethodPath); err != nil {
		return errors.WrapIf(err, fmt.Sprintf("failed to disabling %s auth method for vault", authMethodType))
	}
	logger.Info(fmt.Sprintf("auth method %q for vault deactivated successfully", authMethodType))

	// get orgID to delete policy rule
	orgID, err := op.getOrgID(ctx, clusterID)
	if err != nil {
		return errors.New("failed to get organization ID from context")
	}

	// delete policy
	if err := op.deletePolicy(orgID, clusterID); err != nil {
		return errors.WrapIf(err, fmt.Sprintf("failed to delete policy"))
	}
	logger.Info("policy deleted successfully")

	return nil
}

func (op *FeatureOperator) getOrgID(ctx context.Context, clusterID uint) (uint, error) {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return 0, errors.WrapIf(err, "failed to get cluster by ID")
	}
	org, err := auth.GetOrganizationById(cl.GetOrganizationId())
	if err != nil {
		return 0, errors.WrapIf(err, "failed to get organization by ID")
	}
	return org.ID, nil
}
