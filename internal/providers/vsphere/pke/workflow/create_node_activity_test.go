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

package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateNodeActivity_GenerateVMConfig(t *testing.T) {
	input := CreateNodeActivityInput{
		OrganizationID:   2,
		ClusterID:        269,
		SecretID:         "592cc302663c0755e5b121f8bda",
		ClusterName:      "vmware-test-638",
		ResourcePoolName: "resource-pool",
		FolderName:       "test",
		DatastoreName:    "DatastoreCluster",
		Node: Node{
			AdminUsername: "",
			VCPU:          2,
			RAM:           1024,
			Name:          "vmware-test-638-pool1-01",
			SSHPublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcbjzbnsFLpteiglidLoYny7s93YjBq59oJEN no-reply@banzaicloud.com \n",
			UserDataScriptParams: map[string]string{
				"ClusterID":            "269",
				"ClusterName":          "vmware-test-638",
				"KubernetesMasterMode": "default",
				"KubernetesVersion":    "1.15.12",
				"NodePoolName":         "pool1",
				"OrgID":                "2",
				"PKEVersion":           "0.5.1",
				"PipelineToken":        "###token###",
				"PipelineURL":          "https://externalAddress/pipeline",
				"PipelineURLInsecure":  "false",
				"PublicAddress":        "192.168.33.29",
				"Taints":               "",
			},
			UserDataScriptTemplate: "#!/bin/sh\n#export HTTP_PROXY=\"{{ .HttpProxy }}\"\n#export HTTPS_PROXY=\"{{ .HttpsProxy }}\"\n#export NO_PROXY=\"{{ .NoProxy }}\"\n\nuntil curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke; do sleep 10; done\nchmod +x /usr/local/bin/pke\nexport PATH=$PATH:/usr/local/bin/\n\nPRIVATE_IP=$(hostname -I | cut -d\" \" -f 1)\n\npke install worker --pipeline-url=\"{{ .PipelineURL }}\" \\\n--pipeline-insecure=\"{{ .PipelineURLInsecure }}\" \\\n--pipeline-token=\"{{ .PipelineToken }}\" \\\n--pipeline-org-id={{ .OrgID }} \\\n--pipeline-cluster-id={{ .ClusterID}} \\\n--pipeline-nodepool={{ .NodePoolName }} \\\n--taints={{ .Taints }} \\\n--kubernetes-cloud-provider=vsphere \\\n--kubernetes-api-server={{ .PublicAddress }}:6443 \\\n--kubernetes-infrastructure-cidr=$PRIVATE_IP/32 \\\n--kubernetes-version={{ .KubernetesVersion }} \\\n--kubernetes-pod-network-cidr=\"\"",
			TemplateName:           "centos-7-pke-202001171452",
			NodePoolName:           "pool1",
			Master:                 false,
		},
	}

	vmSpec, err := generateVMConfigs(input)
	require.NoError(t, err)

	expectedConfig := map[string]string{
		"disk.enableUUID":                        "true",
		"guestinfo.userdata.encoding":            "gzip+base64",
		"guestinfo.userdata":                     "H4sIAAAAAAAA/3yTb0/7NhDHn+dV3BIe/NDkhLTQ0WyZFv5MMEWlQLUNCQm58ZWaurY524VWvPifUkCgFvEgkc/fz1ln3/eSRpkgWGP0RN5Hk0ehC1jMnzgh8+g863UPmTVG5Wwvj6bGec3n+A1iCR3SAu8+2AlXDiMKupmLImLwwiKA5JdsLHXmpu0an60hD2ej0fBueHXx/00Z/6ENLLgK+Ge8QVx/jwwuvtIjgKC9VNAEUsAWMPXeuiLLxlyvuFw/QtqYeSbMk1aGC5fZGbYf20sP0hyYgSw4ypRpuFqXbmf4OwgDTiFayPfaQGME0EznRsCvz18kRABvZQ6r0Vm50/6LDaytdXh1/m81Or07H5Y7P95fEtg5vEATPDARQwxsAvluS9sZgtTOc6XgydAMCRiz0qKSGlkgVcbv18Vnj6S5qoQgdC57p2K4jeBzltQOm0BYxuv2benezFCXcZIk61WSJFuIoXsmRdnZ3G9UcB5prfX6m6o2Als3lWtLvameS+1d+RbNwhhJo0fHXt1rySykQCoXzk6RcJvjVrK1L6nM+5007x2m3W7a6Re9/f3uNi71hLjzFBofCFkjBZU7H03Jup3tnAWSk0aXeZofpPkXgDWCafRth15PjOMoOCTXDsXrqHxyYwTg3JTx4KeG5AoFm+HSFe2hELcKOQ5VVVVH3cGKH+fLpnPahifVZXXUbleXx834YTXW7u/aepT3Sora3Ojlb67fvXk4ejzom39OB6ANI7Rq+dfGKMCtbufKBWEKqOq6/FHV9S4MLobV9fV/J0VV19HPAAAA///E4Wn+PgQAAA==",
		"guestinfo.banzaicloud-pipeline-managed": "true",
		"guestinfo.banzaicloud-cluster":          "vmware-test-638-pool1-01",
		"guestinfo.banzaicloud-nodepool":         "pool1",
	}

	if len(expectedConfig) != len(vmSpec.ExtraConfig) {
		t.Errorf("expected config size is %v, actual is %v", len(expectedConfig), len(vmSpec.ExtraConfig))
	}

	for _, config := range vmSpec.ExtraConfig {
		key := config.GetOptionValue().Key
		value := config.GetOptionValue().Value
		expectedValue, ok := expectedConfig[key]
		if !ok {
			t.Errorf("expected config key %s not found", key)
		} else if value != expectedValue {
			t.Errorf("expected config value for key %s: %s\n actual value: %s ", key, expectedValue, value)
		}
	}
}
