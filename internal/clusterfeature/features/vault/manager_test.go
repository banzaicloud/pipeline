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

package vault

import (
	"context"
	"fmt"
	"testing"

	"github.com/banzaicloud/pipeline/config"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
)

func TestFeatureManager_Name(t *testing.T) {
	mng := MakeFeatureManager(nil, nil)

	assert.Equal(t, "vault", mng.Name())
}

func TestFeatureManager_GetOutput(t *testing.T) {
	clusterID := uint(42)
	clusterName := "the-cluster"

	clusterGetter := dummyClusterGetter{
		Clusters: map[uint]clusterfeatureadapter.Cluster{
			clusterID: dummyCluster{
				Name:  clusterName,
				OrgID: 13,
			},
		},
	}

	mng := MakeFeatureManager(clusterGetter, nil)

	ctx := context.Background()

	spec := obj{
		"customVault": obj{
			"enabled": false,
		},
		"settings": obj{
			"namespaces":      []string{"default"},
			"serviceAccounts": []string{"*"},
		},
	}
	output, err := mng.GetOutput(ctx, clusterID, spec)
	assert.NoError(t, err)

	vm, err := newVaultManager(vaultFeatureSpec{}, 13, 42)
	assert.NoError(t, err)

	vVersion, err := vm.getVaultVersion()
	assert.NoError(t, err)

	assert.Equal(t, clusterfeature.FeatureOutput{
		"vault": map[string]interface{}{
			"authMethodPath": "kubernetes/13/42",
			"rolePath":       "auth/kubernetes/13/42/role/pipeline-webhook",
			"version":        vVersion,
			"policy": fmt.Sprintf(`
			path "secret/data/orgs/%d/*" {
				capabilities = [ "read" ]
			}`, 13),
		},
		"webhook": map[string]interface{}{
			"version": viper.GetString(config.VaultWebhookChartVersionKey),
		},
	}, output)
}

func TestFeatureManager_ValidateSpec(t *testing.T) {
	mng := MakeFeatureManager(nil, nil)

	cases := map[string]struct {
		Spec  clusterfeature.FeatureSpec
		Error interface{}
	}{
		"empty spec": {
			Spec:  clusterfeature.FeatureSpec{},
			Error: false,
		},
		"valid spec": {
			Spec: obj{
				"customVault": obj{
					"address": "thisismyaddress",
				},
				"settings": obj{
					"namespaces":      []string{"default"},
					"serviceAccounts": []string{"default"},
				},
			},
			Error: false,
		},
		"both service account and namespaces are '*'": {
			Spec: obj{
				"settings": obj{
					"namespaces":      []string{"*"},
					"serviceAccounts": []string{"*"},
				},
			},
			Error: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			err := mng.ValidateSpec(ctx, tc.Spec)
			switch tc.Error {
			case true:
				assert.True(t, clusterfeature.IsInputValidationError(err))
			case false, nil:
				assert.NoError(t, err)
			default:
				assert.Equal(t, tc.Error, errors.Cause(err))
			}
		})
	}
}
