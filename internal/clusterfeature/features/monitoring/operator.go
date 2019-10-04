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

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"

	"emperror.dev/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/auth"
	pkgCluster "github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

// FeatureOperator implements the Monitoring feature operator
type FeatureOperator struct {
	clusterGetter  clusterfeatureadapter.ClusterGetter
	clusterService clusterfeature.ClusterService
	helmService    features.HelmService
	config         Configuration
	logger         common.Logger
	secretStore    features.SecretStore
}

// MakeFeatureOperator returns a Monitoring feature operator
func MakeFeatureOperator(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	config Configuration,
	logger common.Logger,
	secretStore features.SecretStore,
) FeatureOperator {
	return FeatureOperator{
		clusterGetter:  clusterGetter,
		clusterService: clusterService,
		helmService:    helmService,
		config:         config,
		logger:         logger,
		secretStore:    secretStore,
	}
}

// Name returns the name of the DNS feature
func (FeatureOperator) Name() string {
	return featureName
}

// Apply applies the provided specification to the cluster feature
func (op FeatureOperator) Apply(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
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

	cluster, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster")
	}

	var grafanaSecretID string
	if boundSpec.Grafana.Enabled {
		// get Grafana secret from spec or generate
		grafanaSecretID, err = op.getGrafanaSecret(ctx, cluster, boundSpec, logger)
		if err != nil {
			return errors.WrapIf(err, "failed to get Grafana secret")
		}
	}

	if boundSpec.Prometheus.Enabled {
		// get Prometheus secret from spec or generate
		prometheusSecretName, err := op.getPrometheusSecret(ctx, cluster, boundSpec, logger)
		if err != nil {
			return errors.WrapIf(err, "failed to get Prometheus secret")
		}

		// install Prometheus secret
		if err := op.installPrometheusSecret(ctx, clusterID, prometheusSecretName); err != nil {
			return errors.WrapIfWithDetails(err, "failed to install Prometheus secret to cluster", "clusterID", clusterID)
		}
	}

	// install Prometheus Pushgateway
	if err := op.installPrometheusPushGateway(ctx, cluster, logger); err != nil {
		return errors.WrapIf(err, "failed to install Prometheus Pushgateway")
	}

	// install Prometheus Operator
	if err := op.installPrometheusOperator(ctx, cluster, logger, boundSpec, grafanaSecretID); err != nil {
		return errors.WrapIf(err, "failed to install Prometheus operator")
	}

	return nil
}

// Deactivate deactivates the cluster feature
func (op FeatureOperator) Deactivate(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	if boundSpec.Grafana.SecretId == "" {
		// Grafana secret generated in activation flow, delete it
		if err := op.deleteGrafanaSecret(ctx, clusterID); err != nil && !isSecretNotFoundError(err) {
			return errors.WrapIf(err, "failed to delete Grafana secret")
		}
	}

	if boundSpec.Prometheus.SecretId == "" {
		// Prometheus secret generated in activation flow, delete it
		if err := op.deletePrometheusSecret(ctx, clusterID); err != nil && !isSecretNotFoundError(err) {
			return errors.WrapIf(err, "failed to delete Prometheus secret")
		}
	}

	// delete prometheus operator deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, prometheusOperatorReleaseName); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", prometheusOperatorReleaseName)
	}

	// delete prometheus pushgateway deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, prometheusPushgatewayReleaseName); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", prometheusPushgatewayReleaseName)
	}

	return nil
}

func (op FeatureOperator) installPrometheusSecret(ctx context.Context, clusterID uint, prometheusSecretName string) error {
	pipelineSystemNamespace := op.config.pipelineSystemNamespace

	installPromSecretRequest := pkgCluster.InstallSecretRequest{
		SourceSecretName: prometheusSecretName,
		Namespace:        pipelineSystemNamespace,
		Spec: map[string]pkgCluster.InstallSecretRequestSpecItem{
			"auth": {Source: pkgSecret.HtpasswdFile},
		},
		Update: true,
	}

	if _, err := op.installSecret(ctx, clusterID, prometheusSecretName, installPromSecretRequest); err != nil {
		return errors.WrapIfWithDetails(err, "failed to install Prometheus secret to cluster", "clusterID", clusterID)
	}

	return nil
}

