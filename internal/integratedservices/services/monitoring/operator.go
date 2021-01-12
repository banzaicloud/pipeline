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
	"github.com/mitchellh/copystructure"
	"github.com/mitchellh/mapstructure"
	"k8s.io/api/storage/v1beta1"
	v1beta12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/any"
	"github.com/banzaicloud/pipeline/pkg/jsonstructure"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/src/auth"
	pkgCluster "github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

// IntegratedServiceOperator implements the Monitoring integrated service operator
type IntegratedServiceOperator struct {
	clusterGetter     integratedserviceadapter.ClusterGetter
	clusterService    integratedservices.ClusterService
	helmService       services.HelmService
	kubernetesService KubernetesService
	config            Config
	logger            common.Logger
	secretStore       services.SecretStore
	migrator          Migrator
}

type chartValuesManager struct {
	operator  IntegratedServiceOperator
	clusterID uint
}

// MakeIntegratedServiceOperator returns a Monitoring integrated service operator
func MakeIntegratedServiceOperator(
	clusterGetter integratedserviceadapter.ClusterGetter,
	clusterService integratedservices.ClusterService,
	helmService services.HelmService,
	kubernetesService KubernetesService,
	config Config,
	logger common.Logger,
	secretStore services.SecretStore,
	migrator Migrator,
) IntegratedServiceOperator {
	return IntegratedServiceOperator{
		clusterGetter:     clusterGetter,
		clusterService:    clusterService,
		helmService:       helmService,
		kubernetesService: kubernetesService,
		config:            config,
		logger:            logger,
		secretStore:       secretStore,
		migrator:          migrator,
	}
}

// Name returns the name of the DNS integrated service
func (IntegratedServiceOperator) Name() string {
	return integratedServiceName
}

// Apply applies the provided specification to the integrated service
func (op IntegratedServiceOperator) Apply(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "integrated service": integratedServiceName})

	boundSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               err.Error(),
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
		manager := secretManager{
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
		manager := secretManager{
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

	// install Prometheus Operator
	if err := op.installPrometheusOperator(ctx, cluster, logger, boundSpec, grafanaSecretID, prometheusSecretName, alertmanagerSecretName); err != nil {
		return errors.WrapIf(err, "failed to install Prometheus operator")
	}

	// Pushgateway
	if boundSpec.Pushgateway.Enabled {
		// install Prometheus Pushgateway
		if err := op.installPrometheusPushGateway(ctx, cluster, boundSpec.Pushgateway, logger); err != nil {
			return errors.WrapIf(err, "failed to install Prometheus Pushgateway")
		}
	}

	return nil
}

// Deactivate deactivates the cluster integrated service
func (op IntegratedServiceOperator) Deactivate(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	boundSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               err.Error(),
		}
	}

	if boundSpec.Grafana.SecretId == "" {
		// Grafana secret generated in activation flow, delete it
		if err := op.deleteGrafanaSecret(ctx, clusterID); err != nil && !isSecretNotFoundError(err) {
			return errors.WrapIf(err, "failed to delete Grafana secret")
		}
	}

	if boundSpec.Prometheus.Ingress.SecretID == "" {
		// Prometheus secret generated in activation flow, delete it
		if err := op.deletePrometheusSecret(ctx, clusterID); err != nil && !isSecretNotFoundError(err) {
			return errors.WrapIf(err, "failed to delete Prometheus secret")
		}
	}

	// delete prometheus operator deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, prometheusOperatorReleaseName, op.config.Namespace); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", prometheusOperatorReleaseName)
	}

	// delete prometheus pushgateway deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, prometheusPushgatewayReleaseName, op.config.Namespace); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", prometheusPushgatewayReleaseName)
	}

	// delete custom resources
	if err := op.cleanupCRDs(ctx, clusterID); err != nil {
		op.logger.Warn("failed to delete CRDs", map[string]interface{}{"failures": err})
	}

	return nil
}

func (op IntegratedServiceOperator) installPrometheusPushGateway(
	ctx context.Context,
	cluster integratedserviceadapter.Cluster,
	spec pushgatewaySpec,
	logger common.Logger,
) error {
	chartValues := &prometheusPushgatewayValues{
		Image: imageValues{
			Repository: op.config.Images.Pushgateway.Repository,
			Tag:        op.config.Images.Pushgateway.Tag,
		},
		ServiceMonitor: serviceMonitorValues{
			Enabled:   true,
			Namespace: op.config.Namespace,
		},
	}

	pushgatewayConfigValues, err := copystructure.Copy(op.config.Charts.Pushgateway.Values)
	if err != nil {
		return errors.WrapIf(err, "failed to copy pushgateway values")
	}
	valuesBytes, err := mergeOperatorValuesWithConfig(*chartValues, pushgatewayConfigValues)
	if err != nil {
		return errors.WrapIf(err, "failed to merge pushgateway values with config")
	}

	return op.helmService.ApplyDeployment(
		ctx,
		cluster.GetID(),
		op.config.Namespace,
		op.config.Charts.Pushgateway.Chart,
		prometheusPushgatewayReleaseName,
		valuesBytes,
		op.config.Charts.Pushgateway.Version,
	)
}

