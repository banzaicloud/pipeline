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

package cluster

import (
	"fmt"

	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/providers"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	azureObjectstore "github.com/banzaicloud/pipeline/pkg/providers/azure/objectstore"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/ghodss/yaml"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// InstallLogging to install logging deployment
func InstallLogging(cluster CommonCluster, param pkgCluster.PostHookParam) error {
	var releaseTag = fmt.Sprintf("release:%s", pipConfig.LoggingReleaseName)

	var loggingParam pkgCluster.LoggingParam
	err := castToPostHookParam(&param, &loggingParam)
	if err != nil {
		return emperror.Wrap(err, "posthook param failed")
	}
	// This makes no sense since we can't check if it default false or set false
	// if !checkIfTLSRelatedValuesArePresent(&loggingParam.GenTLSForLogging) {
	// 	return errors.Errorf("TLS related parameter is missing from request!")
	// }
	namespace := viper.GetString(pipConfig.PipelineSystemNamespace)
	loggingParam.GenTLSForLogging.TLSEnabled = true
	// Set TLS default values (default True)
	if loggingParam.SecretId == "" {
		if loggingParam.SecretName == "" {
			return fmt.Errorf("either secretId or secretName has to be set")
		}
		loggingParam.SecretId = string(secret.GenerateSecretIDFromName(loggingParam.SecretName))
	}
	if loggingParam.GenTLSForLogging.Namespace == "" {
		loggingParam.GenTLSForLogging.Namespace = namespace
	}
	if loggingParam.GenTLSForLogging.TLSHost == "" {
		loggingParam.GenTLSForLogging.TLSHost = "fluentd." + loggingParam.GenTLSForLogging.Namespace + ".svc.cluster.local"
	}
	if loggingParam.GenTLSForLogging.GenTLSSecretName == "" {
		loggingParam.GenTLSForLogging.GenTLSSecretName = fmt.Sprintf("logging-tls-%d", cluster.GetID())
	}
	if loggingParam.GenTLSForLogging.TLSEnabled {
		clusterUidTag := fmt.Sprintf("clusterUID:%s", cluster.GetUID())
		req := &secret.CreateSecretRequest{
			Name: loggingParam.GenTLSForLogging.GenTLSSecretName,
			Type: pkgSecret.TLSSecretType,
			Tags: []string{
				clusterUidTag,
				pkgSecret.TagBanzaiReadonly,
				releaseTag,
			},
			Values: map[string]string{
				pkgSecret.TLSHosts: loggingParam.GenTLSForLogging.TLSHost,
			},
		}
		_, err := secret.Store.GetOrCreate(cluster.GetOrganizationId(), req)
		if err != nil {
			return errors.Errorf("failed generate TLS secrets to logging operator: %s", err)
		}
		_, err = InstallSecrets(cluster,
			&pkgSecret.ListSecretsQuery{
				Type: pkgSecret.TLSSecretType,
				Tags: []string{
					clusterUidTag,
					releaseTag,
				},
			}, loggingParam.GenTLSForLogging.Namespace)
		if err != nil {
			return errors.Errorf("could not install created TLS secret to cluster: %s", err)
		}
	}
	operatorValues := map[string]interface{}{
		"image": imageValues{
			Tag: viper.GetString(pipConfig.LoggingOperatorImageTag),
		},
		"tls": map[string]interface{}{
			"enabled":    "true",
			"secretName": loggingParam.GenTLSForLogging.GenTLSSecretName,
		},
		"affinity":    GetHeadNodeAffinity(cluster),
		"tolerations": GetHeadNodeTolerations(),
	}
	operatorYamlValues, err := yaml.Marshal(operatorValues)
	if err != nil {
		return err
	}

	chartVersion := viper.GetString(pipConfig.LoggingOperatorChartVersion)
	err = installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/logging-operator", pipConfig.LoggingReleaseName, operatorYamlValues, chartVersion, true)
	if err != nil {
		return emperror.Wrap(err, "install logging-operator failed")
	}

	operatorFluentValues := map[string]interface{}{
		"tls": map[string]interface{}{
			"enabled":    "true",
			"secretName": loggingParam.GenTLSForLogging.GenTLSSecretName,
		},
	}
	operatorFluentYamlValues, err := yaml.Marshal(operatorFluentValues)
	if err != nil {
		return err
	}
	err = installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/logging-operator-fluent", pipConfig.LoggingReleaseName+"-fluent", operatorFluentYamlValues, chartVersion, true)
	if err != nil {
		return emperror.Wrap(err, "install logging-operator-fluent failed")
	}

	// Determine the type of output plugin
	logSecret, err := secret.Store.Get(cluster.GetOrganizationId(), loggingParam.SecretId)
	if err != nil {
		return err
	}
	log.Infof("logging-hook secret type: %s", logSecret.Type)
	switch logSecret.Type {
	case pkgCluster.Amazon:
		installedSecretValues, err := InstallSecrets(cluster, &pkgSecret.ListSecretsQuery{IDs: []string{loggingParam.SecretId}}, loggingParam.GenTLSForLogging.Namespace)
		if err != nil {
			return emperror.Wrap(err, "install amazon secret failed")
		}

		if len(loggingParam.Region) == 0 {
			// region field is empty in request, get bucket region
			region, err := providers.GetBucketLocation(pkgCluster.Amazon, logSecret, loggingParam.BucketName, cluster.GetOrganizationId(), log)
			if err != nil {
				return emperror.WrapWith(err, "failed to get S3 bucket region", "bucket", loggingParam.BucketName)
			}

			loggingParam.Region = region
		}

		loggingValues := map[string]interface{}{
			"bucketName": loggingParam.BucketName,
			"region":     loggingParam.Region,
			"secret": map[string]interface{}{
				"secretName": installedSecretValues[0].Name,
			},
		}
		marshaledValues, err := yaml.Marshal(loggingValues)
		if err != nil {
			return emperror.Wrap(err, "marshaling failed")
		}
		err = installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/s3-output", "pipeline-s3-output", marshaledValues, "", false)
		if err != nil {
			return emperror.Wrap(err, "install s3-output failed")
		}
	case pkgCluster.Google:
		installedSecretValues, err := InstallSecrets(cluster, &pkgSecret.ListSecretsQuery{IDs: []string{loggingParam.SecretId}}, loggingParam.GenTLSForLogging.Namespace)
		if err != nil {
			return emperror.Wrap(err, "install google secret failed")
		}
		loggingValues := map[string]interface{}{
			"bucketName": loggingParam.BucketName,
			"secret": map[string]interface{}{
				"name": installedSecretValues[0].Name,
			},
		}
		marshaledValues, err := yaml.Marshal(loggingValues)
		if err != nil {
			return emperror.Wrap(err, "marshaling failed")
		}
		err = installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/gcs-output", "pipeline-gcs-output", marshaledValues, "", false)
		if err != nil {
			return emperror.Wrap(err, "install gcs-output failed")
		}
	case pkgCluster.Alibaba:
		installedSecretValues, err := InstallSecrets(cluster, &pkgSecret.ListSecretsQuery{IDs: []string{loggingParam.SecretId}}, loggingParam.GenTLSForLogging.Namespace)
		if err != nil {
			return emperror.Wrap(err, "could not install alibaba logging secret")
		}

		if len(loggingParam.Region) == 0 {
			// region field is empty in request, get bucket region
			region, err := providers.GetBucketLocation(pkgCluster.Alibaba, logSecret, loggingParam.BucketName, cluster.GetOrganizationId(), log)
			if err != nil {
				return emperror.WrapWith(err, "failed to get OSS bucket region", "bucket", loggingParam.BucketName)
			}

			loggingParam.Region = region
		}

		loggingValues := map[string]interface{}{
			"bucket": map[string]interface{}{
				"name":   loggingParam.BucketName,
				"region": loggingParam.Region,
			},
			"secret": map[string]interface{}{
				"name": installedSecretValues[0].Name,
			},
		}
		marshaledValues, err := yaml.Marshal(loggingValues)
		if err != nil {
			return emperror.Wrap(err, "could not marshal alibaba logging values")
		}
		err = installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/oss-output", "pipeline-oss-output", marshaledValues, "", false)
		if err != nil {
			return emperror.Wrap(err, "install oss-output failed")
		}
	case pkgCluster.Azure:

		credentials := *azure.NewCredentials(logSecret.Values)

		storageAccountClient, err := azureObjectstore.NewAuthorizedStorageAccountClientFromSecret(credentials)
		if err != nil {
			return emperror.Wrap(err, "failed to create storage account client")
		}
		sak, err := storageAccountClient.GetStorageAccountKey(loggingParam.ResourceGroup, loggingParam.StorageAccount)
		if err != nil {
			return emperror.Wrap(err, "get storage account key failed")
		}

		clusterUidTag := fmt.Sprintf("clusterUID:%s", cluster.GetUID())

		genericSecretName := fmt.Sprintf("logging-generic-%d", cluster.GetID())
		req := &secret.CreateSecretRequest{
			Name: genericSecretName,
			Type: pkgSecret.GenericSecret,
			Tags: []string{
				clusterUidTag,
				pkgSecret.TagBanzaiReadonly,
				releaseTag,
			},
			Values: map[string]string{
				"storageAccountName": loggingParam.StorageAccount,
				"storageAccountKey":  sak,
			},
		}
		if _, err = secret.Store.GetOrCreate(cluster.GetOrganizationId(), req); err != nil {
			return errors.Errorf("failed generate Generic secrets to logging operator: %s", err)
		}

		_, err = InstallSecrets(cluster,
			&pkgSecret.ListSecretsQuery{
				Type: pkgSecret.GenericSecret,
				Tags: []string{
					clusterUidTag,
					releaseTag,
				},
			}, namespace)
		if err != nil {
			return errors.Errorf("could not install created Generic secret to cluster: %s", err)
		}

		loggingValues := map[string]interface{}{
			"bucketName": loggingParam.BucketName,
			"secret": map[string]interface{}{
				"name": genericSecretName,
			},
		}

		marshaledValues, err := yaml.Marshal(loggingValues)
		if err != nil {
			return emperror.Wrap(err, "marshaling failed")
		}

		err = installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/azure-output", "pipeline-azure-output", marshaledValues, "", false)
		if err != nil {
			return emperror.Wrap(err, "install azure-output failed")
		}
	default:
		return fmt.Errorf("unexpected logging secret type: %s", logSecret.Type)
	}
	// Install output related secret
	cluster.SetLogging(true)
	return nil
}
