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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
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

	cluster, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.New("failed to get cluster")
	}

	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": featureName})

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	if err := op.configureVault(ctx, logger, cluster.GetOrganizationId(), clusterID, boundSpec); err != nil {
		return errors.WrapIf(err, "failed to configure Vault")
	}

	return nil
}

func (op FeatureOperator) configureVault(
	ctx context.Context,
	logger common.Logger,
	orgID,
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

		// Prepare cluster first with the proper token reviwer SA
		cluster, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return errors.WrapIf(err, "failed to get cluster")
		}

		kubeConfigRaw, err := cluster.GetK8sConfig()
		if err != nil {
			return errors.WrapIf(err, "failed to get cluster Kubernetes config")
		}

		kubeConfig, err := k8sclient.NewClientConfig(kubeConfigRaw)
		if err != nil {
			return errors.WrapIf(err, "failed to create cluster Kubernetes config")
		}

		k8sClient, err := k8sclient.NewClientFromConfig(kubeConfig)
		if err != nil {
			return errors.WrapIf(err, "failed to create Kubernetes client")
		}

		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "vault-token-reviewer",
			},
		}
		_, err = k8sClient.CoreV1().ServiceAccounts("pipeline-system").Create(serviceAccount)
		if err != nil && !k8sapierrors.IsAlreadyExists(err) {
			return errors.WrapIf(err, "failed to create token reviewer ServiceAccount")
		}

		serviceAccount, err = k8sClient.CoreV1().ServiceAccounts("pipeline-system").Get(serviceAccount.Name, metav1.GetOptions{})
		if err != nil {
			return errors.WrapIf(err, "failed to get token reviewer ServiceAccount")
		}

		saTokenSecretRef := serviceAccount.Secrets[0]

		saTokenSecret, err := k8sClient.CoreV1().Secrets("pipeline-system").Get(saTokenSecretRef.Name, metav1.GetOptions{})
		if err != nil {
			return errors.WrapIf(err, "failed to find token reviewer ServiceAccount's Secret")
		}

		tokenReviewerRoleBinding := rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "vault-token-reviewer",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "system:auth-delegator",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccount.Name,
					Namespace: serviceAccount.Namespace,
				},
			},
		}
		_, err = k8sClient.RbacV1().ClusterRoleBindings().Create(&tokenReviewerRoleBinding)
		if err != nil && !k8sapierrors.IsAlreadyExists(err) {
			return errors.WrapIf(err, "failed to create token reviewer cluster role binding")
		}

		tokenReviewerJWT := string(saTokenSecret.Data["token"])

		// create vault client
		vaultManager, err := newVaultManager(boundSpec, orgID, clusterID)
		if err != nil {
			return errors.WrapIf(err, "failed to create Vault client")
		}

		// create policy
		var policy string
		if boundSpec.CustomVault.Enabled {
			policy = boundSpec.CustomVault.Policy
		} else {
			policy = getDefaultPolicy(orgID)
		}
		if err := vaultManager.createPolicy(orgID, clusterID, policy); err != nil {
			return errors.WrapIf(err, "failed to create policy")
		}
		logger.Info("policy created successfully")

		// enable auth method
		if err := vaultManager.enableAuth(getAuthMethodPath(orgID, clusterID), authMethodType); err != nil {
			return errors.WrapIf(err, fmt.Sprintf("failed to enabling %s auth method for vault", authMethodType))
		}
		logger.Info(fmt.Sprintf("auth method %q enabled for vault", authMethodType))

		// config auth method
		if _, err := vaultManager.configureAuth(orgID, clusterID, tokenReviewerJWT, kubeConfig.Host, kubeConfig.CAData); err != nil {
			return errors.WrapIf(err, fmt.Sprintf("failed to configure %s auth method for vault", authMethodType))
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

	cluster, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.New("failed to get cluster")
	}

	orgID := cluster.GetOrganizationId()

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
