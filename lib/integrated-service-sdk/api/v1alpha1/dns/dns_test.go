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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClusterDomainSpec_Validate_Empty(t *testing.T) {
	err := ClusterDomainSpec("").Validate()
	require.Error(t, err)
}

func TestDomainFiltersSpec_Validate_Empty(t *testing.T) {
	err := DomainFiltersSpec(nil).Validate()
	require.NoError(t, err)
}

func TestProviderSpec_Validate(t *testing.T) {
	cases := map[string]struct {
		Spec  ProviderSpec
		Valid bool
	}{
		"missing name": {
			Spec: ProviderSpec{
				SecretID: "0123456789abcdef",
			},
			Valid: false,
		},
		"missing secret": {
			Spec: ProviderSpec{
				Name: dnsRoute53,
			},
			Valid: false,
		},
		"valid banzaicloud-dns": {
			Spec: ProviderSpec{
				Name: dnsBanzai,
			},
			Valid: true,
		},
		"missing options (azure)": {
			Spec: ProviderSpec{
				Name:     dnsAzure,
				SecretID: "0123456789abcdef",
			},
			Valid: false,
		},
		"missing resource group": {
			Spec: ProviderSpec{
				Name:     dnsAzure,
				SecretID: "0123456789abcdef",
				Options:  &ProviderOptions{},
			},
			Valid: false,
		},
		"valid azure": {
			Spec: ProviderSpec{
				Name:     dnsAzure,
				SecretID: "0123456789abcdef",
				Options: &ProviderOptions{
					AzureResourceGroup: "my-resource-group",
				},
			},
			Valid: true,
		},
		"missing options (google)": {
			Spec: ProviderSpec{
				Name:     dnsGoogle,
				SecretID: "0123456789abcdef",
			},
			Valid: false,
		},
		"missing project": {
			Spec: ProviderSpec{
				Name:     dnsGoogle,
				SecretID: "0123456789abcdef",
				Options:  &ProviderOptions{},
			},
			Valid: false,
		},
		"valid google": {
			Spec: ProviderSpec{
				Name:     dnsGoogle,
				SecretID: "0123456789abcdef",
				Options: &ProviderOptions{
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
	err := SourcesSpec(nil).Validate()
	require.NoError(t, err)
}

func TestTXTOwnerIDSpec_Validate_Empty(t *testing.T) {
	err := TxtOwnerIDSpec("").Validate()
	require.NoError(t, err)
}

func TestTXTPrefixSpec_Validate_Empty(t *testing.T) {
	err := TxtPrefixSpec("").Validate()
	require.NoError(t, err)
}
