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
	"github.com/banzaicloud/pipeline/dns"
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

func (m *dnsFeatureManager) processAutoDNSFeatureValues(ctx context.Context, clusterID uint, autoDns AutoDns) (*dns.ExternalDnsChartValues, error) {

	values, err := m.getDefaultValues(ctx, clusterID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to process default values")
	}

	// todo make this "provider agnostic"
	route53Secret, err := m.secretStore.GetSecretByName(ctx, clusterID, route53.IAMUserAccessKeySecretName)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to get secret")
	}

	// parse secrets - aws only for the time being
	creds := awsCredentials{}
	if err := mapstructure.Decode(route53Secret, &creds); err != nil {

		return nil, errors.WrapIf(err, "failed to bind feature spec credentials")
	}

	// set secret values
	values.Aws = &dns.ExternalDnsAwsSettings{}
	values.Aws.Credentials = &dns.ExternalDnsAwsCredentials{
		AccessKey: creds.AccessKeyID,
		SecretKey: creds.SecretAccessKey,
	}

	return values, nil
}

func (m *dnsFeatureManager) processCustomDNSFeatureValues(ctx context.Context, clusterID uint, customDns CustomDns) (*dns.ExternalDnsChartValues, error) {
	secrets, err := m.secretStore.GetSecret(ctx, clusterID, customDns.Provider.SecretID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to process feature spec secrets")
	}

	// parse secrets - aws only for the time being
	creds := awsCredentials{}
	if err := mapstructure.Decode(secrets, &creds); err != nil {

		return nil, errors.WrapIf(err, "failed to bind feature spec credentials")
	}

	values, err := m.getDefaultValues(ctx, clusterID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to process default values")
	}

	// set secret values
	values.Aws = &dns.ExternalDnsAwsSettings{
		Region: creds.Region,
	}
	values.Aws.Credentials = &dns.ExternalDnsAwsCredentials{
		AccessKey: creds.AccessKeyID,
		SecretKey: creds.SecretAccessKey,
	}

	values.DomainFilters = customDns.DomainFilters

	return values, nil
}

func (m *dnsFeatureManager) getDefaultValues(ctx context.Context, clusterID uint) (*dns.ExternalDnsChartValues, error) {

	commonCluster, err := m.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	headNodeAffinity := cluster.GetHeadNodeAffinity(commonCluster)
	externalDnsValues := dns.ExternalDnsChartValues{
		Rbac: &dns.ExternalDnsRbacSettings{
			Create: commonCluster.RbacEnabled() == true,
		},
		Sources: []string{"service", "ingress"},
		Image: &dns.ExternalDnsImageSettings{
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
