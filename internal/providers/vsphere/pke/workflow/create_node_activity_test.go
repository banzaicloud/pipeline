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
				"KubernetesVersion":    "1.21.14",
				"NodePoolName":         "pool1",
				"OrgID":                "2",
				"PKEVersion":           "0.9.4-dev.1",
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
		"guestinfo.userdata":                     "H4sIAAAAAAAA/3yTXU/jOhCG7/Mr5iRcgI6ckIbTQ7Ob1YaPFayiUqDaXSQk5MZTauraxh+FVvz4lQsI1CIuEnn8PmONPe8krVCekVbJMb+NxvdMljCfPVCDxKF1pFvsE62UyMluHk2UdZLO8BNEG7Ro5njzxo6psBgZL9sZKyMCTyQCSP7JRlxmdhLW+KiVcXAyHA5uBhdnf66q+KtUMKfC47d4jbj8HOmffaRHAF46LqD1RgCZw8Q5bcssG1G5pHz1CGmrZhlTD1Ioymympxg+spv20gKIgsxbkwnVUrEqXU/xCzAFViBqyHdDIDECaCczxeDfxw8SIoCXMgf18KTaCv9yDQu1Di5Of9XD45vTQbW1/fqSQE7hCVrvgLAYYiBjyHcCracIXFpHhYAHZaZogBDNNQoukXgjqvj1uvjo0EgqasYMWpu9UjFcR/A+i0uLrTdYxav2behOTVFWcZIkq1WSJBuIMreEs6qzvt8Kbx2aldbtratSMQxuqlaWelEd5dLZ6iWa+hEaiQ4teXavNmrOGZpqbvUEDW5yVHOy8qWp8l4nzbv7aVGknV7Z3dsrNnEux4ZaZ3zrvEHScmaqrbemZEVnM2eOxnIlqzzt5Gm+twloxYhEFzr0fGIcR96isWEonkflnRsjAGsnhHo3UYYvkZEpLmwZDoU4KMZSqOu6Pij6S3qYL9rOcQiP6vP6IGzX54ft6G45kvZHox3yW8FZo67k4n/bK67uDu7/66mfx32QihjUYvF9bRQiAIBrGWbLeqZKqJum2q6bZgf6Z4P68vL3UVk3TfQ3AAD//0a1BFpCBAAA",
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
