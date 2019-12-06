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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClusterDomainSpec_Validate_Empty(t *testing.T) {
	err := clusterDomainSpec("").Validate()
	require.Error(t, err)
}

func TestDomainFiltersSpec_Validate_Empty(t *testing.T) {
	err := domainFiltersSpec(nil).Validate()
	require.NoError(t, err)
}

func TestProviderSpec_Validate(t *testing.T) {
	cases := map[string]struct {
		Spec  providerSpec
		Valid bool
	}{
		"missing name": {
			Spec: providerSpec{
				SecretID: "0123456789abcdef",
			},
			Valid: false,
		},
		"missing secret": {
			Spec: providerSpec{
				Name: dnsRoute53,
			},
			Valid: false,
		},
		"valid banzaicloud-dns": {
			Spec: providerSpec{
				Name: dnsBanzai,
			},
			Valid: true,
		},
		"missing options (azure)": {
			Spec: providerSpec{
				Name:     dnsAzure,
				SecretID: "0123456789abcdef",
			},
			Valid: false,
		},
		"missing resource group": {
			Spec: providerSpec{
				Name:     dnsAzure,
				SecretID: "0123456789abcdef",
				Options:  &providerOptions{},
			},
			Valid: false,
		},
		"valid azure": {
			Spec: providerSpec{
				Name:     dnsAzure,
				SecretID: "0123456789abcdef",
				Options: &providerOptions{
					AzureResourceGroup: "my-resource-group",
				},
			},
			Valid: true,
		},
		"missing options (google)": {
			Spec: providerSpec{
				Name:     dnsGoogle,
				SecretID: "0123456789abcdef",
			},
			Valid: false,
		},
		"missing project": {
			Spec: providerSpec{
				Name:     dnsGoogle,
				SecretID: "0123456789abcdef",
				Options:  &providerOptions{},
			},
			Valid: false,
		},
		"valid google": {
			Spec: providerSpec{
				Name:     dnsGoogle,
				SecretID: "0123456789abcdef",
				Options: &providerOptions{
					GoogleProject: "my-project",
				},
			},
			Valid: true,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.Spec.Validate()
			if tc.Valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestSourcesSpec_Validate_Empty(t *testing.T) {
	err := sourcesSpec(nil).Validate()
	require.NoError(t, err)
}

func TestTXTOwnerIDSpec_Validate_Empty(t *testing.T) {
	err := txtOwnerIDSpec("").Validate()
	require.NoError(t, err)
}

func TestTXTPrefixSpec_Validate_Empty(t *testing.T) {
	err := txtPrefixSpec("").Validate()
	require.NoError(t, err)
}
