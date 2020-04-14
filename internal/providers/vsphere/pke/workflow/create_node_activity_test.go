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
				"KubernetesVersion":    "1.15.3",
				"NodePoolName":         "pool1",
				"OrgID":                "2",
				"PKEVersion":           "0.4.24",
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
		"guestinfo.userdata":                     "H4sIAAAAAAAA/3yT3U7jPBCGz3MV8yUcgD45IW3p0uxmteFnBauoFKh2FwkJufGUmrq28U+hFRe/cgGBWsRBK4/fZ0YTzztJI5RnpFFyzG+j8T2TBcxnD9QgcWgd6bb3iVZK5GQ3jybKOkln+AmiDVo0c7x5Y8dUWIyMl82MFRGBJxIBJP9lIy4zOwlnfNTKODgZDgc3g4uzv1dl/E0qmFPh8Xu8Rlx+jvTPPtIjAC8dF9B4I4DMYeKctkWWjahcUr56hLRRs4ypBykUZTbTUww/spt20lYHiILMW5MJ1VCx6l1P8SswBVYgash3QyAxAmgmM8Xg/8cPEiKAlz4H1fCk3Ar/xRoWmh1cnP6uhsc3p4Nya/v1KYGcwhM03gFhMcRAxpDvBFpPEbi0jgoBD8pM0QAhmmsUXCLxRpTx6/fio0MjqagYM2ht9krFcB3B+ywuLTbeYBmv5rehOzVFWcZJkqxOSZJsIMrcEs7K1vp9I7x1aFZat7euSsUw2KlceepFdZRLZ8uXaOpHaCQ6tOTZvtqoOWdoyrnVEzS4yVHNycqYpsx7rTTv7qftdtrqFd1Op72Jczk21DrjG+cNkoYzU269DSVrtzZz5mgsV7LM03wv/aCmVoxIdGFAzwXjOPIWjQ1L8bwq79wYAVg7IdS7iTJ8iYxMcWGLUBTioBhLoaqq6qDdX9LDfNG0jkN4VJ1XB+G6Oj9sRnfLkbQ/a+2Q3wrOanUlF19sr311d3C/11O/jvsgFTGoxeLH2irAtQx7ZT1TBVR1XW5Xdb0D/bNBdXn556io6jr6FwAA///eHD8HPgQAAA==",
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
			t.Errorf("expected config value for key %s: %s\n actual value: %s ", key, value, expectedValue)
		}
	}
}