func (op FeatureOperator) installPrometheusPushGateway(
	ctx context.Context,
	cluster clusterfeatureadapter.Cluster,
	logger common.Logger,
) error {
	headNodeAffinity := GetHeadNodeAffinity(cluster, op.config)
	tolerations := GetHeadNodeTolerations(op.config)

	pipelineSystemNamespace := op.config.pipelineSystemNamespace
	var chartValues = &prometheusPushgatewayValues{
		affinityValues:   affinityValues{Affinity: headNodeAffinity},
		tolerationValues: tolerationValues{Tolerations: tolerations},
	}

	valuesBytes, err := json.Marshal(chartValues)
	if err != nil {
		logger.Debug("failed to marshal chartValues")
		return errors.WrapIf(err, "failed to decode chartValues")
	}

	chartName := op.config.pushgateway.chartName
	chartVersion := op.config.pushgateway.chartVersion

	return op.helmService.ApplyDeployment(
		ctx,
		cluster.GetID(),
		pipelineSystemNamespace,
		chartName,
		prometheusPushgatewayReleaseName,
		valuesBytes,
		chartVersion,
	)
}

func (op FeatureOperator) installPrometheusOperator(
	ctx context.Context,
	cluster clusterfeatureadapter.Cluster,
	logger common.Logger,
	spec featureSpec,
	grafanaSecretID string,
) error {
	var grafanaUser string
	var grafanaPass string
	if spec.Grafana.Enabled {
		grafanaSecret, err := op.secretStore.GetSecretValues(ctx, grafanaSecretID)
		if err != nil {
			return errors.WrapIf(err, "failed to get Grafana secret")
		}
		grafanaUser = grafanaSecret[pkgSecret.Username]
		grafanaPass = grafanaSecret[pkgSecret.Password]
	}

	headNodeAffinity := GetHeadNodeAffinity(cluster, op.config)
	tolerations := GetHeadNodeTolerations(op.config)

	// create chart values
	pipelineSystemNamespace := op.config.pipelineSystemNamespace
	var chartValues = &prometheusOperatorValues{
		Grafana: grafanaValues{
			baseValues: baseValues{
				Enabled: spec.Grafana.Enabled,
				Ingress: ingressValues{
					Enabled: spec.Grafana.Public.Enabled,
					Hosts:   []string{spec.Grafana.Public.Domain},
					Path:    spec.Grafana.Public.Path,
				},
			},
			affinityValues:   affinityValues{Affinity: headNodeAffinity},
			tolerationValues: tolerationValues{Tolerations: tolerations},
			AdminUser:        grafanaUser,
			AdminPassword:    grafanaPass,
			GrafanaIni: grafanaIniValues{Server: grafanaIniServerValues{
				RootUrl:          fmt.Sprintf("http://0.0.0.0:3000%s/", spec.Grafana.Public.Path),
				ServeFromSubPath: true,
			}},
		},
		Alertmanager: alertmanagerValues{
			baseValues: baseValues{
				Enabled: spec.Alertmanager.Enabled,
				Ingress: ingressValues{
					Enabled: spec.Alertmanager.Public.Enabled,
					Hosts:   []string{spec.Alertmanager.Public.Domain},
					Paths:   []string{spec.Alertmanager.Public.Path},
				},
			},
			Spec: SpecValues{
				affinityValues:   affinityValues{Affinity: headNodeAffinity},
				tolerationValues: tolerationValues{Tolerations: tolerations},
				RoutePrefix:      spec.Alertmanager.Public.Path,
			},
			Config: op.generateAlertManagerProvidersConfig(spec.Alertmanager.Provider),
		},
		Prometheus: prometheusValues{
			baseValues: baseValues{
				Enabled: spec.Prometheus.Enabled,
				Ingress: ingressValues{
					Enabled: spec.Prometheus.Public.Enabled,
					Hosts:   []string{spec.Prometheus.Public.Domain},
					Paths:   []string{spec.Prometheus.Public.Path},
				},
			},
			Spec: SpecValues{
				affinityValues:   affinityValues{Affinity: headNodeAffinity},
				tolerationValues: tolerationValues{Tolerations: tolerations},
				RoutePrefix:      spec.Prometheus.Public.Path,
			},
			Annotations: map[string]interface{}{
				"traefik.ingress.kubernetes.io/auth-type":   "basic",
				"traefik.ingress.kubernetes.io/auth-secret": kubePrometheusSecretName,
			},
		},
		KubeStateMetrics: kubeStateMetricsValues{
			Enabled: true,
			SpecValues: SpecValues{
				affinityValues:   affinityValues{Affinity: headNodeAffinity},
				tolerationValues: tolerationValues{Tolerations: tolerations},
			},
		},
		NodeExporter: nodeExporterValues{
			Enabled: true,
		},
	}

	valuesBytes, err := json.Marshal(chartValues)
	if err != nil {
		logger.Debug("failed to marshal chartValues")
		return errors.WrapIf(err, "failed to decode chartValues")
	}

	chartName := op.config.operator.chartName
	chartVersion := op.config.operator.chartVersion

	return op.helmService.ApplyDeployment(
		ctx,
		cluster.GetID(),
		pipelineSystemNamespace,
		chartName,
		prometheusOperatorReleaseName,
		valuesBytes,
		chartVersion,
	)
}

