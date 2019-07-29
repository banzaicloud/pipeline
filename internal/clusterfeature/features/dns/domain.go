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
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/dns"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common"
)

// OrgDomainService interface for abstracting DNS provider related operations
// intended to be used in conjunction with the autoDNS feature in pipeline
type OrgDomainService interface {
	// EnsureClusterDomain checks for the org related hosted zone, triggers the creation of it if required
	EnsureOrgDomain(ctx context.Context, clusterID uint) error

	GetDomain(ctx context.Context, clusterID uint) (string, uint, error)
}

type hostedZoneService struct {
	clusterGetter    clusterfeatureadapter.ClusterGetter
	dnsServiceClient dns.DnsServiceClient

	logger common.Logger
}

func NewOrgDomainService(clusterGetter clusterfeatureadapter.ClusterGetter, dnsServiceClient dns.DnsServiceClient, logger common.Logger) OrgDomainService {

	return &hostedZoneService{
		clusterGetter:    clusterGetter,
		dnsServiceClient: dnsServiceClient,

		logger: logger,
	}
}

func (h *hostedZoneService) GetDomain(ctx context.Context, clusterID uint) (string, uint, error) {
	cCluster, err := h.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		h.logger.Debug("failed to retrieve cluster", map[string]interface{}{"clusterID": clusterID})

		return "", 0, errors.WrapIf(err, "failed to retrieve cluster")
	}

	org, err := auth.GetOrganizationById(cCluster.GetOrganizationId())
	if err != nil {
		h.logger.Debug("failed to retrieve organization", map[string]interface{}{"orgID": cCluster.GetOrganizationId()})

		return "", 0, errors.WrapIff(err, "failed to retrieve organization with id %d", cCluster.GetOrganizationId())
	}

	domainBase, err := dns.GetBaseDomain()
	if err != nil {
		h.logger.Debug("failed to retrieve base domain")

		return "", 0, errors.WrapIfWithDetails(err, "failed to get base domain", "clusterID", clusterID)
	}

	domain := strings.ToLower(fmt.Sprintf("%s.%s", org.Name, domainBase))

	return domain, cCluster.GetOrganizationId(), nil
}

func (h *hostedZoneService) EnsureOrgDomain(ctx context.Context, clusterID uint) error {
	domain, orgID, err := h.GetDomain(ctx, clusterID)
	if err != nil {
		h.logger.Debug("failed to get the domain", map[string]interface{}{"clusterID": clusterID})

		return errors.WrapIf(err, "failed to retrieve cluster")
	}

	registered, err := h.dnsServiceClient.IsDomainRegistered(orgID, domain)
	if err != nil {

		return errors.Wrapf(err, "failed to check if domain '%s' is already registered", domain)
	}

	if registered {
		h.logger.Info("domain is already registered", map[string]interface{}{"domain": domain})

		return nil
	}

	// the domain is not registered, try registering it
	if err = h.dnsServiceClient.RegisterDomain(orgID, domain); err != nil {
		h.logger.Debug("failed to register hosted zone", map[string]interface{}{"orgHZ": domain})

		return errors.WrapIff(err, "failed to register org HZ '%s'", domain)
	}

	return nil
}
