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
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/src/auth"
)

func TestOrgDomainService_EnsureDomain(t *testing.T) {
	clusterID1 := uint(42)
	clusterID2 := uint(21)
	org1 := auth.Organization{
		ID:   13,
		Name: "org-1",
	}
	org2 := auth.Organization{
		ID:   7,
		Name: "org-2",
	}

	s := NewOrgDomainService(
		"my.domain",
		dummyDNSServiceClient{
			RegisteredDomains: map[uint]string{
				org1.ID: "org-1.my.domain",
			},
		},
		dummyClusterOrgGetter{
			Mapping: map[uint]auth.Organization{
				clusterID1: org1,
				clusterID2: org2,
			},
		},
		commonadapter.NewNoopLogger(),
	)

	{
		err := s.EnsureOrgDomain(context.Background(), clusterID1)
		require.NoError(t, err)
	}

	{
		err := s.EnsureOrgDomain(context.Background(), clusterID2)
		require.NoError(t, err)
	}
}

type dummyClusterOrgGetter struct {
	Mapping map[uint]auth.Organization
}

func (d dummyClusterOrgGetter) GetOrganization(ctx context.Context, clusterID uint) (auth.Organization, error) {
	if org, ok := d.Mapping[clusterID]; ok {
		return org, nil
	}
	return auth.Organization{}, errors.New("cluster not found")
}

type dummyDNSServiceClient struct {
	RegisteredDomains map[uint]string
}

func (d dummyDNSServiceClient) IsDomainRegistered(orgID uint, domain string) (bool, error) {
	return d.RegisteredDomains[orgID] == domain, nil
}

func (d dummyDNSServiceClient) RegisterDomain(orgID uint, domain string) error {
	return nil
}
