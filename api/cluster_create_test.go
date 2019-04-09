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

package api

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	clusterAPI "github.com/banzaicloud/pipeline/api/cluster"
)

func TestDecodeRequest(t *testing.T) {
	t.Run("decode PKE-on-Azure request", func(t *testing.T) {
		input := obj{
			"name":          "bnz-azure-pke-0",
			"location":      "eu-central-1",
			"type":          "pke-on-azure",
			"secretId":      "abcdefghijklmnopqrstuvwxyz0123456789",
			"resourceGroup": "testRG",
			"nodepools": arr{
				obj{
					"name": "distrMaster",
					"roles": arr{
						"master",
						"pipeline-system",
					},
					"subnet": obj{
						"name": "subnetA",
						"cidr": "10.10.10.0/24",
					},
					"zones":        arr{"1", "2", "3"},
					"instanceType": "Standard_B2s",
				},
				obj{
					"name": "distrWorker",
					"roles": arr{
						"worker",
					},
					"subnet": obj{
						"name": "subnetB",
						"cidr": "10.10.20.0/24",
					},
					"zones":        arr{"1", "3"},
					"instanceType": "Standard_B2s",
				},
			},
			"kubernetes": obj{
				"version": "v1.12.2",
				"rbac":    true,
				"network": obj{
					"serviceCIDR": "10.32.0.0/24",
					"podCIDR":     "10.200.0.0/16",
					"provider":    "weave",
					"providerConfig": obj{
						"key1": "value1",
						"key2": 42,
					},
				},
				"cri": obj{
					"runtime": "containerd",
					"runtimeConfig": obj{
						"key1": "value1",
						"key2": 42,
					},
				},
			},
			"network": obj{
				"name": "vnet",
				"cidr": "10.0.0.0/16",
			},
		}
		var result clusterAPI.CreatePKEOnAzureClusterRequest
		if err := decodeRequest(input, &result); err != nil {
			t.Error(err)
		}
		assert.Equal(t, get(input, "name"), result.Name)
		assert.Equal(t, get(input, "location"), result.Location)
		assert.Equal(t, get(input, "type"), result.Type)
		assert.Equal(t, get(input, "secretId"), result.SecretID)
		assert.Equal(t, get(input, "resourceGroup"), result.ResourceGroup)
		assert.Equal(t, get(get(input, "network"), "name"), result.Network.Name)
		assert.Equal(t, get(get(input, "network"), "cidr"), result.Network.CIDR)
		assert.Equal(t, get(get(input, "kubernetes"), "version"), result.Kubernetes.Version)
		assert.Equal(t, get(get(input, "kubernetes"), "rbac"), result.Kubernetes.RBAC)
		assert.Equal(t, get(get(get(input, "kubernetes"), "network"), "serviceCIDR"), result.Kubernetes.Network.ServiceCIDR)
		assert.Equal(t, get(get(get(input, "kubernetes"), "network"), "podCIDR"), result.Kubernetes.Network.PodCIDR)
		assert.Equal(t, get(get(get(input, "kubernetes"), "network"), "provider"), result.Kubernetes.Network.Provider)
		assert.Equal(t, get(get(get(input, "kubernetes"), "network"), "providerConfig"), result.Kubernetes.Network.ProviderConfig)
		assert.Equal(t, get(get(get(input, "kubernetes"), "cri"), "runtime"), result.Kubernetes.CRI.Runtime)
		assert.Equal(t, get(get(get(input, "kubernetes"), "cri"), "runtimeConfig"), result.Kubernetes.CRI.RuntimeConfig)
		for i, np := range get(input, "nodepools").(arr) {
			assert.Equal(t, get(np, "name"), result.NodePools[i].Name)
			equalLists(t, get(np, "roles"), result.NodePools[i].Roles)
			assert.Equal(t, get(get(np, "subnet"), "name"), result.NodePools[i].Subnet.Name)
			assert.Equal(t, get(get(np, "subnet"), "cidr"), result.NodePools[i].Subnet.CIDR)
			assert.Equal(t, get(np, "instanceType"), result.NodePools[i].InstanceType)
			equalLists(t, get(np, "zones"), result.NodePools[i].Zones)
		}
	})
}

type obj = map[string]interface{}
type arr = []interface{}

func get(o interface{}, key string) interface{} {
	m := o.(map[string]interface{})
	return m[key]
}

func equalLists(t *testing.T, expected interface{}, actual interface{}) {
	expectedVal := reflect.ValueOf(expected)
	actualVal := reflect.ValueOf(actual)
	if k := expectedVal.Kind(); k != reflect.Array && k != reflect.Slice {
		t.Errorf("expected value is not a list but a %v (%T)", k, expected)
	}
	if k := actualVal.Kind(); k != reflect.Array && k != reflect.Slice {
		t.Errorf("actual value is not a list but a %v (%T)", k, actual)
	}

	l := expectedVal.Len()
	if al := actualVal.Len(); al != l {
		t.Errorf("expected and actual lists are of different length: %d <> %d", l, al)
	}
	for i := 0; i < l; i++ {
		assert.EqualValues(t, expectedVal.Index(i).Interface(), actualVal.Index(i).Interface())
	}
}
