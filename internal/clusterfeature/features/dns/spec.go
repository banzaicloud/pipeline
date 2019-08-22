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
	"encoding/json"
	"fmt"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns/route53"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/pkg/secret"
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
	Options  *providerOptions `json:"options,omitempty"`
}

// providerOptions placeholder struct for additional provider specific configuration
// extrend it with the required fields here as appropriate
type providerOptions struct {
	DnsMasked          bool   `json:"dnsMasked" mapstructure:"dnsMasked"`
	AzureResourceGroup string `json:"resourceGroup,omitempty" mapstructure:"resourceGroup"`
	GoogleProject      string `json:"project,omitempty" mapstructure:"project"`
}

func (o *providerOptions) Validate(provider string) error {
	switch provider {
	case dnsAzure:
		if o == nil || len(o.AzureResourceGroup) == 0 {
			return &EmptyOptionFieldError{
				fieldName: "resourceGroup",
			}
		}
	case dnsGoogle:
		if o == nil || len(o.GoogleProject) == 0 {
			return &EmptyOptionFieldError{
				fieldName: "project",
			}
		}
	}

	return nil
}

// EmptyOptionFieldError is returned when resource group field is empty in case of Azure provider.
type EmptyOptionFieldError struct {
	fieldName string
}

func (e *EmptyOptionFieldError) Error() string {
	return fmt.Sprintf("%s cannot be empty", e.fieldName)
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

	values.Aws = providerSettings
	values.Provider = "aws"

	return values, nil
}

func (m *dnsFeatureManager) processCustomDNSFeatureValues(ctx context.Context, clusterID uint, customDns CustomDns) (*ExternalDnsChartValues, error) {

	secretValues, err := m.secretStore.GetSecretValues(ctx, customDns.Provider.SecretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to process feature spec secrets")
	}

	values, err := m.getDefaultValues(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to process default values")
	}

	switch provider := customDns.Provider.Name; provider {
	case dnsRoute53:
		if err := m.createCustomDnsChartValuesAmazon(secretValues, values); err != nil {
			return nil, errors.Wrap(err, "failed to create Amazon custom DNS chart values")
		}

	case dnsAzure:
		if err := m.createCustomDnsChartValuesAzure(
			clusterID,
			customDns.Provider.Options,
			secretValues,
			values,
		); err != nil {
			return nil, errors.Wrap(err, "failed to create Azure custom DNS chart values")
		}

	case dnsGoogle:
		if err := m.createCustomDnsChartValuesGoogle(
			clusterID,
			customDns.Provider.Options,
			secretValues,
			values,
		); err != nil {
			return nil, errors.Wrap(err, "failed to create Google custom DNS chart values")
		}

	default:

		return nil, errors.New("DNS provider must be set")
	}

	values.DomainFilters = customDns.DomainFilters
	values.Provider = getProviderNameForChart(customDns.Provider.Name)

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
	ClientID       string `json:"aadClientId" mapstructure:"AZURE_CLIENT_ID"`
	ClientSecret   string `json:"aadClientSecret" mapstructure:"AZURE_CLIENT_SECRET"`
	TenantID       string `json:"tenantId" mapstructure:"AZURE_TENANT_ID"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"AZURE_SUBSCRIPTION_ID"`
}

// installSecret installs a secret to the cluster identified by the provided clusterID
// secrets to be installed are expected to be contained in the request's value field
func (m *dnsFeatureManager) installSecret(ctx context.Context, clusterID uint, secretName string, secretRequest cluster.InstallSecretRequest) (*secret.K8SSourceMeta, error) {
	cc, err := m.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get cluster", "clusterID", clusterID)
	}

	k8sSec, err := cluster.InstallSecret(cc, secretName, secretRequest)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to install secret to the cluster", "clusterID", clusterID)
	}

	return k8sSec, nil

}

