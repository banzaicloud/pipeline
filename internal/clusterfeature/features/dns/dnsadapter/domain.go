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

package dnsadapter

import (
	"context"
	"fmt"

	"github.com/banzaicloud/pipeline/auth"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/common"
)

// NewOrgDomainService returns a new OrgDomainService initialized with the specified values
func NewOrgDomainService(baseDomain string, dnsServiceClient DNSServiceClient, clusterOrgGetter ClusterOrgGetter, logger common.Logger) OrgDomainService {
	return OrgDomainService{
		baseDomain:       baseDomain,
		dnsServiceClient: dnsServiceClient,
		clusterOrgGetter: clusterOrgGetter,
		logger:           logger,
	}
}

// OrgDomainService can be used to ensure that the organization's domain is registered with a DNS service
type OrgDomainService struct {
	baseDomain       string
	dnsServiceClient DNSServiceClient
	clusterOrgGetter ClusterOrgGetter
	logger           common.Logger
}

// EnsureOrgDomain makes sure that the organization the specified cluster belongs to has its domain registered with the DNS service
func (s OrgDomainService) EnsureOrgDomain(ctx context.Context, clusterID uint) error {
	if s.dnsServiceClient == nil {
		return errors.New("DNS service unavailable")
	}

	org, err := s.clusterOrgGetter.GetOrganization(ctx, clusterID)
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to get org for cluster", "clusterId", clusterID)
	}

	orgDomain := fmt.Sprintf("%s.%s", org.Name, s.baseDomain)

	registered, err := s.dnsServiceClient.IsDomainRegistered(org.ID, orgDomain)
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to check domain registration", "orgId", org.ID, "domain", orgDomain)
	}

	if !registered {
		if err := s.dnsServiceClient.RegisterDomain(org.ID, orgDomain); err != nil {
			return errors.WrapIfWithDetails(err, "failed to register domain", "orgId", org.ID, "domain", orgDomain)
		}
	}

	s.logger.Info("domain registration ensured", map[string]interface{}{"domain": orgDomain})

	return nil
}

// ClusterOrgGetter can be used to get the organization a cluster belongs to
type ClusterOrgGetter interface {
	GetOrganization(ctx context.Context, clusterID uint) (auth.Organization, error)
}

// NewClusterOrgGetter returns a new ClusterOrgGetter implementation
func NewClusterOrgGetter(clusterGetter ClusterGetter, orgGetter OrgGetter) ClusterOrgGetterImpl {
	return ClusterOrgGetterImpl{
		clusterGetter: clusterGetter,
	}
}

// DNSServiceClient can be used to register and check registration of an organization's domain
type DNSServiceClient interface {
	IsDomainRegistered(orgID uint, domain string) (bool, error)
	RegisterDomain(orgID uint, domain string) error
}

// ClusterOrgGetterImpl implements a ClusterOrgGetter
type ClusterOrgGetterImpl struct {
	clusterGetter ClusterGetter
	orgGetter     OrgGetter
}

// ClusterGetter can be used to get a cluster by its ID
type ClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

// OrgGetter can be used to get an organization by its ID
type OrgGetter interface {
	Get(ctx context.Context, id uint) (auth.Organization, error)
}

// GetOrganization returns the organization the specified cluster belongs to
func (g ClusterOrgGetterImpl) GetOrganization(ctx context.Context, clusterID uint) (auth.Organization, error) {
	c, err := g.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return auth.Organization{}, errors.WrapIf(err, "failed to get cluster")
	}

	org, err := g.orgGetter.Get(ctx, c.GetOrganizationId())
	if err != nil {
		return auth.Organization{}, errors.WrapIf(err, "failed to get organization")
	}

	return org, nil
}
