// Copyright © 2020 Banzai Cloud
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
	"bytes"
	"context"
	"encoding/json"
	"text/template"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1/dns"
	"github.com/mitchellh/mapstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/dns/externaldns"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/dns/route53"
)

type Operator struct {
	clusterGetter    integratedserviceadapter.ClusterGetter
	clusterService   integratedservices.ClusterService
	orgDomainService OrgDomainService
	secretStore      services.SecretStore
	config           Config
	reconciler       integratedserviceadapter.Reconciler
	logger           common.Logger
}

func NewDNSISOperator(
	clusterGetter integratedserviceadapter.ClusterGetter,
	clusterService integratedservices.ClusterService,
	orgDomainService OrgDomainService,
	secretStore services.SecretStore,
	config Config,
	logger common.Logger,
) Operator {
	return Operator{
		clusterGetter:    clusterGetter,
		clusterService:   clusterService,
		orgDomainService: orgDomainService,
		secretStore:      secretStore,
		config:           config,
		reconciler:       integratedserviceadapter.NewISReconciler(logger),
		logger:           logger,
	}
}

func (o Operator) Deactivate(ctx context.Context, clusterID uint, _ integratedservices.IntegratedServiceSpec) error {
	ctx, err := o.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := o.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	cl, err := o.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve the cluster")
	}

	k8sConfig, err := cl.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve the k8s config")
	}

	si := v1alpha1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.config.Namespace,
			Name:      IntegratedServiceName,
		},
	}
	if rErr := o.reconciler.Disable(ctx, k8sConfig, si); rErr != nil {
		return errors.Wrap(rErr, "failed to reconcile the integrated service resource")
	}

	return nil
}

func (o Operator) Apply(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	ctx, err := o.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := o.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	boundSpec, err := dns.BindIntegratedServiceSpec(spec)
	if err != nil {
		return errors.WrapIf(err, "failed to bind integrated service spec")
	}

	if err := boundSpec.Validate(); err != nil {
		return errors.WrapIf(err, "spec validation failed")
	}

	if boundSpec.ExternalDNS.Provider.Name == dnsBanzai {
		if err := o.orgDomainService.EnsureOrgDomain(ctx, clusterID); err != nil {
			return errors.WrapIf(err, "failed to ensure org domain")
		}
		boundSpec.ExternalDNS.Provider.SecretID = route53.IAMUserAccessKeySecretID
	}

	secretName, err := o.secretStore.GetNameByID(ctx, boundSpec.ExternalDNS.Provider.SecretID)
	if err != nil {
		return errors.WrapIf(err, "failed to get secret name by id")
	}

	if err = o.installSecret(ctx, clusterID, secretName, boundSpec); err != nil {
		return errors.WrapIf(err, "failed to install secret")
	}

	// Update the secretID here so that it contains the actual K8s secret reference on the cluster
	// This will need to be reverted when converting the spec back to clients
	boundSpec.ExternalDNS.Provider.SecretID = secretName

	cl, err := o.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve the cluster")
	}

	k8sConfig, err := cl.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve the k8s config")
	}

	// decorate the input with cluster data
	boundSpec.RBACEnabled = cl.RbacEnabled()

	si := v1alpha1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.config.Namespace,
			Name:      IntegratedServiceName,
		},
		Spec: v1alpha1.ServiceInstanceSpec{
			Service: IntegratedServiceName,
			Enabled: nil,
			DNS: v1alpha1.DNS{
				Spec: &boundSpec,
			},
		},
	}

	if rErr := o.reconciler.Reconcile(ctx, k8sConfig, si); rErr != nil {
		return errors.Wrap(rErr, "failed to reconcile the integrated service resource")
	}

	return nil
}

func (o Operator) Name() string {
	return IntegratedServiceName
}

// installSecret installs secret to the cluster (from the vault secret store) and returns the name
func (o Operator) installSecret(ctx context.Context, clusterID uint, secretName string, spec dns.ServiceSpec) error {
	cl, err := o.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster")
	}

	// the secretID is always populated here!
	secretValues, err := o.secretStore.GetSecretValues(ctx, spec.ExternalDNS.Provider.SecretID)
	if err != nil {
		return errors.WrapIf(err, "failed to get secret")
	}

	switch spec.ExternalDNS.Provider.Name {
	case dnsBanzai, dnsRoute53:
		const credentialsTpl = `
[default]
aws_access_key_id={{.AWS_ACCESS_KEY_ID}}
aws_secret_access_key={{.AWS_SECRET_ACCESS_KEY}}
`
		t := template.Must(template.New("default").Parse(credentialsTpl))
		var tpl bytes.Buffer
		if err := t.Execute(&tpl, secretValues); err != nil {
			return errors.WrapIf(err, "failed to transform aws secret")
		}

		if _, err := installSecret(cl, o.config.Namespace, secretName, externaldns.AwsSecretDataKey, tpl.Bytes()); err != nil {
			return errors.WrapIf(err, "failed to install aws secret")
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
			return errors.WrapIf(err, "failed to decode secret values")
		}

		secretBytes, mErr := json.Marshal(secret)
		if mErr != nil {
			return errors.Wrap(err, "failed to marshal secret values")
		}

		if _, err := installSecret(cl, o.config.Namespace, secretName, externaldns.AzureSecretDataKey, secretBytes); err != nil {
			return errors.WrapIfWithDetails(err, "failed to install secret to cluster", "clusterId", clusterID)
		}
	case dnsGoogle:
		secretBytes, mErr := json.Marshal(secretValues)
		if mErr != nil {
			return errors.Wrap(err, "failed to marshal secret values")
		}

		if _, err := installSecret(cl, o.config.Namespace, secretName, externaldns.GoogleSecretDataKey, secretBytes); err != nil {
			return errors.WrapIfWithDetails(err, "failed to install secret to cluster", "clusterId", clusterID)
		}
	default:
	}

	return nil
}

func (o Operator) ensureOrgIDInContext(ctx context.Context, clusterID uint) (context.Context, error) {
	if _, ok := auth.GetCurrentOrganizationID(ctx); !ok {
		cluster, err := o.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		ctx = auth.SetCurrentOrganizationID(ctx, cluster.GetOrganizationId())
	}
	return ctx, nil
}
