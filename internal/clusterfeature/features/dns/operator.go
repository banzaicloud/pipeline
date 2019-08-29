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

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns/route53"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/pkg/secret"
)

// FeatureOperator implements the DNS feature operator
type FeatureOperator struct {
	clusterGetter    clusterfeatureadapter.ClusterGetter
	clusterService   clusterfeature.ClusterService
	helmService      features.HelmService
	logger           common.Logger
	orgDomainService OrgDomainService
	secretStore      features.SecretStore
}

// MakeFeatureOperator returns a DNS feature operator
func MakeFeatureOperator(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	logger common.Logger,
	orgDomainService OrgDomainService,
	secretStore features.SecretStore,
) FeatureOperator {
	return FeatureOperator{
		clusterGetter:    clusterGetter,
		clusterService:   clusterService,
		helmService:      helmService,
		logger:           logger,
		orgDomainService: orgDomainService,
		secretStore:      secretStore,
	}
}

const (
	externalDNSChartVersion = "2.3.3"
	externalDNSChartName    = "stable/external-dns"
	externalDNSNamespace    = "pipeline-system"
	externalDNSRelease      = "dns"

	externalDNSAzureSecret  = "azure-config-file"
	externalDNSGoogleSecret = "google-config-file"

	externalDNSAzureSecretDataKey  = "azure.json"
	externalDNSGoogleSecretDataKey = "credentials.json"

	// supported DNS provider names
	dnsRoute53 = "route53"
	dnsAzure   = "azure"
	dnsGoogle  = "google"
)

// Name returns the name of the DNS feature
func (op FeatureOperator) Name() string {
	return FeatureName
}

