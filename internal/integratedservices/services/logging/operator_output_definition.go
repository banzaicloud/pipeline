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

	"emperror.dev/errors"
	"github.com/banzaicloud/logging-operator/pkg/sdk/api/v1beta1"

	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
)

func (op IntegratedServiceOperator) createClusterOutputDefinitions(ctx context.Context, spec integratedServiceSpec, cl integratedserviceadapter.Cluster) ([]outputDefinitionManager, error) {
	var creators []outputManagerCreator
	if spec.ClusterOutput.Enabled {
		// install secrets to cluster
		sourceSecretName, err := op.secretStore.GetNameByID(ctx, spec.ClusterOutput.Provider.SecretID)
		if err != nil {
			return nil, errors.WrapIfWithDetails(err, "failed to get secret name", "secretID", spec.ClusterOutput.Provider.SecretID)
		}

		if err := op.installSecretForOutput(ctx, spec.ClusterOutput, sourceSecretName, cl); err != nil {
			return nil, errors.WrapIf(err, "failed to install secret to cluster for cluster output")
		}

		creators = append(creators, outputManagerCreator{
			name:             spec.ClusterOutput.Provider.Name,
			sourceSecretName: sourceSecretName,
			providerSpec:     spec.ClusterOutput.Provider,
		})
	}

	if spec.Loki.Enabled {
		serviceURL, err := op.getLokiServiceURL(cl)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to get Loki service url")
		}

		creators = append(creators, outputManagerCreator{
			name:       providerLoki,
			serviceURL: serviceURL,
		})
	}

	if spec.ElasticSearch.Enabled {
		creators = append(creators, outputManagerCreator{
			name: providerElasticSearch,
		})
	}

	// remove old output definitions with integrated service labels
	var outputList v1beta1.ClusterOutputList
	if err := op.kubernetesService.List(ctx, cl.GetID(), map[string]string{resourceLabelKey: integratedServiceName}, &outputList); err != nil {
		return nil, errors.WrapIf(err, "failed to list output definitions")
	}

	for _, item := range outputList.Items {
		if err := op.kubernetesService.DeleteObject(ctx, cl.GetID(), &item); err != nil {
			return nil, errors.WrapIfWithDetails(err, "failed to delete output definition", "name", item.Name)
		}
	}

	// create output definition managers
	var managers = newOutputDefinitionManager(creators)
	for _, m := range managers {
		// generate output definition
		outputDefinition, err := generateOutputDefinition(ctx, m, op.secretStore, op.config.Namespace, cl.GetOrganizationId())
		if err != nil {
			return nil, errors.WrapIf(err, "failed to generate output definition")
		}

		// create new output definition
		if err := op.kubernetesService.EnsureObject(ctx, cl.GetID(), outputDefinition); err != nil {
			return nil, errors.WrapIf(err, "failed to create output definition")
		}
	}

	return managers, nil
}

func (op IntegratedServiceOperator) getLokiServiceURL(cl integratedserviceadapter.Cluster) (string, error) {
	k8sConfig, err := cl.GetK8sConfig()
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "failed to get kubeconfig", "cluster", cl.GetID())
	}

	return op.endpointsService.GetServiceURL(k8sConfig, lokiServiceName, op.config.Namespace)
}

func (op IntegratedServiceOperator) installSecretForOutput(ctx context.Context, spec clusterOutputSpec, sourceSecretName string, cl integratedserviceadapter.Cluster) error {
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

	if _, err := op.installSecret(cl, sourceSecretName, *installSecretRequest); err != nil {
		return errors.WrapIf(err, "failed to install secret to cluster")
	}

	return nil
}
