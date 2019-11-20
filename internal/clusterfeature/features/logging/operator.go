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

package logging

import (
	"context"
	"encoding/json"

	"emperror.dev/errors"
	"github.com/banzaicloud/logging-operator/pkg/sdk/api/v1beta1"
	"github.com/banzaicloud/logging-operator/pkg/sdk/model/output"
	"github.com/mitchellh/copystructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/auth"
	pkgCluster "github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/util"
	"github.com/banzaicloud/pipeline/pkg/jsonstructure"
	"github.com/banzaicloud/pipeline/secret"
)

// FeatureOperator implements the Logging feature operator
type FeatureOperator struct {
	clusterGetter     clusterfeatureadapter.ClusterGetter
	clusterService    clusterfeature.ClusterService
	helmService       features.HelmService
	kubernetesService features.KubernetesService
	endpointsService  endpoints.EndpointService
	config            Config
	logger            common.Logger
	secretStore       features.SecretStore
}

// MakeFeatureOperator returns a Logging feature operator
func MakeFeatureOperator(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	kubernetesService features.KubernetesService,
	endpointsService endpoints.EndpointService,
	config Config,
	logger common.Logger,
	secretStore features.SecretStore,
) FeatureOperator {
	return FeatureOperator{
		clusterGetter:     clusterGetter,
		clusterService:    clusterService,
		helmService:       helmService,
		kubernetesService: kubernetesService,
		endpointsService:  endpointsService,
		config:            config,
		logger:            logger,
		secretStore:       secretStore,
	}
}

// Name returns the name of the Logging feature
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

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     err.Error(),
		}
	}

	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster")
	}

	if err := op.processTLS(ctx, boundSpec, cl); err != nil {
		return errors.WrapIf(err, "failed to generate and install TLS secret to the cluster")
	}

	if err := op.installLoggingOperator(ctx, cl.GetID()); err != nil {
		return errors.WrapIf(err, "failed to install logging-operator")
	}

	if err := op.installLoggingOperatorLogging(ctx, cl.GetID(), boundSpec); err != nil {
		return errors.WrapIf(err, "failed to install logging-operator-logging")
	}

	if err := op.processClusterOutput(ctx, boundSpec.ClusterOutput, cl); err != nil {
		return errors.WrapIf(err, "failed to create output definition and flow resource")
	}

	if err := op.processLoki(ctx, boundSpec.Loki, cl); err != nil {
		return errors.WrapIf(err, "failed to install Loki")
	}

	return nil
}

func (op FeatureOperator) installLoggingOperatorLogging(ctx context.Context, clusterID uint, spec featureSpec) error {
	var tlsEnabled = spec.Logging.TLS
	var chartValues = loggingOperatorLoggingValues{
		Tls: tlsValues{
			Enabled: tlsEnabled,
		},
		Fluentbit: fluentValues{
			Enabled: true,
			Image: imageValues{
				Repository: op.config.Images.Fluentbit.Repository,
				Tag:        op.config.Images.Fluentbit.Tag,
				PullPolicy: "IfNotPresent",
			},
			Metrics: metricsValues{
				ServiceMonitor: spec.Logging.Metrics,
			},
		},
		Fluentd: fluentValues{
			Enabled: true,
			Image: imageValues{
				Repository: op.config.Images.Fluentd.Repository,
				Tag:        op.config.Images.Fluentd.Tag,
				PullPolicy: "IfNotPresent",
			},
			Metrics: metricsValues{
				ServiceMonitor: spec.Logging.Metrics,
			},
		},
	}

	if tlsEnabled {
		chartValues.Tls.FluentdSecretName = fluentdSecretName
		chartValues.Tls.FluentbitSecretName = fluentbitSecretName
	}

	loggingConfigValues, err := copystructure.Copy(op.config.Charts.Logging.Values)
	if err != nil {
		return errors.WrapIf(err, "failed to copy logging values")
	}
	valuesBytes, err := mergeValuesWithConfig(chartValues, loggingConfigValues)
	if err != nil {
		return errors.WrapIf(err, "failed to merge logging values with config")
	}

	var chartName = op.config.Charts.Logging.Chart
	var chartVersion = op.config.Charts.Logging.Version
	return op.helmService.ApplyDeployment(
		ctx,
		clusterID,
		op.config.Namespace,
		chartName,
		loggingOperatorLoggingReleaseName,
		valuesBytes,
		chartVersion,
	)
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

	// delete Loki deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, lokiReleaseName); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", lokiReleaseName)
	}

	// delete Logging-operator deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, loggingOperatorReleaseName); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", loggingOperatorReleaseName)
	}

	// delete Logging-operator-logging deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, loggingOperatorLoggingReleaseName); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", loggingOperatorLoggingReleaseName)
	}

	return nil
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

