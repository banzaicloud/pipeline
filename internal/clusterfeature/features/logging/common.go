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
	"fmt"
)

const (
	featureName = "logging"

	providerAmazonS3   = "s3"
	providerGoogleGCS  = "gcs"
	providerAlibabaOSS = "oss"
	providerAzure      = "azure"

	tlsSecretName                     = "logging-tls-secret"
	loggingOperatorReleaseName        = "logging-operator"
	loggingOperatorLoggingReleaseName = "logging-operator-logging"
	lokiReleaseName                   = "loki"
	lokiServiceName                   = "loki"
	releaseSecretTag                  = "release:logging"
	featureSecretTag                  = "feature:logging"
	lokiSecretTag                     = "app:loki"
	generatedSecretUsername           = "admin"
	loggingOperatorAPIVersion         = "logging.banzaicloud.io/v1beta1"
	loggingOperatorKindClusterOutput  = "ClusterOutput"
	loggingOperatorKindClusterFlow    = "ClusterFlow"

	outputDefinitionSecretKeyOSSAccessKeyID      = "accessKeyId"
	outputDefinitionSecretKeyOSSAccessKey        = "accessKeySecret"
	outputDefinitionSecretKeyS3AccessKeyID       = "awsAccessKeyId"
	outputDefinitionSecretKeyS3AccessKey         = "awsSecretAccessKey"
	outputDefinitionSecretKeyGCS                 = "credentials.json"
	outputDefinitionSecretKeyAzureStorageAccount = "azureStorageAccount"
	outputDefinitionSecretKeyAzureStorageAccess  = "azureStorageAccessKey"

	fluentbitSecretName = "logging-operator-fluentbit-secret"
	fluentdSecretName   = "logging-operator-fluentd-secret"
)

func getLokiSecretName(clusterID uint) string {
	return fmt.Sprintf("cluster-%d-loki", clusterID)
}

func generateClusterUIDSecretTag(clusterUID string) string {
	return fmt.Sprintf("clusterUID:%s", clusterUID)
}

func generateClusterNameSecretTag(clusterName string) string {
	return fmt.Sprintf("cluster:%s", clusterName)
}

func generateAnnotations(secretName string) map[string]interface{} {
	return map[string]interface{}{
		"traefik.ingress.kubernetes.io/auth-type":   "basic",
		"traefik.ingress.kubernetes.io/auth-secret": secretName,
	}
}