func (op FeatureOperator) generateGrafanaSecret(
	ctx context.Context,
	cluster clusterfeatureadapter.Cluster,
	logger common.Logger,
) (string, error) {

	clusterNameSecretTag := getClusterNameSecretTag(cluster.GetName())
	clusterUidSecretTag := getClusterUIDSecretTag(cluster.GetUID())
	releaseSecretTag := getReleaseSecretTag()

	// Generating Grafana credentials
	username := op.config.grafanaAdminUsername
	password, err := secret.RandomString("randAlphaNum", 12)
	if err != nil {
		return "", errors.WrapIf(err, "failed to generate Grafana admin user password")
	}

	grafanaSecretRequest := secret.CreateSecretRequest{
		Name: getGrafanaSecretName(cluster.GetID()),
		Type: pkgSecret.PasswordSecretType,
		Values: map[string]string{
			pkgSecret.Username: username,
			pkgSecret.Password: password,
		},
		Tags: []string{
			clusterNameSecretTag,
			clusterUidSecretTag,
			pkgSecret.TagBanzaiReadonly,
			releaseSecretTag,
			grafanaSecretTag,
		},
	}
	grafanaSecretID, err := secret.Store.CreateOrUpdate(cluster.GetOrganizationId(), &grafanaSecretRequest)
	if err != nil {
		return "", errors.WrapIf(err, "error store prometheus secret")
	}
	logger.Debug("grafana secret stored")

	return grafanaSecretID, nil
}

func (op FeatureOperator) deleteGrafanaSecret(ctx context.Context, clusterID uint) error {
	secretID, err := op.secretStore.GetIDByName(ctx, getGrafanaSecretName(clusterID))
	if err != nil {
		return errors.WrapIf(err, "failed to get Grafana secret")
	}
	return op.secretStore.Delete(ctx, secretID)
}

func (op FeatureOperator) deletePrometheusSecret(ctx context.Context, clusterID uint) error {
	secretID, err := op.secretStore.GetIDByName(ctx, getPrometheusSecretName(clusterID))
	if err != nil {
		return errors.WrapIf(err, "failed to get Prometheus secret")
	}
	return op.secretStore.Delete(ctx, secretID)
}

func (op FeatureOperator) generatePrometheusSecret(ctx context.Context, cluster clusterfeatureadapter.Cluster) (string, error) {

	clusterNameSecretTag := getClusterNameSecretTag(cluster.GetName())
	clusterUidSecretTag := getClusterUIDSecretTag(cluster.GetUID())
	releaseSecretTag := getReleaseSecretTag()
	prometheusSecretName := getPrometheusSecretName(cluster.GetID())

	prometheusAdminPass, err := secret.RandomString("randAlphaNum", 12)
	if err != nil {
		return "", errors.WrapIf(err, "Prometheus password generation failed")
	}

	prometheusSecretRequest := &secret.CreateSecretRequest{
		Name: prometheusSecretName,
		Type: pkgSecret.HtpasswdSecretType,
		Values: map[string]string{
			pkgSecret.Username: prometheusSecretUserName,
			pkgSecret.Password: prometheusAdminPass,
		},
		Tags: []string{
			clusterNameSecretTag,
			clusterUidSecretTag,
			pkgSecret.TagBanzaiReadonly,
			releaseSecretTag,
		},
	}
	_, err = secret.Store.CreateOrUpdate(cluster.GetOrganizationId(), prometheusSecretRequest)
	if err != nil {
		return "", errors.WrapIf(err, "failed to store Prometheus secret")
	}

	return prometheusSecretName, nil
}

func (op FeatureOperator) installSecret(ctx context.Context, clusterID uint, secretName string, secretRequest pkgCluster.InstallSecretRequest) (*pkgSecret.K8SSourceMeta, error) {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get cluster", "clusterID", clusterID)
	}

	k8sSec, err := pkgCluster.InstallSecret(cl, secretName, secretRequest)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to install secret to the cluster", "clusterID", clusterID)
	}

	return k8sSec, nil
}

func (op FeatureOperator) getGrafanaSecret(
	ctx context.Context,
	cluster clusterfeatureadapter.Cluster,
	spec featureSpec,
	logger common.Logger,
) (string, error) {
	var secretID = spec.Grafana.SecretId
	if secretID == "" {
		// check Grafana secret exists
		existingSecretID, err := op.secretStore.GetIDByName(ctx, getGrafanaSecretName(cluster.GetID()))
		if existingSecretID != "" {
			logger.Debug("Grafana secret already exists")
			return existingSecretID, nil
		} else if isSecretNotFoundError(err) {
			// generate and store Grafana secret
			secretID, err = op.generateGrafanaSecret(ctx, cluster, logger)
			if err != nil {
				return "", errors.WrapIf(err, "failed to create Grafana secret")
			}
		} else {
			return "", errors.WrapIf(err, "error during getting Grafana secret")
		}
	}

	return secretID, nil
}