func (op FeatureOperator) processTLS(ctx context.Context, spec featureSpec, cl clusterfeatureadapter.Cluster) error {
	if spec.Logging.TLS {

		// generate TLS secret and save to Vault
		if err := op.generateTLSSecret(cl); err != nil {
			return errors.WrapIf(err, "failed to generate TLS secret")
		}

		// install secret to cluster
		if err := op.installTLSSecretsToCluster(ctx, cl); err != nil {
			return errors.WrapIf(err, "failed to install TLS secret to the cluster")
		}
	}
	return nil
}

func (op FeatureOperator) generateTLSSecret(cl clusterfeatureadapter.Cluster) error {
	var namespace = op.config.Namespace
	var clusterUIDSecretTag = generateClusterUIDSecretTag(cl.GetUID())
	var clusterNameSecretTag = generateClusterNameSecretTag(cl.GetName())
	var tlsHost = "fluentd." + namespace + ".svc.cluster.local"

	req := &secret.CreateSecretRequest{
		Name: tlsSecretName,
		Type: secrettype.TLSSecretType,
		Tags: []string{
			clusterNameSecretTag,
			clusterUIDSecretTag,
			secret.TagBanzaiReadonly,
			releaseSecretTag,
			featureSecretTag,
		},
		Values: map[string]string{
			secrettype.TLSHosts: tlsHost,
		},
	}

	_, err := secret.Store.GetOrCreate(cl.GetOrganizationId(), req)
	if err != nil {
		return errors.WrapIf(err, "failed to create secret")
	}

	return nil
}
func (op FeatureOperator) generateHTPasswordSecretForLoki(ctx context.Context, cl clusterfeatureadapter.Cluster) error {
	var clusterNameSecretTag = generateClusterNameSecretTag(cl.GetName())
	var clusterUIDSecretTag = generateClusterUIDSecretTag(cl.GetUID())

	var secretTags = []string{
		clusterNameSecretTag,
		clusterUIDSecretTag,
		releaseSecretTag,
		featureSecretTag,
		lokiSecretTag,
	}

	adminPass, err := secret.RandomString("randAlphaNum", 12)
	if err != nil {
		return errors.WrapIf(err, "Loki password generation failed")
	}

	secretRequest := &secret.CreateSecretRequest{
		Name: getLokiSecretName(cl.GetID()),
		Type: secrettype.HtpasswdSecretType,
		Values: map[string]string{
			secrettype.Username: generatedSecretUsername,
			secrettype.Password: adminPass,
		},
		Tags: secretTags,
	}
	_, err = secret.Store.CreateOrUpdate(cl.GetOrganizationId(), secretRequest)
	if err != nil {
		return errors.WrapIf(err, "failed to store Loki secret")
	}

	return nil
}

func (op FeatureOperator) installTLSSecretsToCluster(ctx context.Context, cl clusterfeatureadapter.Cluster) error {

	const kubeCaCertKey = "ca.crt"
	const kubeTlsCertKey = "tls.crt"
	const kubeTlsKeyKey = "tls.key"

	var namespace = op.config.Namespace
	var installSecretRequest = pkgCluster.InstallSecretRequest{
		SourceSecretName: tlsSecretName,
		Namespace:        namespace,
		Update:           true,
		Spec: map[string]pkgCluster.InstallSecretRequestSpecItem{
			kubeCaCertKey:  {Source: secrettype.CACert},
			kubeTlsCertKey: {Source: secrettype.ClientCert},
			kubeTlsKeyKey:  {Source: secrettype.ClientKey},
		},
	}

	// install fluentbit secret
	if _, err := op.installSecret(ctx, cl, fluentbitSecretName, installSecretRequest); err != nil {
		return errors.WrapIfWithDetails(err,
			"failed to install fluentbit secret to the cluster",
			"clusterID", cl.GetID())
	}

	// install fluentd secret
	if _, err := op.installSecret(ctx, cl, fluentdSecretName, installSecretRequest); err != nil {
		return errors.WrapIfWithDetails(err,
			"failed to install fluentd secret to the cluster",
			"clusterID", cl.GetID())
	}

	return nil
}