func (op IntegratedServiceOperator) installPrometheusOperator(
	ctx context.Context,
	cluster integratedserviceadapter.Cluster,
	logger common.Logger,
	spec integratedServiceSpec,
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

	valuesManager := chartValuesManager{
		operator:  op,
		clusterID: cluster.GetID(),
	}

	alertmanagerValues, err := valuesManager.generateAlertmanagerChartValues(ctx, spec.Alertmanager, alertmanagerSecretName, op.config.Images.Alertmanager)
	if err != nil {
		return errors.WrapIf(err, "failed to generate Alertmanager chart values")
	}

	// create chart values
	chartValues := &prometheusOperatorValues{
		PrometheusOperator: operatorSpecValues{
			Image: imageValues{
				Repository: op.config.Images.Operator.Repository,
				Tag:        op.config.Images.Operator.Tag,
			},
			CleanupCustomResource: false,
			CreateCustomResource:  true,
		},
		Grafana:      valuesManager.generateGrafanaChartValues(spec.Grafana, grafanaUser, grafanaPass, op.config.Images.Grafana),
		Alertmanager: alertmanagerValues,
		Prometheus:   valuesManager.generatePrometheusChartValues(ctx, spec.Prometheus, prometheusSecretName, op.config.Images.Prometheus),
	}

	// todo consider disabling cleanup in favor of installing crds from the chart's crds folder, but will need to take care of upgrades in that case
	chartValues.PrometheusOperator.CreateCustomResource = false

	if spec.Exporters.Enabled {
		chartValues.KubeStateMetrics = valuesManager.generateKubeStateMetricsChartValues(spec.Exporters.KubeStateMetrics)
		if spec.Exporters.KubeStateMetrics.Enabled {
			chartValues.KsmValues = &ksmValues{Image: imageValues{
				Repository: op.config.Images.Kubestatemetrics.Repository,
				Tag:        op.config.Images.Kubestatemetrics.Tag,
			}}
		}

		chartValues.NodeExporter = valuesManager.generateNodeExporterChartValues(spec.Exporters.NodeExporter)
		if spec.Exporters.NodeExporter.Enabled {
			chartValues.NeValues = &neValues{Image: imageValues{
				Repository: op.config.Images.Nodeexporter.Repository,
				Tag:        op.config.Images.Nodeexporter.Tag,
			}}
		}
	}

	operatorConfigValues, err := copystructure.Copy(op.config.Charts.Operator.Values)
	if err != nil {
		return errors.WrapIf(err, "failed to copy operator values")
	}
	valuesBytes, err := mergeOperatorValuesWithConfig(*chartValues, operatorConfigValues)
	if err != nil {
		return errors.WrapIf(err, "failed to merge operator values with config")
	}

	if op.migrator != nil {
		release, err := op.helmService.GetDeployment(ctx, cluster.GetID(), prometheusOperatorReleaseName, op.config.Namespace)
		if err != nil {
			if !helm.ErrReleaseNotFound(err) {
				return err
			}
		} else {
			k8sClientFactory := func() (kubernetes.Interface, error) {
				kubeConfig, err := cluster.GetK8sConfig()
				if err != nil {
					return nil, err
				}
				client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
				if err != nil {
					return nil, err
				}
				return client, nil
			}

			err = op.migrator(ctx, k8sClientFactory, op.config.Namespace, release.ChartVersion, op.config.Charts.Operator.Version)
			if err != nil {
				return err
			}
		}
	}

	return op.helmService.ApplyDeployment(
		ctx,
		cluster.GetID(),
		op.config.Namespace,
		op.config.Charts.Operator.Chart,
		prometheusOperatorReleaseName,
		valuesBytes,
		op.config.Charts.Operator.Version,
	)
}

func mergeOperatorValuesWithConfig(chartValues interface{}, configValues interface{}) ([]byte, error) {
	out, err := jsonstructure.Encode(chartValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to encode chart values")
	}

	result, err := any.Merge(configValues, out, jsonstructure.DefaultMergeOptions())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to merge values")
	}

	return json.Marshal(result)
}

