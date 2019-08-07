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

package dns

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns/route53"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
)

type DnsFeatureSpec struct {
	CustomDns CustomDns `json:"customDns"`
	AutoDns   AutoDns   `json:"autoDns"`
}

type CustomDns struct {
	Enabled       bool        `json:"enabled"`
	DomainFilters []string    `json:"domainFilters"`
	ClusterDomain string      `json:"clusterDomain"`
	Provider      DnsProvider `json:"provider"`
}

type DnsProvider struct {
	Name     string           `json:"name"`
	SecretID string           `json:"secret"`
	Options  *ProviderOptions `json:"options,omitempty"`
}

type ProviderOptions struct {
	DnsMasked bool `json:"dnsMasked"`
}

type AutoDns struct {
	Enabled bool `json:"enabled"`
}

func (m *dnsFeatureManager) bindInput(ctx context.Context, spec clusterfeature.FeatureSpec) (*DnsFeatureSpec, error) {
	var dnsFeatureSpec DnsFeatureSpec

	if err := mapstructure.Decode(spec, &dnsFeatureSpec); err != nil {
		return nil, clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     "failed to bind feature spec",
		}
	}

	return &dnsFeatureSpec, nil

}

func (m *dnsFeatureManager) processAutoDNSFeatureValues(ctx context.Context, clusterID uint, autoDns AutoDns) (*ExternalDnsChartValues, error) {

	// this is only supported for route53

	values, err := m.getDefaultValues(ctx, clusterID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to process default values")
	}

	route53Secret, err := m.secretStore.GetSecretValues(ctx, route53.IAMUserAccessKeySecretID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to get secret")
	}

	// parse secrets - aws only for the time being
	creds := awsCredentials{}
	if err := mapstructure.Decode(route53Secret, &creds); err != nil {

		return nil, errors.WrapIf(err, "failed to bind feature spec credentials")
	}

	// set secret values
	providerSettings := &ExternalDnsAwsSettings{
		Region: creds.Region,
	}

	providerSettings.Credentials = &ExternalDnsAwsCredentials{
		AccessKey: creds.AccessKeyID,
		SecretKey: creds.SecretAccessKey,
	}

	values.Settings = providerSettings

	return values, nil
}

func (m *dnsFeatureManager) processCustomDNSFeatureValues(ctx context.Context, clusterID uint, customDns CustomDns) (*ExternalDnsChartValues, error) {
	//secrets, err := m.secretStore.GetSecretValues(ctx, customDns.Provider.SecretID)
	//if err != nil {
	//
	//	return nil, errors.WrapIf(err, "failed to process feature spec secrets")
	//}

	values, err := m.getDefaultValues(ctx, clusterID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to process default values")
	}

	switch customDns.Provider.Name {
	case "route53":
		creds := awsCredentials{}
		//if err := mapstructure.Decode(secrets, &creds); err != nil {
		//
		//	return nil, errors.WrapIf(err, "failed to bind feature spec credentials")
		//}

		// set secret values
		providerSettings := &ExternalDnsAwsSettings{
			Region: creds.Region,
		}
		providerSettings.Credentials = &ExternalDnsAwsCredentials{
			AccessKey: creds.AccessKeyID,
			SecretKey: creds.SecretAccessKey,
		}

		values.Aws = providerSettings

	case "azure":
		// todo install azure secret - k8s secret name: azure-config-file
		azureSettings := &ExternalDnsAzureSettings{
			SecretName:    "azure-config-file",
			ResourceGroup: "SebaWEU",
		}
		values.Azure = azureSettings
		values.TxtPrefix = "txt-"

	case "google":

		googleSttings := &ExternalDnsGoogleSettings{}
		values.Aws = googleSttings

	default:

		return nil, errors.New("DNS provider must be set")
	}

	values.DomainFilters = customDns.DomainFilters
	values.Provider = customDns.Provider.Name

	return values, nil
}

func (m *dnsFeatureManager) getDefaultValues(ctx context.Context, clusterID uint) (*ExternalDnsChartValues, error) {

	commonCluster, err := m.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	headNodeAffinity := cluster.GetHeadNodeAffinity(commonCluster)
	externalDnsValues := ExternalDnsChartValues{
		Rbac: &ExternalDnsRbacSettings{
			Create: commonCluster.RbacEnabled() == true,
		},
		Sources: []string{"service", "ingress"},
		Image: &ExternalDnsImageSettings{
			Tag: viper.GetString(config.DNSExternalDnsImageVersion),
		},
		Policy:      "sync",
		TxtOwnerId:  commonCluster.GetUID(),
		Tolerations: cluster.GetHeadNodeTolerations(),
	}

	if headNodeAffinity != (v1.Affinity{}) {
		externalDnsValues.Affinity = &headNodeAffinity
	}

	return &externalDnsValues, nil
}

// awsCredentials helper struct for getting secret values
type awsCredentials struct {
	AccessKeyID     string `mapstructure:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `mapstructure:"AWS_SECRET_ACCESS_KEY"`
	Region          string `mapstructure:"AWS_REGION"`
}

type azureCredentials struct {
	ClientID       string `mapstructure:"AZURE_CLIENT_ID"`
	ClientSecret   string `mapstructure:"AZURE_CLIENT_SECRET"`
	TenantID       string `mapstructure:"AZURE_TENANT_ID"`
	SubscriptionID string `mapstructure:"AZURE_SUBSCRIPTION_ID"`
}

type googleCredentials struct {
}
