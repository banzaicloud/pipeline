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

package ingressadapter

import (
	"context"
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/integratedservices/services/ingress"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/dns"
)

type OrgDomainService struct {
	baseDomain    string
	organizations OrganizationStore
}

func NewOrgDomainService(baseDomain string, organizations OrganizationStore) OrgDomainService {
	return OrgDomainService{
		baseDomain:    baseDomain,
		organizations: organizations,
	}
}

type OrganizationStore interface {
	Get(ctx context.Context, orgID uint) (auth.Organization, error)
}

func (s OrgDomainService) GetOrgDomain(ctx context.Context, orgID uint) (ingress.OrgDomain, error) {
	if s.baseDomain == "" {
		return ingress.OrgDomain{}, nil
	}

	org, err := s.organizations.Get(ctx, orgID)
	if err != nil {
		return ingress.OrgDomain{}, errors.WrapIf(err, "failed to get organization")
	}

	orgDomainName := fmt.Sprintf("%s.%s", org.NormalizedName, s.baseDomain)

	if err = dns.ValidateSubdomain(orgDomainName); err != nil {
		return ingress.OrgDomain{}, errors.WrapIf(err, "invalid domain for TLS cert")
	}

	wildcardOrgDomainName := fmt.Sprintf("*.%s", orgDomainName)

	if err = dns.ValidateWildcardSubdomain(wildcardOrgDomainName); err != nil {
		return ingress.OrgDomain{}, errors.WrapIf(err, "invalid wildcard domain for TLS cert")
	}

	return ingress.OrgDomain{
		Name:         orgDomainName,
		WildcardName: wildcardOrgDomainName,
	}, nil
}