func (op FeatureOperator) processLoki(ctx context.Context, spec lokiSpec, cl clusterfeatureadapter.Cluster) error {
	if spec.Enabled {
		var chartName = op.config.Charts.Loki.Chart
		var chartVersion = op.config.Charts.Loki.Version

		var annotations map[string]interface{}
		if spec.Ingress.Enabled {
			secretName, err := op.getLokiSecret(ctx, spec.Ingress, cl)
			if err != nil {
				return errors.WrapIf(err, "failed to get Loki secret")
			}

			if err := op.installLokiSecret(ctx, secretName, cl); err != nil {
				return errors.WrapIf(err, "failed to install Loki secret to cluster")
			}

			annotations = generateAnnotations(secretName)
		}

		var chartValues = &lokiValues{
			Ingress: ingressValues{
				Enabled: spec.Ingress.Enabled,
				Hosts:   []string{spec.Ingress.Domain},
				Path:    spec.Ingress.Path,
			},
			Annotations: annotations,
			Image: imageValues{
				Repository: op.config.Images.Loki.Repository,
				Tag:        op.config.Images.Loki.Tag,
			},
		}

		lokiConfigValues, err := copystructure.Copy(op.config.Charts.Loki.Values)
		if err != nil {
			return errors.WrapIf(err, "failed to copy loki values")
		}
		valuesBytes, err := mergeValuesWithConfig(chartValues, lokiConfigValues)
		if err != nil {
			return errors.WrapIf(err, "failed to merge loki values with config")
		}

		if err := op.helmService.ApplyDeployment(
			ctx,
			cl.GetID(),
			op.config.Namespace,
			chartName,
			lokiReleaseName,
			valuesBytes,
			chartVersion,
		); err != nil {
			return errors.WrapIf(err, "failed to apply Loki deployment")
		}

		if err := op.createLokiOutputDefinition(ctx, cl); err != nil {
			return errors.WrapIf(err, "failed to create output definition for Loki")
		}

		if err := op.createLokiFlowResource(ctx, cl.GetID()); err != nil {
			return errors.WrapIf(err, "failed to create flow resource for Loki")
		}
	}

	return nil
}

func (op FeatureOperator) createLokiOutputDefinition(ctx context.Context, cl clusterfeatureadapter.Cluster) error {
	k8sConfig, err := cl.GetK8sConfig()
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to get kubeconfig", "cluster", cl.GetID())
	}

	serviceURL, err := op.endpointsService.GetServiceURL(k8sConfig, lokiServiceName, op.config.Namespace)
	if err != nil {
		return errors.WrapIf(err, "failed to get Loki service url")
	}

	// delete former Loki outputs
	var formerOutputs v1beta1.ClusterOutputList
	if err := op.kubernetesService.List(ctx, cl.GetID(), &formerOutputs); err != nil {
		return errors.WrapIf(err, "failed to list cluster outputs")
	}
	for _, item := range formerOutputs.Items {
		if item.Name == lokiOutputDefinitionName {
			if err := op.kubernetesService.DeleteObject(ctx, cl.GetID(), &item); err != nil {
				return errors.WrapIf(err, "failed to delete Loki cluster outputs")
			}
		}
	}

	var outputDef = &v1beta1.ClusterOutput{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lokiOutputDefinitionName,
			Namespace: op.config.Namespace,
		},
		Spec: v1beta1.ClusterOutputSpec{
			OutputSpec: v1beta1.OutputSpec{
				LokiOutput: &output.LokiOutput{
					Url:                       serviceURL,
					ConfigureKubernetesLabels: true,
				},
			},
		},
	}

	return op.kubernetesService.EnsureObject(ctx, cl.GetID(), outputDef)
}