func (op IntegratedServiceOperator) generateGrafanaSecret(
	ctx context.Context,
	cluster integratedserviceadapter.Cluster,
	logger common.Logger,
) (string, error) {
	clusterNameSecretTag := getClusterNameSecretTag(cluster.GetName())
	clusterUIDSecretTag := getClusterUIDSecretTag(cluster.GetUID())
	releaseSecretTag := getReleaseSecretTag()

	// Generating Grafana credentials
	username := op.config.Grafana.AdminUser
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
			clusterUIDSecretTag,
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

func (op IntegratedServiceOperator) deleteGrafanaSecret(ctx context.Context, clusterID uint) error {
	secretID, err := op.secretStore.GetIDByName(ctx, getGrafanaSecretName(clusterID))
	if err != nil {
		return errors.WrapIf(err, "failed to get Grafana secret")
	}
	return op.secretStore.Delete(ctx, secretID)
}

func (op IntegratedServiceOperator) deletePrometheusSecret(ctx context.Context, clusterID uint) error {
	secretID, err := op.secretStore.GetIDByName(ctx, getPrometheusSecretName(clusterID))
	if err != nil {
		return errors.WrapIf(err, "failed to get Prometheus secret")
	}
	return op.secretStore.Delete(ctx, secretID)
}

func (op IntegratedServiceOperator) installSecret(ctx context.Context, clusterID uint, secretName string, secretRequest pkgCluster.InstallSecretRequest) (string, error) {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "failed to get cluster", "clusterID", clusterID)
	}

	k8sSecName, err := pkgCluster.InstallSecret(cl, secretName, secretRequest)
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "failed to install secret to the cluster", "clusterID", clusterID)
	}

	return k8sSecName, nil
}

func (op IntegratedServiceOperator) getGrafanaSecret(
	ctx context.Context,
	cluster integratedserviceadapter.Cluster,
	spec integratedServiceSpec,
	logger common.Logger,
) (string, error) {
	secretID := spec.Grafana.SecretId
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

func (op IntegratedServiceOperator) ensureOrgIDInContext(ctx context.Context, clusterID uint) (context.Context, error) {
	if _, ok := auth.GetCurrentOrganizationID(ctx); !ok {
		cluster, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		ctx = auth.SetCurrentOrganizationID(ctx, cluster.GetOrganizationId())
	}
	return ctx, nil
}

func (op IntegratedServiceOperator) generateAlertManagerProvidersConfig(ctx context.Context, spec map[string]interface{}) (*configValues, error) {
	var err error
	var hasProvider bool

	// generate Slack configs
	var slackConfigs []slackConfigValues
	if slackProv, ok := spec[alertmanagerProviderSlack]; ok {
		var slack slackSpec
		if err := mapstructure.Decode(slackProv, &slack); err != nil {
			return nil, errors.WrapIf(err, "failed to bind Slack config")
		}
		if slack.Enabled {
			hasProvider = true
			slackConfigs, err = op.generateSlackConfig(ctx, slack)
			if err != nil {
				return nil, errors.WrapIf(err, "failed to generate Slack config")
			}
		}
	}

	// generate PagerDuty configs
	var pageDutyConfigs []pagerdutyConfigValues
	if pdProv, ok := spec[alertmanagerProviderPagerDuty]; ok {
		var pd pagerDutySpec
		if err := mapstructure.Decode(pdProv, &pd); err != nil {
			return nil, errors.WrapIf(err, "failed to bind PagerDuty config")
		}
		if pd.Enabled {
			hasProvider = true
			pageDutyConfigs, err = op.generatePagerdutyConfig(ctx, pd)
			if err != nil {
				return nil, errors.WrapIf(err, "failed to generate PagerDuty config")
			}
		}
	}

	receiverName := alertManagerNullReceiverName
	if hasProvider {
		receiverName = alertManagerProviderConfigName
	}
	result := &configValues{
		Receivers: []receiverItemValues{
			{
				Name: receiverName,
			},
		},
		Route: routeValues{
			Receiver: receiverName,
			Routes:   []interface{}{},
		},
	}

	if slackConfigs != nil {
		result.Receivers[0].SlackConfigs = slackConfigs
	}

	if pageDutyConfigs != nil {
		result.Receivers[0].PagerdutyConfigs = pageDutyConfigs
	}

	return result, nil
}

func (op IntegratedServiceOperator) generateSlackConfig(ctx context.Context, config slackSpec) ([]slackConfigValues, error) {
	if config.Enabled {
		slackSecret, err := op.secretStore.GetSecretValues(ctx, config.SecretID)
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

	return nil, nil
}

func (op IntegratedServiceOperator) generatePagerdutyConfig(ctx context.Context, config pagerDutySpec) ([]pagerdutyConfigValues, error) {
	if config.Enabled {
		pdSecret, err := op.secretStore.GetSecretValues(ctx, config.SecretID)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to get PagerDuty secret")
		}

		pdConfig := pagerdutyConfigValues{
			Url:          config.URL,
			SendResolved: config.SendResolved,
		}

		integrationKey := pdSecret[secrettype.PagerDutyIntegrationKey]
		if config.IntegrationType == pagerDutyIntegrationEventApiV2 {
			pdConfig.RoutingKey = integrationKey
		} else {
			pdConfig.ServiceKey = integrationKey
		}

		return []pagerdutyConfigValues{pdConfig}, nil
	}

	return nil, nil
}

func isSecretNotFoundError(err error) bool {
	errCause := errors.Cause(err)
	if errCause == secret.ErrSecretNotExists {
		return true
	}
	return false
}

func (m chartValuesManager) generateGrafanaChartValues(
	spec grafanaSpec,
	username string,
	password string,
	config ImageConfig,
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
			AdminUser:     username,
			AdminPassword: password,
			GrafanaIni: grafanaIniValues{Server: grafanaIniServerValues{
				RootUrl:          fmt.Sprintf("http://0.0.0.0:3000%s/", spec.Ingress.Path),
				ServeFromSubPath: true,
			}},
			DefaultDashboardsEnabled: spec.Dashboards,
			Image: imageValues{
				Repository: config.Repository,
				Tag:        config.Tag,
			},
			Persistence: persistenceValues{
				Enabled: true,
			},
			Sidecar: sidecar{
				Datasources: datasources{
					Enabled:         true,
					Label:           "grafana_datasource",
					SearchNamespace: "ALL",
				},
			},
		}
	}

	return &grafanaValues{
		baseValues: baseValues{
			Enabled: false,
		},
	}
}