// newInstallSecretRequest creates a new request instance with provider specific settings
func (m *dnsFeatureManager) newInstallSecretRequest(provider string, secretValue string) (*cluster.InstallSecretRequest, error) {
	switch provider {
	case dnsRoute53:
		m.logger.Debug("no secrets to be installed to the cluster for this provider", map[string]interface{}{"provider": provider})

		return nil, nil
	case dnsAzure:
		req := cluster.InstallSecretRequest{
			// Note: leave the Source field empty as the secret needs to be transformed
			Namespace: externalDnsNamespace,
			Update:    true,
			Spec: map[string]cluster.InstallSecretRequestSpecItem{
				externalDnsAzureSecretDataKey: { // the data key of the k8s secret
					Value: secretValue,
				},
			},
		}

		return &req, nil
	case dnsGoogle:
		req := cluster.InstallSecretRequest{
			Namespace: externalDnsNamespace,
			Update:    true,
			Spec: map[string]cluster.InstallSecretRequestSpecItem{
				externalDnsGoogleSecretDataKey: {
					Value: secretValue,
				},
			},
		}

		return &req, nil

	default:
		return nil, errors.NewWithDetails("unsupported provider", "provider", provider)
	}
}

func (m *dnsFeatureManager) createCustomDnsChartValuesAmazon(secretValues map[string]string, values *ExternalDnsChartValues) error {
	creds := awsCredentials{}
	if err := mapstructure.Decode(secretValues, &creds); err != nil {
		return errors.WrapIf(err, "failed to bind feature spec credentials")
	}

	// set secret values
	providerSettings := &ExternalDnsAwsSettings{
		Region: creds.Region,
		Credentials: &ExternalDnsAwsCredentials{
			AccessKey: creds.AccessKeyID,
			SecretKey: creds.SecretAccessKey,
		},
	}

	values.Aws = providerSettings

	return nil
}

func (m *dnsFeatureManager) createCustomDnsChartValuesAzure(
	clusterID uint,
	options *providerOptions,
	secretValues map[string]string,
	values *ExternalDnsChartValues,
) error {
	// get parse secret values into a struct
	azCreds := azureCredentials{}
	if err := mapstructure.Decode(secretValues, &azCreds); err != nil {
		return errors.WrapIf(err, "failed to bind feature spec credentials")
	}

	if err := options.Validate(dnsAzure); err != nil {
		return errors.Wrap(err, "error during options validation")
	}

	kubeSecretVal, err := json.Marshal(
		// inline composite struct for adding  extra fields
		struct {
			*azureCredentials
			ResourceGroup string `json:"resourceGroup"`
		}{
			&azCreds,
			options.AzureResourceGroup,
		})
	if err != nil {
		return errors.WrapIf(err, "failed to marshal secret values")
	}

	req, err := m.newInstallSecretRequest(dnsAzure, string(kubeSecretVal))
	if err != nil {
		return errors.WrapIf(err, "failed to create install secret request")
	}

	k8sSec, err := m.installSecret(context.Background(), clusterID, externalDnsAzureSecret, *req)
	if err != nil {
		return errors.WrapIf(err, "failed to install secret to the cluster")
	}

	azureSettings := &ExternalDnsAzureSettings{
		SecretName:    k8sSec.Name,
		ResourceGroup: options.AzureResourceGroup,
	}
	values.Azure = azureSettings
	values.TxtPrefix = "txt-"

	return nil
}

func (m *dnsFeatureManager) createCustomDnsChartValuesGoogle(
	clusterID uint,
	options *providerOptions,
	secretValues map[string]string,
	values *ExternalDnsChartValues,
) error {
	// set google project
	if options == nil || len(options.GoogleProject) == 0 {
		options = &providerOptions{
			GoogleProject: secretValues[secret.ProjectId],
		}
	}

	if err := options.Validate(dnsGoogle); err != nil {
		return errors.Wrap(err, "error during options validation")
	}

	// create kubernetes secret values
	kubeSecretVal, err := json.Marshal(secretValues)
	if err != nil {
		return errors.WrapIf(err, "failed to marshal secret values")
	}

	req, err := m.newInstallSecretRequest(dnsGoogle, string(kubeSecretVal))
	if err != nil {
		return errors.WrapIf(err, "failed to create install secret request")
	}

	k8sSec, err := m.installSecret(context.Background(), clusterID, externalDnsGoogleSecret, *req)
	if err != nil {
		return errors.WrapIf(err, "failed to install secret to the cluster")
	}

	providerSettings := &ExternalDnsGoogleSettings{
		Project:              options.GoogleProject,
		ServiceAccountSecret: k8sSec.Name,
	}

	values.Google = providerSettings
	values.TxtPrefix = "txt-"

	return nil
}

func getProviderNameForChart(p string) string {
	switch p {
	case dnsRoute53:
		return "aws"
	default:
		return p
	}
}
