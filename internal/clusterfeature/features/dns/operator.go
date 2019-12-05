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

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features/dns/externaldns"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/dns/route53"
)

const ReleaseName = "dns"

// FeatureOperator implements the DNS feature operator
type FeatureOperator struct {
	clusterGetter    clusterfeatureadapter.ClusterGetter
	clusterService   clusterfeature.ClusterService
	helmService      features.HelmService
	logger           common.Logger
	orgDomainService OrgDomainService
	secretStore      features.SecretStore
	config           Config
}

// OrgDomainService interface for abstracting DNS provider related operations
// intended to be used in conjunction with the autoDNS feature in pipeline
type OrgDomainService interface {
	// EnsureClusterDomain checks for the org related hosted zone, triggers the creation of it if required
	EnsureOrgDomain(ctx context.Context, clusterID uint) error
}

// MakeFeatureOperator returns a DNS feature operator
func MakeFeatureOperator(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	logger common.Logger,
	orgDomainService OrgDomainService,
	secretStore features.SecretStore,
	config Config,
) FeatureOperator {
	return FeatureOperator{
		clusterGetter:    clusterGetter,
		clusterService:   clusterService,
		helmService:      helmService,
		logger:           logger,
		orgDomainService: orgDomainService,
		secretStore:      secretStore,
		config:           config,
	}
}

const (
	// supported DNS provider names
	dnsRoute53 = "route53"
	dnsAzure   = "azure"
	dnsGoogle  = "google"
	dnsBanzai  = "banzaicloud-dns"
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

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return errors.WrapIf(err, "failed to bind feature spec")
	}

	if err := boundSpec.Validate(); err != nil {
		return errors.WrapIf(err, "spec validation failed")
	}

	if boundSpec.ExternalDNS.Provider.Name == dnsBanzai {
		if err := op.orgDomainService.EnsureOrgDomain(ctx, clusterID); err != nil {
			return errors.WrapIf(err, "failed to ensure org domain")
		}
	}

	chartValues, err := op.getChartValues(ctx, clusterID, boundSpec)
	if err != nil {
		return errors.WrapIf(err, "failed to get chart values")
	}

	if err = op.helmService.ApplyDeployment(
		ctx,
		clusterID,
		op.config.Namespace,
		op.config.Charts.ExternalDNS.Chart,
		ReleaseName,
		chartValues,
		op.config.Charts.ExternalDNS.Version,
	); err != nil {
		return errors.WrapIf(err, "failed to apply deployment")
	}

	return nil
}

// Deactivate deactivates the cluster feature
func (op FeatureOperator) Deactivate(ctx context.Context, clusterID uint, _ clusterfeature.FeatureSpec) error {
	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	if err := op.helmService.DeleteDeployment(ctx, clusterID, ReleaseName); err != nil {
		return errors.WrapIf(err, "failed to delete deployment")
	}

	return nil
}

func (op FeatureOperator) getChartValues(ctx context.Context, clusterID uint, spec dnsFeatureSpec) ([]byte, error) {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	chartValues := externaldns.ChartValues{
		Sources: spec.ExternalDNS.Sources,
		RBAC: &externaldns.RBACSettings{
			Create: cl.RbacEnabled(),
		},
		Image: &externaldns.ImageSettings{
			Repository: op.config.Charts.ExternalDNS.Values.Image.Repository,
			Tag:        op.config.Charts.ExternalDNS.Values.Image.Tag,
		},
		DomainFilters: spec.ExternalDNS.DomainFilters,
		Policy:        string(spec.ExternalDNS.Policy),
		TXTOwnerID:    string(spec.ExternalDNS.TXTOwnerID),
		TXTPrefix:     string(spec.ExternalDNS.TXTPrefix),
		Provider:      getProviderNameForChart(spec.ExternalDNS.Provider.Name),
	}

	if spec.ExternalDNS.Provider.Name == dnsBanzai {
		spec.ExternalDNS.Provider.SecretID = route53.IAMUserAccessKeySecretID
	}

	secretValues, err := op.secretStore.GetSecretValues(ctx, spec.ExternalDNS.Provider.SecretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get secret")
	}

	switch spec.ExternalDNS.Provider.Name {
	case dnsBanzai, dnsRoute53:
		chartValues.AWS = &externaldns.AWSSettings{
			Region: secretValues[secrettype.AwsRegion],
			Credentials: &externaldns.AWSCredentials{
				AccessKey: secretValues[secrettype.AwsAccessKeyId],
				SecretKey: secretValues[secrettype.AwsSecretAccessKey],
			},
		}

		if options := spec.ExternalDNS.Provider.Options; options != nil {
			chartValues.AWS.BatchChangeSize = options.BatchChangeSize
			chartValues.AWS.Region = options.Region
		}

	case dnsAzure:
		type azureSecret struct {
			ClientID       string `json:"aadClientId" mapstructure:"AZURE_CLIENT_ID"`
			ClientSecret   string `json:"aadClientSecret" mapstructure:"AZURE_CLIENT_SECRET"`
			TenantID       string `json:"tenantId" mapstructure:"AZURE_TENANT_ID"`
			SubscriptionID string `json:"subscriptionId" mapstructure:"AZURE_SUBSCRIPTION_ID"`
		}

		var secret azureSecret
		if err := mapstructure.Decode(secretValues, &secret); err != nil {
			return nil, errors.WrapIf(err, "failed to decode secret values")
		}

		secretName, err := installSecret(cl, op.config.Namespace, externaldns.AzureSecretName, externaldns.AzureSecretDataKey, secret)
		if err != nil {
			return nil, errors.WrapIfWithDetails(err, "failed to install secret to cluster", "clusterId", clusterID)
		}

		chartValues.Azure = &externaldns.AzureSettings{
			SecretName:    secretName,
			ResourceGroup: spec.ExternalDNS.Provider.Options.AzureResourceGroup,
		}

	case dnsGoogle:
		secretName, err := installSecret(cl, op.config.Namespace, externaldns.GoogleSecretName, externaldns.GoogleSecretDataKey, secretValues)
		if err != nil {
			return nil, errors.WrapIfWithDetails(err, "failed to install secret to cluster", "clusterId", clusterID)
		}

		chartValues.Google = &externaldns.GoogleSettings{
			Project:              secretValues[secrettype.ProjectId],
			ServiceAccountSecret: secretName,
		}

		if options := spec.ExternalDNS.Provider.Options; options != nil {
			chartValues.Google.Project = options.GoogleProject
		}

	default:
	}

	rawValues, err := json.Marshal(chartValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to marshal chart values")
	}

	return rawValues, nil
}

func getProviderNameForChart(p string) string {
	switch p {
	case dnsBanzai, dnsRoute53:
		return "aws"
	default:
		return p
	}
}

// installSecret installs a secret to the specified cluster
func installSecret(
	cl interface {
		GetK8sConfig() ([]byte, error)
		GetOrganizationId() uint
	},
	namespace string,
	secretName string,
	secretDataKey string,
	secretValue interface{},
) (string, error) {
	raw, err := json.Marshal(secretValue)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal secret values")
	}

	req := cluster.InstallSecretRequest{
		// Note: leave the Source field empty as the secret needs to be transformed
		Namespace: namespace,
		Update:    true,
		Spec: map[string]cluster.InstallSecretRequestSpecItem{
			secretDataKey: {
				Value: string(raw),
			},
		},
	}

	k8sSec, err := cluster.InstallSecret(cl, secretName, req)
	if err != nil {
		return "", errors.WrapIf(err, "failed to install secret to cluster")
	}

	return k8sSec.Name, nil
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
