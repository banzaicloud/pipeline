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
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/any"
	"github.com/banzaicloud/pipeline/pkg/jsonstructure"
	"github.com/banzaicloud/pipeline/src/auth"
	pkgCluster "github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

// IntegratedServiceOperator implements the Logging integrated service operator
type IntegratedServiceOperator struct {
	clusterGetter     integratedserviceadapter.ClusterGetter
	clusterService    integratedservices.ClusterService
	helmService       services.HelmService
	kubernetesService KubernetesService
	endpointsService  endpoints.EndpointService
	config            Config
	logger            common.Logger
	secretStore       services.SecretStore
}

// MakeIntegratedServicesOperator returns a Logging integrated service operator
func MakeIntegratedServicesOperator(
	clusterGetter integratedserviceadapter.ClusterGetter,
	clusterService integratedservices.ClusterService,
	helmService services.HelmService,
	kubernetesService KubernetesService,
	endpointsService endpoints.EndpointService,
	config Config,
	logger common.Logger,
	secretStore services.SecretStore,
) IntegratedServiceOperator {
	return IntegratedServiceOperator{
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

// Name returns the name of the Logging integrated service
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

	boundSpec, err := bindIntegratedServiceSpec(spec)
	if err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:               err.Error(),
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

// Deactivate deactivates the integrated service
func (op IntegratedServiceOperator) Deactivate(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	// delete Loki deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, lokiReleaseName, op.config.Namespace); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", lokiReleaseName)
	}

	// delete Logging-operator deployment
	if err := op.helmService.DeleteDeployment(ctx, clusterID, loggingOperatorReleaseName, op.config.Namespace); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete deployment", "release", loggingOperatorReleaseName)
	}

	return nil
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

func (op IntegratedServiceOperator) processTLS(ctx context.Context, spec integratedServiceSpec, cl integratedserviceadapter.Cluster) error {
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

func (op IntegratedServiceOperator) generateTLSSecret(cl integratedserviceadapter.Cluster) error {
	namespace := op.config.Namespace
	clusterUIDSecretTag := generateClusterUIDSecretTag(cl.GetUID())
	clusterNameSecretTag := generateClusterNameSecretTag(cl.GetName())
	tlsHost := "fluentd." + namespace + ".svc.cluster.local"

	req := &secret.CreateSecretRequest{
		Name: tlsSecretName,
		Type: secrettype.TLSSecretType,
		Tags: []string{
			clusterNameSecretTag,
			clusterUIDSecretTag,
			secret.TagBanzaiReadonly,
			releaseSecretTag,
			integratedServiceSecretTag,
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

func (op IntegratedServiceOperator) generateHTPasswordSecretForLoki(ctx context.Context, cl integratedserviceadapter.Cluster) error {
	clusterNameSecretTag := generateClusterNameSecretTag(cl.GetName())
	clusterUIDSecretTag := generateClusterUIDSecretTag(cl.GetUID())

	secretTags := []string{
		clusterNameSecretTag,
		clusterUIDSecretTag,
		releaseSecretTag,
		integratedServiceSecretTag,
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

func (op IntegratedServiceOperator) installTLSSecretsToCluster(ctx context.Context, cl integratedserviceadapter.Cluster) error {
	const kubeCaCertKey = "ca.crt"
	const kubeTlsCertKey = "tls.crt"
	const kubeTlsKeyKey = "tls.key"

	namespace := op.config.Namespace
	installSecretRequest := pkgCluster.InstallSecretRequest{
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

func (op IntegratedServiceOperator) processLoki(ctx context.Context, spec lokiSpec, cl integratedserviceadapter.Cluster) error {
	if spec.Enabled {
		chartName := op.config.Charts.Loki.Chart
		chartVersion := op.config.Charts.Loki.Version

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

		domain := spec.Ingress.Domain
		if domain == "" {
			domain = "/"
		}

		chartValues := &lokiValues{
			Ingress: ingressValues{
				Enabled:     spec.Ingress.Enabled,
				Hosts:       []string{path.Join(domain, spec.Ingress.Path)},
				Annotations: annotations,
			},
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

func (op IntegratedServiceOperator) installLokiSecret(ctx context.Context, secretName string, cl integratedserviceadapter.Cluster) error {
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

func (op IntegratedServiceOperator) getLokiSecret(
	ctx context.Context,
	ingress ingressSpec,
	cl integratedserviceadapter.Cluster,
) (string, error) {
	var secretName string
	if ingress.SecretID == "" {
		// get secret by name, this necessary in case of integrated service update
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

func (op IntegratedServiceOperator) installSecret(ctx context.Context, cl integratedserviceadapter.Cluster, secretName string, secretRequest pkgCluster.InstallSecretRequest) (string, error) {
	k8sSecName, err := pkgCluster.InstallSecret(cl, secretName, secretRequest)
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "failed to install secret to the cluster", "clusterID", cl.GetID())
	}

	return k8sSecName, nil
}

func (op IntegratedServiceOperator) installLoggingOperator(ctx context.Context, clusterID uint) error {
	chartValues := loggingOperatorValues{
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

	return op.helmService.ApplyDeploymentSkipCRDs(
		ctx, clusterID, op.config.Namespace, op.config.Charts.Operator.Chart, loggingOperatorReleaseName, valuesBytes, op.config.Charts.Operator.Version)
}

func mergeValuesWithConfig(chartValues interface{}, configValues interface{}) ([]byte, error) {
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

func (op IntegratedServiceOperator) createLoggingResource(ctx context.Context, clusterID uint, spec integratedServiceSpec) error {
	tlsEnabled := spec.Logging.TLS
	loggingResource := &v1beta1.Logging{
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggingResourceName,
			Namespace: op.config.Namespace,
			Labels:    map[string]string{resourceLabelKey: integratedServiceName},
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
		sharedKey := "fluentSharedKey"
		loggingResource.Spec.FluentdSpec.TLS.SecretName = fluentSharedSecretName
		loggingResource.Spec.FluentdSpec.TLS.SharedKey = sharedKey
		loggingResource.Spec.FluentbitSpec.TLS.SecretName = fluentSharedSecretName
		loggingResource.Spec.FluentbitSpec.TLS.SharedKey = sharedKey
	}

	var oldLoggingResource v1beta1.Logging
	if err := op.kubernetesService.GetObject(ctx, clusterID, corev1.ObjectReference{
		Namespace: op.config.Namespace,
		Name:      loggingResourceName,
	}, &oldLoggingResource); err != nil {
		if k8sapierrors.IsNotFound(err) {
			// Logging resource is not found, create it
			return op.kubernetesService.EnsureObject(ctx, clusterID, loggingResource)
		}

		return errors.WrapIf(err, "failed to get Logging resource")
	}

	loggingResource.ResourceVersion = oldLoggingResource.ResourceVersion
	return op.kubernetesService.Update(ctx, clusterID, loggingResource)
}
