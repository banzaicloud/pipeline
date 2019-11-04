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

package externaldns

const (
	ChartVersion = "2.3.3"
	ChartName    = "stable/external-dns"
	Namespace    = "pipeline-system"
	ReleaseName  = "dns"

	AzureSecretName  = "azure-config-file"
	GoogleSecretName = "google-config-file"

	AzureSecretDataKey  = "azure.json"
	GoogleSecretDataKey = "credentials.json"
)

// ChartValues describes external-dns helm chart values (https://hub.helm.sh/charts/stable/external-dns)
type ChartValues struct {
	Sources       []string          `json:"sources,omitempty"`
	RBAC          *RBACSettings     `json:"rbac,omitempty"`
	Image         *ImageSettings    `json:"image,omitempty"`
	DomainFilters []string          `json:"domainFilters,omitempty"`
	Policy        string            `json:"policy,omitempty"`
	TXTOwnerID    string            `json:"txtOwnerId,omitempty"`
	ExtraArgs     map[string]string `json:"extraArgs,omitempty"`
	TXTPrefix     string            `json:"txtPrefix,omitempty"`
	Azure         *AzureSettings    `json:"azure,omitempty"`
	AWS           *AWSSettings      `json:"aws,omitempty"`
	Google        *GoogleSettings   `json:"google,omitempty"`
	Provider      string            `json:"provider"`
}

type RBACSettings struct {
	Create             bool   `json:"create,omitempty"`
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	APIVersion         string `json:"apiVersion,omitempty"`
	PSPEnabled         bool   `json:"pspEnabled,omitempty"`
}

type ImageSettings struct {
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

type AWSSettings struct {
	Credentials     *AWSCredentials `json:"credentials,omitempty"`
	Region          string          `json:"region,omitempty"`
	ZoneType        string          `json:"zoneType,omitempty"`
	AssumeRoleARN   string          `json:"assumeRoleArn,omitempty"`
	BatchChangeSize uint            `json:"batchChangeSize,omitempty"`
}

type AWSCredentials struct {
	AccessKey string `json:"accessKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
}

type AzureSettings struct {
	SecretName    string `json:"secretName,omitempty"`
	ResourceGroup string `json:"resourceGroup,omitempty"`
}

type GoogleSettings struct {
	Project              string `json:"project"`
	ServiceAccountSecret string `json:"serviceAccountSecret"`
	ServiceAccountKey    string `json:"serviceAccountKey"`
}