func (op FeatureOperator) getPrometheusSecret(
	ctx context.Context,
	cluster clusterfeatureadapter.Cluster,
	spec featureSpec,
	logger common.Logger,
) (string, error) {
	var secretName string
	if spec.Prometheus.SecretId == "" {
		// generate Prometheus secret
		var prometheusSecretName = getPrometheusSecretName(cluster.GetID())
		existingSecretID, err := op.secretStore.GetIDByName(ctx, prometheusSecretName)
		if existingSecretID != "" {
			logger.Debug("Prometheus secret already exists")
			return prometheusSecretName, nil
		} else if isSecretNotFoundError(err) {
			// generate and store Prometheus secret
			secretName, err = op.generatePrometheusSecret(ctx, cluster)
			if err != nil {
				return "", errors.WrapIf(err, "failed to generate Prometheus secret")
			}
		} else {
			return "", errors.WrapIf(err, "error during getting Prometheus secret")
		}
	} else {
		var err error
		secretName, err = op.secretStore.GetNameByID(ctx, spec.Prometheus.SecretId)
		if err != nil {
			return "", errors.WrapIfWithDetails(err, "failed to get Prometheus secret", "secretID", spec.Prometheus.SecretId)
		}
	}
	return secretName, nil
}

func (op FeatureOperator) ensureOrgIDInContext(ctx context.Context, clusterID uint) (context.Context, error) {
	if _, ok := auth.GetCurrentOrganizationID(ctx); !ok {
		cluster, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		ctx = auth.SetCurrentOrganizationID(ctx, cluster.GetOrganizationId())
	}
	return ctx, nil
}

func (op FeatureOperator) generateAlertManagerProvidersConfig(spec providerSpec) configValues {
	return configValues{
		Global: configGlobalValues{
			Receivers: []receiverItemValues{
				{
					Name:             alertManagerProviderConfigName,
					SlackConfigs:     op.generateSlackConfig(spec.Slack),
					EmailConfigs:     op.generateEmailConfig(spec.Email),
					PagerdutyConfigs: op.generatePagerdutyConfig(spec.Pagerduty),
				},
			},
		},
	}
}

func (op FeatureOperator) generateSlackConfig(config slackPropertiesSpec) []slackConfigValues {
	if config.Enabled {
		return []slackConfigValues{
			{
				ApiUrl:       config.ApiUrl,
				Channel:      config.Channel,
				SendResolved: config.SendResolved,
			},
		}
	}

	return nil
}

func (op FeatureOperator) generateEmailConfig(config emailPropertiesSpec) []emailConfigValues {
	if config.Enabled {
		return []emailConfigValues{
			{
				To:           config.To,
				From:         config.From,
				SendResolved: config.SendResolved,
			},
		}
	}

	return nil
}

func (op FeatureOperator) generatePagerdutyConfig(config pagerdutyPropertiesSpec) []pagerdutyConfigValues {
	if config.Enabled {
		return []pagerdutyConfigValues{
			{
				RoutingKey:   config.RoutingKey,
				ServiceKey:   config.ServiceKey,
				Url:          config.Url,
				SendResolved: config.SendResolved,
			},
		}
	}

	return nil
}

func isSecretNotFoundError(err error) bool {
	errCause := errors.Cause(err)
	if errCause == secret.ErrSecretNotExists {
		return true
	}
	return false
}

func GetHeadNodeAffinity(cluster interface {
	NodePoolExists(nodePoolName string) bool
}, config Configuration) v1.Affinity {
	headNodePoolName := config.headNodepoolName
	if headNodePoolName == "" {
		return v1.Affinity{}
	}
	if !cluster.NodePoolExists(headNodePoolName) {
		return v1.Affinity{}
	}
	return v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
				{
					Weight: 100,
					Preference: v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{
							{
								Key:      pkgCommon.LabelKey,
								Operator: v1.NodeSelectorOpIn,
								Values: []string{
									headNodePoolName,
								},
							},
						},
					},
				},
			},
		},
	}
}

func GetHeadNodeTolerations(config Configuration) []v1.Toleration {
	headNodePoolName := config.headNodepoolName
	if headNodePoolName == "" {
		return []v1.Toleration{}
	}
	return []v1.Toleration{
		{
			Key:      pkgCommon.NodePoolNameTaintKey,
			Operator: v1.TolerationOpEqual,
			Value:    headNodePoolName,
		},
	}
}