// Apply applies the provided specification to the cluster feature
func (op FeatureOperator) Apply(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {

		return err
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": FeatureName})

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {

		return err
	}

	dnsChartValues := &ExternalDnsChartValues{}

	switch {
	case boundSpec.AutoDNS.Enabled:
		dnsChartValues, err = op.processAutoDNSFeatureValues(ctx, clusterID, boundSpec.AutoDNS)
		if err != nil {
			logger.Debug("failed to process autoDNS values")

			return errors.WrapIf(err, "failed to process autoDNS values")
		}

		if err := op.orgDomainService.EnsureOrgDomain(ctx, clusterID); err != nil {
			logger.Debug("failed to enable autoDNS")

			return errors.WrapIf(err, "failed to register org hosted zone")
		}

		d, _, _ := op.orgDomainService.GetDomain(ctx, clusterID)

		dnsChartValues.DomainFilters = []string{d}

	case boundSpec.CustomDNS.Enabled:
		dnsChartValues, err = op.processCustomDNSFeatureValues(ctx, clusterID, boundSpec.CustomDNS)
		if err != nil {
			logger.Debug("failed to process customDNS values")

			return errors.WrapIf(err, "failed to process customDNS values")
		}
	}

	valuesBytes, err := json.Marshal(dnsChartValues)
	if err != nil {
		logger.Debug("failed to marshal values")

		return errors.WrapIf(err, "failed to decode values")
	}

	if err = op.helmService.ApplyDeployment(
		ctx,
		clusterID,
		externalDNSNamespace,
		externalDNSChartName,
		externalDNSRelease,
		valuesBytes,
		externalDNSChartVersion,
	); err != nil {
		return errors.WrapIf(err, "failed to deploy feature")
	}

	return nil
}

// Deactivate deactivates the cluster feature
func (op FeatureOperator) Deactivate(ctx context.Context, clusterID uint) error {
	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {

		return err
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": FeatureName})

	if err := op.helmService.DeleteDeployment(ctx, clusterID, externalDNSRelease); err != nil {
		logger.Info("failed to delete feature deployment")

		return errors.WrapIf(err, "failed to uninstall feature")
	}

	return nil
}

func (op FeatureOperator) processAutoDNSFeatureValues(ctx context.Context, clusterID uint, autoDNS autoDNSSpec) (*ExternalDnsChartValues, error) {

	// this is only supported for route53

	values, err := op.getDefaultValues(ctx, clusterID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to process default values")
	}

	route53Secret, err := op.secretStore.GetSecretValues(ctx, route53.IAMUserAccessKeySecretID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to get secret")
	}

	// parse secrets - aws only for the time being
	var creds awsCredentials
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

func (op FeatureOperator) processCustomDNSFeatureValues(ctx context.Context, clusterID uint, customDNS customDNSSpec) (*ExternalDnsChartValues, error) {

	secretValues, err := op.secretStore.GetSecretValues(ctx, customDNS.Provider.SecretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to process feature spec secrets")
	}

	values, err := op.getDefaultValues(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to process default values")
	}

	switch provider := customDNS.Provider.Name; provider {
	case dnsRoute53:
		if err := op.createCustomDNSChartValuesAmazon(secretValues, values); err != nil {
			return nil, errors.Wrap(err, "failed to create Amazon custom DNS chart values")
		}

	case dnsAzure:
		if err := op.createCustomDNSChartValuesAzure(
			ctx,
			clusterID,
			customDNS.Provider.Options,
			secretValues,
			values,
		); err != nil {
			return nil, errors.Wrap(err, "failed to create Azure custom DNS chart values")
		}

	case dnsGoogle:
		if err := op.createCustomDNSChartValuesGoogle(
			ctx,
			clusterID,
			customDNS.Provider.Options,
			secretValues,
			values,
		); err != nil {
			return nil, errors.Wrap(err, "failed to create Google custom DNS chart values")
		}

	default:

		return nil, errors.New("DNS provider must be set")
	}

	values.DomainFilters = customDNS.DomainFilters
	values.Provider = getProviderNameForChart(customDNS.Provider.Name)

	return values, nil
}

func getProviderNameForChart(p string) string {
	switch p {
	case dnsRoute53:
		return "aws"
	default:
		return p
	}
}

func (op FeatureOperator) getDefaultValues(ctx context.Context, clusterID uint) (*ExternalDnsChartValues, error) {

	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	return getDefaultValues(cl), nil
}

func getDefaultValues(cl clusterfeatureadapter.Cluster) *ExternalDnsChartValues {
	externalDNSValues := ExternalDnsChartValues{
		Rbac: &ExternalDnsRbacSettings{
			Create: cl.RbacEnabled(),
		},
		Sources: []string{"service", "ingress"},
		Image: &ExternalDnsImageSettings{
			Tag: viper.GetString(config.DNSExternalDnsImageVersion),
		},
		Policy:      "sync",
		TxtOwnerId:  cl.GetUID(),
		Tolerations: cluster.GetHeadNodeTolerations(),
	}

	if headNodeAffinity := cluster.GetHeadNodeAffinity(cl); headNodeAffinity != (v1.Affinity{}) {
		externalDNSValues.Affinity = &headNodeAffinity
	}

	return &externalDNSValues
}

type awsCredentials struct {
	AccessKeyID     string `mapstructure:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `mapstructure:"AWS_SECRET_ACCESS_KEY"`
	Region          string `mapstructure:"AWS_REGION"`
}

func (op FeatureOperator) createCustomDNSChartValuesAmazon(secretValues map[string]string, values *ExternalDnsChartValues) error {
	var creds awsCredentials
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

func (op FeatureOperator) createCustomDNSChartValuesAzure(
	ctx context.Context,
	clusterID uint,
	options *providerOptions,
	secretValues map[string]string,
	values *ExternalDnsChartValues,
) error {
	type azureCredentials struct {
		ClientID       string `json:"aadClientId" mapstructure:"AZURE_CLIENT_ID"`
		ClientSecret   string `json:"aadClientSecret" mapstructure:"AZURE_CLIENT_SECRET"`
		TenantID       string `json:"tenantId" mapstructure:"AZURE_TENANT_ID"`
		SubscriptionID string `json:"subscriptionId" mapstructure:"AZURE_SUBSCRIPTION_ID"`
	}

	// get parse secret values into a struct
	var azCreds azureCredentials
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
		},
	)
	if err != nil {
		return errors.WrapIf(err, "failed to marshal secret values")
	}

	req := makeInstallSecretRequest(externalDNSAzureSecretDataKey, string(kubeSecretVal))

	k8sSec, err := op.installSecret(ctx, clusterID, externalDNSAzureSecret, req)
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

func (op FeatureOperator) createCustomDNSChartValuesGoogle(
	ctx context.Context,
	clusterID uint,
	options *providerOptions,
	secretValues map[string]string,
	values *ExternalDnsChartValues,
) error {
	// set google project
	if options == nil || options.GoogleProject == "" {
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

	req := makeInstallSecretRequest(externalDNSGoogleSecretDataKey, string(kubeSecretVal))

	k8sSec, err := op.installSecret(ctx, clusterID, externalDNSGoogleSecret, req)
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

func makeInstallSecretRequest(secretDataKey string, secretValue string) cluster.InstallSecretRequest {
	return cluster.InstallSecretRequest{
		// Note: leave the Source field empty as the secret needs to be transformed
		Namespace: externalDNSNamespace,
		Update:    true,
		Spec: map[string]cluster.InstallSecretRequestSpecItem{
			secretDataKey: {
				Value: secretValue,
			},
		},
	}
}

// installSecret installs a secret to the cluster identified by the provided clusterID
// secrets to be installed are expected to be contained in the request's value field
func (op FeatureOperator) installSecret(ctx context.Context, clusterID uint, secretName string, secretRequest cluster.InstallSecretRequest) (*secret.K8SSourceMeta, error) {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get cluster", "clusterID", clusterID)
	}

	k8sSec, err := cluster.InstallSecret(cl, secretName, secretRequest)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to install secret to the cluster", "clusterID", clusterID)
	}

	return k8sSec, nil
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