func (op FeatureOperator) createLokiFlowResource(ctx context.Context, clusterID uint) error {
	var flowRes = &v1beta1.ClusterFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lokiFlowResourceName,
			Namespace: op.config.Namespace,
		},
		Spec: v1beta1.FlowSpec{
			Selectors:  map[string]string{},
			OutputRefs: []string{lokiOutputDefinitionName},
		},
	}

	// delete former Loki cluster flows
	var formerFlowRes v1beta1.ClusterFlowList
	if err := op.kubernetesService.List(ctx, clusterID, &formerFlowRes); err != nil {
		return errors.WrapIf(err, "failed to list cluster flows")
	}
	for _, item := range formerFlowRes.Items {
		if item.Name == lokiFlowResourceName {
			if err := op.kubernetesService.DeleteObject(ctx, clusterID, &item); err != nil {
				return errors.WrapIf(err, "failed to delete Loki cluster flow")
			}
		}
	}

	return op.kubernetesService.EnsureObject(ctx, clusterID, flowRes)
}

func (op FeatureOperator) installLokiSecret(ctx context.Context, secretName string, cl clusterfeatureadapter.Cluster) error {
	installSecretRequest := pkgCluster.InstallSecretRequest{
		SourceSecretName: secretName,
		Namespace:        op.config.Namespace,
		Spec: map[string]pkgCluster.InstallSecretRequestSpecItem{
			"auth": {Source: secrettype.HtpasswdFile},
		},
		Update: true,
	}

	if _, err := op.installSecret(ctx, cl, secretName, installSecretRequest); err != nil {
		return errors.WrapIfWithDetails(err, "failed to install Loki secret to cluster")
	}

	return nil
}

func (op FeatureOperator) getLokiSecret(
	ctx context.Context,
	ingress ingressSpec,
	cl clusterfeatureadapter.Cluster,
) (string, error) {
	var secretName string
	if ingress.SecretID == "" {
		// get secret by name, this necessary in case of feature update
		secretName = getLokiSecretName(cl.GetID())
		existingSecretID, err := op.secretStore.GetIDByName(ctx, secretName)
		if existingSecretID != "" {
			return secretName, nil
		} else if isSecretNotFoundError(err) {
			// generate and store secret
			err = op.generateHTPasswordSecretForLoki(ctx, cl)
			if err != nil {
				return "", errors.WrapIf(err, "failed to generate Loki secret")
			}
		} else {
			return "", errors.WrapIf(err, "error during getting Loki secret")
		}
	} else {
		var err error
		secretName, err = op.secretStore.GetNameByID(ctx, ingress.SecretID)
		if err != nil {
			return "", errors.WrapIfWithDetails(err,
				"failed to get Loki secret",
				"secretID", ingress.SecretID)
		}
	}
	return secretName, nil
}

func isSecretNotFoundError(err error) bool {
	errCause := errors.Cause(err)
	if errCause == secret.ErrSecretNotExists {
		return true
	}
	return false
}

func (op FeatureOperator) installSecret(ctx context.Context, cl clusterfeatureadapter.Cluster, secretName string, secretRequest pkgCluster.InstallSecretRequest) (*secret.K8SSourceMeta, error) {
	k8sSec, err := pkgCluster.InstallSecret(cl, secretName, secretRequest)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to install secret to the cluster", "clusterID", cl.GetID())
	}

	return k8sSec, nil
}

func (op FeatureOperator) installLoggingOperator(ctx context.Context, clusterID uint) error {
	var chartValues = loggingOperatorValues{
		Image: imageValues{
			Repository: op.config.Images.Operator.Repository,
			Tag:        op.config.Images.Operator.Tag,
		},
	}

	operatorConfigValues, err := copystructure.Copy(op.config.Charts.Operator.Values)
	if err != nil {
		return errors.WrapIf(err, "failed to copy operator values")
	}
	valuesBytes, err := mergeValuesWithConfig(chartValues, operatorConfigValues)
	if err != nil {
		return errors.WrapIf(err, "failed to merge operator values with config")
	}

	return op.helmService.ApplyDeployment(
		ctx,
		clusterID,
		op.config.Namespace,
		op.config.Charts.Operator.Chart,
		loggingOperatorReleaseName,
		valuesBytes,
		op.config.Charts.Operator.Version,
	)
}

