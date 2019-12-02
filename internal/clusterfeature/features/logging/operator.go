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
	"path"

	"emperror.dev/errors"
	"github.com/banzaicloud/logging-operator/pkg/sdk/api/v1beta1"
	"github.com/mitchellh/copystructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/util"
	"github.com/banzaicloud/pipeline/pkg/jsonstructure"
	"github.com/banzaicloud/pipeline/src/auth"
	pkgCluster "github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
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

	if err := op.processLoki(ctx, boundSpec.Loki, cl); err != nil {
		return errors.WrapIf(err, "failed to install Loki")
	}

	if err := op.createLoggingResource(ctx, clusterID, boundSpec); err != nil {
		return errors.WrapIf(err, "failed to create logging resource")
	}

	outputManagers, err := op.createClusterOutputDefinitions(ctx, boundSpec, cl)
	if err != nil {
		return errors.WrapIf(err, "failed to create cluster output definitions")
	}

	if err := op.createClusterFlowResource(ctx, outputManagers, cl.GetID()); err != nil {
		return errors.WrapIf(err, "failed to create cluster flow resource")
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

	// delete Loki deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, lokiReleaseName); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", lokiReleaseName)
	}

	// delete Logging-operator deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, loggingOperatorReleaseName); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", loggingOperatorReleaseName)
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

	// install TLS shared secret
	if _, err := op.installSecret(ctx, cl, fluentSharedSecretName, installSecretRequest); err != nil {
		return errors.WrapIfWithDetails(err,
			"failed to install fluent shared secret to the cluster",
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

		var domain = spec.Ingress.Domain
		if domain == "" {
			domain = "/"
		}

		var chartValues = &lokiValues{
			Ingress: ingressValues{
				Enabled: spec.Ingress.Enabled,
				Hosts:   []string{path.Join(domain, spec.Ingress.Path)},
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
	}

	return nil
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

func (op FeatureOperator) installSecret(ctx context.Context, cl clusterfeatureadapter.Cluster, secretName string, secretRequest pkgCluster.InstallSecretRequest) (string, error) {
	k8sSecName, err := pkgCluster.InstallSecret(cl, secretName, secretRequest)
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "failed to install secret to the cluster", "clusterID", cl.GetID())
	}

	return k8sSecName, nil
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

func (op FeatureOperator) createLoggingResource(ctx context.Context, clusterID uint, spec featureSpec) error {
	var tlsEnabled = spec.Logging.TLS
	var loggingResource = &v1beta1.Logging{
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggingResourceName,
			Namespace: op.config.Namespace,
			Labels:    map[string]string{resourceLabelKey: featureName},
		},
		Spec: v1beta1.LoggingSpec{
			FluentbitSpec: &v1beta1.FluentbitSpec{
				Image: v1beta1.ImageSpec{
					Repository: op.config.Images.Fluentbit.Repository,
					Tag:        op.config.Images.Fluentbit.Tag,
					PullPolicy: "IfNotPresent",
				},
				TLS: v1beta1.FluentbitTLS{
					Enabled: tlsEnabled,
				},
				Metrics: &v1beta1.Metrics{
					ServiceMonitor: spec.Logging.Metrics,
				},
			},
			FluentdSpec: &v1beta1.FluentdSpec{
				TLS: v1beta1.FluentdTLS{
					Enabled: tlsEnabled,
				},
				Image: v1beta1.ImageSpec{
					Repository: op.config.Images.Fluentd.Repository,
					Tag:        op.config.Images.Fluentd.Tag,
					PullPolicy: "IfNotPresent",
				},
				Metrics: &v1beta1.Metrics{
					ServiceMonitor: spec.Logging.Metrics,
				},
			},
			ControlNamespace: op.config.Namespace,
		},
	}

	if tlsEnabled {
		var sharedKey = "fluentSharedKey"
		loggingResource.Spec.FluentdSpec.TLS.SecretName = fluentSharedSecretName
		loggingResource.Spec.FluentdSpec.TLS.SharedKey = sharedKey
		loggingResource.Spec.FluentbitSpec.TLS.SecretName = fluentSharedSecretName
		loggingResource.Spec.FluentbitSpec.TLS.SharedKey = sharedKey
	}

	return op.kubernetesService.EnsureObject(ctx, clusterID, loggingResource)
}