func (m chartValuesManager) generateAlertmanagerChartValues(
	ctx context.Context,
	spec alertmanagerSpec,
	secretName string,
	config ImageConfig,
) (*alertmanagerValues, error) {
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
			Spec: baseSpecValues{
				RoutePrefix: spec.Ingress.Path,
				Image: imageValues{
					Repository: config.Repository,
					Tag:        config.Tag,
				},
			},
			Config: alertmanagerConfig,
		}, nil
	}

	return &alertmanagerValues{
		baseValues: baseValues{
			Enabled: false,
		},
	}, nil
}

func (m chartValuesManager) generatePrometheusChartValues(
	ctx context.Context,
	spec prometheusSpec,
	secretName string,
	config ImageConfig,
) *prometheusValues {
	if spec.Enabled {
		defaultStorageClassName := spec.Storage.Class
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
					Enabled:     spec.Ingress.Enabled,
					Hosts:       []string{spec.Ingress.Domain},
					Paths:       []string{spec.Ingress.Path},
					Annotations: annotations,
				},
			},
			Spec: PrometheusSpecValues{
				baseSpecValues: baseSpecValues{
					RoutePrefix: spec.Ingress.Path,
					Image: imageValues{
						Repository: config.Repository,
						Tag:        config.Tag,
					},
				},
				RetentionSize: fmt.Sprintf("%.2fGiB", float64(spec.Storage.Size)*0.95),
				Retention:     spec.Storage.Retention,
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
				ServiceMonitorSelectorNilUsesHelmValues: false,
			},
		}
	}

	return &prometheusValues{
		baseValues: baseValues{
			Enabled: false,
		},
	}
}

func (m chartValuesManager) generateKubeStateMetricsChartValues(spec exporterBaseSpec) kubeStateMetricsValues {
	return kubeStateMetricsValues{
		Enabled: spec.Enabled,
	}
}

func (m chartValuesManager) generateNodeExporterChartValues(spec exporterBaseSpec) nodeExporterValues {
	return nodeExporterValues{
		Enabled: spec.Enabled,
	}
}

func (op IntegratedServiceOperator) getDefaultStorageClassName(ctx context.Context, clusterID uint) (string, error) {
	var storageClass v1beta1.StorageClassList
	if err := op.kubernetesService.List(ctx, clusterID, nil, &storageClass); err != nil {
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

// cleanupCRDs deletes CRDs after the release is deleted
func (op IntegratedServiceOperator) cleanupCRDs(ctx context.Context, clusterID uint) error {
	// list with the monitoring related CRDs
	crdNames := []string{
		"alertmanagerconfigs.monitoring.coreos.com",
		"alertmanagers.monitoring.coreos.com",
		"podmonitors.monitoring.coreos.com",
		"probes.monitoring.coreos.com",
		"prometheuses.monitoring.coreos.com",
		"prometheusrules.monitoring.coreos.com",
		"servicemonitors.monitoring.coreos.com",
		"thanosrulers.monitoring.coreos.com",
	}

	var failures error
	for _, crdName := range crdNames {
		crd := v1beta12.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: crdName,
			},
		}
		op.logger.Debug("deleting CRD", map[string]interface{}{"name": crdName})
		if err := op.kubernetesService.DeleteObject(ctx, clusterID, &crd); err != nil {
			failures = errors.Append(failures, err)
		}
	}
	return failures
}