func (op FeatureOperator) createOutputDefinition(ctx context.Context, spec clusterOutputSpec, cl clusterfeatureadapter.Cluster) (outputDefinitionManager, error) {
	// install secrets to cluster
	sourceSecretName, err := op.secretStore.GetNameByID(ctx, spec.Provider.SecretID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get secret name", "secretID", spec.Provider.SecretID)
	}

	if err := op.installSecretForOutput(ctx, spec, sourceSecretName, cl); err != nil {
		return nil, errors.WrapIf(err, "failed to install secret to cluster for cluster output")
	}

	// create output definition manager
	manager, err := newOutputDefinitionManager(spec.Provider.Name, sourceSecretName)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create output definition manager")
	}

	// generate output definition
	outputDefinition, err := generateOutputDefinition(ctx, manager, op.secretStore, spec, op.config.Namespace, cl.GetOrganizationId())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to generate output definition")
	}

	// remove old output definitions
	var outputList v1beta1.ClusterOutputList
	if err := op.kubernetesService.List(ctx, cl.GetID(), &outputList); err != nil {
		return nil, errors.WrapIf(err, "failed to list output definitions")
	}

	for _, item := range outputList.Items {
		if item.Name != lokiOutputDefinitionName {
			if err := op.kubernetesService.DeleteObject(ctx, cl.GetID(), &item); err != nil {
				return nil, errors.WrapIfWithDetails(err, "failed to delete output definition", "name", item.Name)
			}
		}
	}

	// create new output definition
	if err := op.kubernetesService.EnsureObject(ctx, cl.GetID(), outputDefinition); err != nil {
		return nil, errors.WrapIf(err, "failed to create output definition")
	}

	return manager, nil
}

func (op FeatureOperator) installSecretForOutput(ctx context.Context, spec clusterOutputSpec, sourceSecretName string, cl clusterfeatureadapter.Cluster) error {
	secretManager, err := newOutputSecretInstallManager(spec.Provider.Name, sourceSecretName, op.config.Namespace)
	if err != nil {
		return errors.WrapIf(err, "failed to create output secret installer")
	}

	secretValues, err := op.secretStore.GetSecretValues(ctx, spec.Provider.SecretID)
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to get secret values", "secretID", spec.Provider.SecretID)
	}

	installSecretRequest, err := secretManager.generateSecretRequest(secretValues, spec.Provider.Bucket)
	if err != nil {
		return errors.WrapIf(err, "failed to generate install secret request")
	}

	if _, err := op.installSecret(ctx, cl, sourceSecretName, *installSecretRequest); err != nil {
		return errors.WrapIf(err, "failed to install secret to cluster")
	}

	return nil
}

func (op FeatureOperator) processClusterOutput(ctx context.Context, spec clusterOutputSpec, cl clusterfeatureadapter.Cluster) error {
	if spec.Enabled {
		// create output definitions
		outputDefinition, err := op.createOutputDefinition(ctx, spec, cl)
		if err != nil {
			return errors.WrapIf(err, "failed to create output definition")
		}

		// create flow resource
		if err := op.createFlowResource(ctx, outputDefinition, cl.GetID()); err != nil {
			return errors.WrapIf(err, "failed to create flow resource")
		}
	}

	return nil
}

func (op FeatureOperator) createFlowResource(ctx context.Context, outputDefinition outputDefinitionManager, clusterID uint) error {
	var flowResource = op.generateFlowResource(outputDefinition)

	// remove old flow resources
	var flowList v1beta1.ClusterFlowList
	if err := op.kubernetesService.List(ctx, clusterID, &flowList); err != nil {
		return errors.WrapIf(err, "failed to list flow resources")
	}

	for _, item := range flowList.Items {
		if item.Name != lokiFlowResourceName {
			if err := op.kubernetesService.DeleteObject(ctx, clusterID, &item); err != nil {
				return errors.WrapIfWithDetails(err, "failed to delete flow resource", "name", item.Name)
			}
		}
	}

	// create new flow resource
	return op.kubernetesService.EnsureObject(ctx, clusterID, flowResource)
}

func (op FeatureOperator) generateFlowResource(outputDefinition outputDefinitionManager) *v1beta1.ClusterFlow {
	return &v1beta1.ClusterFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      outputDefinition.getFlowName(),
			Namespace: op.config.Namespace,
		},
		Spec: v1beta1.FlowSpec{
			Selectors:  map[string]string{},
			OutputRefs: []string{outputDefinition.getOutputName()},
		},
	}
}

func mergeValuesWithConfig(chartValues interface{}, configValues interface{}) ([]byte, error) {
	out, err := jsonstructure.Encode(chartValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to encode chart values")
	}

	result, err := util.Merge(configValues, out)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to merge values")
	}

	return json.Marshal(result)
}
