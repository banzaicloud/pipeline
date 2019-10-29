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
	"github.com/mitchellh/mapstructure"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/storage/v1beta1"

	"github.com/banzaicloud/pipeline/auth"
	pkgCluster "github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/secret"
)

// FeatureOperator implements the Monitoring feature operator
type FeatureOperator struct {
	clusterGetter     clusterfeatureadapter.ClusterGetter
	clusterService    clusterfeature.ClusterService
	helmService       features.HelmService
	kubernetesService features.KubernetesService
	config            Configuration
	logger            common.Logger
	secretStore       features.SecretStore
}

type chartValuesManager struct {
	operator         FeatureOperator
	clusterID        uint
	headNodeAffinity v1.Affinity
	tolerations      []v1.Toleration
}

// MakeFeatureOperator returns a Monitoring feature operator
func MakeFeatureOperator(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	kubernetesService features.KubernetesService,
	config Configuration,
	logger common.Logger,
	secretStore features.SecretStore,
) FeatureOperator {
	return FeatureOperator{
		clusterGetter:     clusterGetter,
		clusterService:    clusterService,
		helmService:       helmService,
		kubernetesService: kubernetesService,
		config:            config,
		logger:            logger,
		secretStore:       secretStore,
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

	baseSecretInfoer := baseSecretInfoer{
		clusterID: clusterID,
	}

	// Prometheus
	var prometheusSecretName string
	if boundSpec.Prometheus.Enabled && boundSpec.Prometheus.Ingress.Enabled {
		// get Prometheus secret from spec or generate
		var manager = secretManager{
			operator: op,
			cluster:  cluster,
			tags:     []string{prometheusSecretTag},
			infoer:   prometheusSecretInfoer{baseSecretInfoer: baseSecretInfoer},
		}
		prometheusSecretName, err = generateAndInstallSecret(ctx, boundSpec.Prometheus.Ingress, manager, logger)
		if err != nil {
			return errors.WrapIf(err, "failed to setup Prometheus ingress")
		}
	}

	// Alertmanager
	var alertmanagerSecretName string
	if boundSpec.Alertmanager.Enabled && boundSpec.Alertmanager.Ingress.Enabled {
		// get Alertmanager secret from spec or generate
		var manager = secretManager{
			operator: op,
			cluster:  cluster,
			tags:     []string{alertmanagerSecretTag},
			infoer:   alertmanagerSecretInfoer{baseSecretInfoer: baseSecretInfoer},
		}
		alertmanagerSecretName, err = generateAndInstallSecret(ctx, boundSpec.Alertmanager.Ingress, manager, logger)
		if err != nil {
			return errors.WrapIf(err, "failed to setup Alertmanager ingress")
		}
	}

	// Pushgateway
	var pushgatewaySecretName string
	if boundSpec.Pushgateway.Enabled {
		if boundSpec.Pushgateway.Ingress.Enabled {
			var manager = secretManager{
				operator: op,
				cluster:  cluster,
				tags:     []string{pushgatewaySecretTag},
				infoer:   pushgatewaySecretInfoer{baseSecretInfoer: baseSecretInfoer},
			}

			pushgatewaySecretName, err = generateAndInstallSecret(ctx, boundSpec.Pushgateway.Ingress, manager, logger)
			if err != nil {
				return errors.WrapIf(err, "failed to setup Pushgateway ingress")
			}
		}

		// install Prometheus Pushgateway
		if err := op.installPrometheusPushGateway(ctx, cluster, boundSpec.Pushgateway, pushgatewaySecretName, logger); err != nil {
			return errors.WrapIf(err, "failed to install Prometheus Pushgateway")
		}
	}

	// install Prometheus Operator
	if err := op.installPrometheusOperator(ctx, cluster, logger, boundSpec, grafanaSecretID, prometheusSecretName, alertmanagerSecretName); err != nil {
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

	if boundSpec.Prometheus.Ingress.SecretId == "" {
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

func (op FeatureOperator) installPrometheusPushGateway(
	ctx context.Context,
	cluster clusterfeatureadapter.Cluster,
	spec pushgatewaySpec,
	secretName string,
	logger common.Logger,
) error {
	headNodeAffinity := GetHeadNodeAffinity(cluster, op.config)
	tolerations := GetHeadNodeTolerations(op.config)

	var annotations map[string]interface{}
	if spec.Ingress.Enabled {
		annotations = generateAnnotations(secretName)
	}

	pipelineSystemNamespace := op.config.pipelineSystemNamespace
	var chartValues = &prometheusPushgatewayValues{
		affinityValues:   affinityValues{Affinity: headNodeAffinity},
		tolerationValues: tolerationValues{Tolerations: tolerations},
		Ingress: ingressValues{
			Enabled: spec.Ingress.Enabled,
			Hosts:   []string{spec.Ingress.Domain},
			Paths:   []string{spec.Ingress.Path},
		},
		Annotations: annotations,
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
	prometheusSecretName string,
	alertmanagerSecretName string,
) error {
	var grafanaUser string
	var grafanaPass string
	if spec.Grafana.Enabled {
		grafanaSecret, err := op.secretStore.GetSecretValues(ctx, grafanaSecretID)
		if err != nil {
			return errors.WrapIf(err, "failed to get Grafana secret")
		}
		grafanaUser = grafanaSecret[secrettype.Username]
		grafanaPass = grafanaSecret[secrettype.Password]
	}

	var valuesManager = chartValuesManager{
		operator:         op,
		clusterID:        cluster.GetID(),
		headNodeAffinity: GetHeadNodeAffinity(cluster, op.config),
		tolerations:      GetHeadNodeTolerations(op.config),
	}

	alertmanagerValues, err := valuesManager.generateAlertmanagerChartValues(ctx, spec.Alertmanager, alertmanagerSecretName)
	if err != nil {
		return errors.WrapIf(err, "failed to generate Alertmanager chart values")
	}

	// create chart values
	pipelineSystemNamespace := op.config.pipelineSystemNamespace
	var chartValues = &prometheusOperatorValues{
		Grafana:          valuesManager.generateGrafanaChartValues(spec.Grafana, grafanaUser, grafanaPass),
		Alertmanager:     alertmanagerValues,
		Prometheus:       valuesManager.generatePrometheusChartValues(ctx, spec.Prometheus, prometheusSecretName),
		KubeStateMetrics: valuesManager.generateKubeStateMetricsChartValues(),
		NodeExporter:     valuesManager.generateNodeExporterChartValues(),
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
		Type: secrettype.PasswordSecretType,
		Values: map[string]string{
			secrettype.Username: username,
			secrettype.Password: password,
		},
		Tags: []string{
			clusterNameSecretTag,
			clusterUidSecretTag,
			secret.TagBanzaiReadonly,
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

func (op FeatureOperator) installSecret(ctx context.Context, clusterID uint, secretName string, secretRequest pkgCluster.InstallSecretRequest) (*secret.K8SSourceMeta, error) {
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

func (op FeatureOperator) generateAlertManagerProvidersConfig(ctx context.Context, spec map[string]interface{}) (*configValues, error) {
	var err error

	// generate Slack configs
	var slackConfigs []slackConfigValues
	if slackProv, ok := spec[alertmanagerProviderSlack]; ok {
		var slack slackSpec
		if err := mapstructure.Decode(slackProv, &slack); err != nil {
			return nil, errors.WrapIf(err, "failed to bind Slack config")
		}
		slackConfigs, err = op.generateSlackConfig(ctx, slack)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to generate Slack config")
		}
	}

	// generate PagerDuty configs
	var pageDutyConfigs []pagerdutyConfigValues
	if pdProv, ok := spec[alertmanagerProviderPagerDuty]; ok {
		var pd pagerDutySpec
		if err := mapstructure.Decode(pdProv, &pd); err != nil {
			return nil, errors.WrapIf(err, "failed to bind PagerDuty config")
		}
		pageDutyConfigs, err = op.generatePagerdutyConfig(ctx, pd)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to generate PagerDuty config")
		}
	}

	return &configValues{
		Global: configGlobalValues{
			Receivers: []receiverItemValues{
				{
					Name:             alertManagerProviderConfigName,
					SlackConfigs:     slackConfigs,
					PagerdutyConfigs: pageDutyConfigs,
				},
			},
		},
	}, nil
}

func (op FeatureOperator) generateSlackConfig(ctx context.Context, config slackSpec) ([]slackConfigValues, error) {
	slackSecret, err := op.secretStore.GetSecretValues(ctx, config.SecretId)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get Slack secret")
	}

	return []slackConfigValues{
		{
			ApiUrl:       slackSecret[secrettype.SlackApiUrl],
			Channel:      config.Channel,
			SendResolved: config.SendResolved,
		},
	}, nil

}

func (op FeatureOperator) generatePagerdutyConfig(ctx context.Context, config pagerDutySpec) ([]pagerdutyConfigValues, error) {
	pdSecret, err := op.secretStore.GetSecretValues(ctx, config.SecretId)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get PagerDuty secret")
	}

	var pdConfig = pagerdutyConfigValues{
		Url:          config.Url,
		SendResolved: config.SendResolved,
	}

	var integrationKey = pdSecret[secrettype.PagerDutyIntegrationKey]
	if config.IntegrationType == pagerDutyIntegrationEventApiV2 {
		pdConfig.RoutingKey = integrationKey
	} else {
		pdConfig.ServiceKey = integrationKey
	}

	return []pagerdutyConfigValues{pdConfig}, nil
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

func (m chartValuesManager) generateGrafanaChartValues(
	spec grafanaSpec,
	username string,
	password string,
) *grafanaValues {
	if spec.Enabled {
		return &grafanaValues{
			baseValues: baseValues{
				Enabled: spec.Enabled,
				Ingress: ingressValues{
					Enabled: spec.Ingress.Enabled,
					Hosts:   []string{spec.Ingress.Domain},
					Path:    spec.Ingress.Path,
				},
			},
			affinityValues:   affinityValues{Affinity: m.headNodeAffinity},
			tolerationValues: tolerationValues{Tolerations: m.tolerations},
			AdminUser:        username,
			AdminPassword:    password,
			GrafanaIni: grafanaIniValues{Server: grafanaIniServerValues{
				RootUrl:          fmt.Sprintf("http://0.0.0.0:3000%s/", spec.Ingress.Path),
				ServeFromSubPath: true,
			}},
			DefaultDashboardsEnabled: spec.Dashboards,
		}
	}

	return nil
}

func (m chartValuesManager) generateAlertmanagerChartValues(ctx context.Context, spec alertmanagerSpec, secretName string) (*alertmanagerValues, error) {
	if spec.Enabled {

		var annotations map[string]interface{}
		if spec.Ingress.Enabled {
			annotations = generateAnnotations(secretName)
		}

		alertmanagerConfig, err := m.operator.generateAlertManagerProvidersConfig(ctx, spec.Provider)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to generate Alertmanager Provider config")
		}

		return &alertmanagerValues{
			baseValues: baseValues{
				Enabled: spec.Enabled,
				Ingress: ingressValues{
					Enabled:     spec.Ingress.Enabled,
					Hosts:       []string{spec.Ingress.Domain},
					Paths:       []string{spec.Ingress.Path},
					Annotations: annotations,
				},
			},
			Spec: SpecValues{
				affinityValues:   affinityValues{Affinity: m.headNodeAffinity},
				tolerationValues: tolerationValues{Tolerations: m.tolerations},
				RoutePrefix:      spec.Ingress.Path,
			},
			Config: alertmanagerConfig,
		}, nil
	}

	return nil, nil
}

func (m chartValuesManager) generatePrometheusChartValues(ctx context.Context, spec prometheusSpec, secretName string) *prometheusValues {
	if spec.Enabled {

		var defaultStorageClassName = spec.Storage.Class
		if defaultStorageClassName == "" {
			var err error
			defaultStorageClassName, err = m.operator.getDefaultStorageClassName(ctx, m.clusterID)
			if err != nil {
				m.operator.logger.Warn("failed to get default storage class")
			}
		}

		var annotations map[string]interface{}
		if spec.Ingress.Enabled {
			annotations = generateAnnotations(secretName)
		}

		return &prometheusValues{
			baseValues: baseValues{
				Enabled: spec.Enabled,
				Ingress: ingressValues{
					Enabled: spec.Ingress.Enabled,
					Hosts:   []string{spec.Ingress.Domain},
					Paths:   []string{spec.Ingress.Path},
				},
			},
			Spec: SpecValues{
				tolerationValues: tolerationValues{Tolerations: m.tolerations},
				affinityValues:   affinityValues{Affinity: m.headNodeAffinity},
				RoutePrefix:      spec.Ingress.Path,
				RetentionSize:    fmt.Sprintf("%.2fGiB", float64(spec.Storage.Size)*0.95),
				Retention:        spec.Storage.Retention,
				StorageSpec: map[string]interface{}{
					"volumeClaimTemplate": map[string]interface{}{
						"spec": map[string]interface{}{
							"storageClassName": defaultStorageClassName,
							"accessModes":      []string{"ReadWriteOnce"},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"storage": fmt.Sprintf("%dGi", spec.Storage.Size),
								},
							},
						},
					},
				},
			},
			Annotations: annotations,
		}
	}

	return nil
}

func (m chartValuesManager) generateKubeStateMetricsChartValues() kubeStateMetricsValues {
	return kubeStateMetricsValues{
		Enabled: true,
		SpecValues: SpecValues{
			affinityValues:   affinityValues{Affinity: m.headNodeAffinity},
			tolerationValues: tolerationValues{Tolerations: m.tolerations},
		},
	}
}

func (m chartValuesManager) generateNodeExporterChartValues() nodeExporterValues {
	return nodeExporterValues{
		Enabled: true,
	}
}

func (op FeatureOperator) getDefaultStorageClassName(ctx context.Context, clusterID uint) (string, error) {
	var storageClass v1beta1.StorageClassList
	if err := op.kubernetesService.List(ctx, clusterID, &storageClass); err != nil {
		return "", errors.WrapIf(err, "failed to list storage classes")
	}

	var defaultStorageClassName string
	for _, sc := range storageClass.Items {
		for key, value := range sc.Annotations {
			if key == "storageclass.kubernetes.io/is-default-class" && value == "true" {
				defaultStorageClassName = sc.ObjectMeta.Name
			}
		}
	}
	return defaultStorageClassName, nil
}
