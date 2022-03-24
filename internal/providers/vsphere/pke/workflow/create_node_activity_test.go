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
				"PKEVersion":           "0.9.1",
				"PipelineToken":        "###token###",
				"PipelineURL":          "https://externalAddress/pipeline",
				"PipelineURLInsecure":  "false",
				"PublicAddress":        "192.168.33.29",
				"Taints":               "",
			},
			UserDataScriptTemplate: "#!/bin/sh\n#export HTTP_PROXY=\"{{ .HttpProxy }}\"\n#export HTTPS_PROXY=\"{{ .HttpsProxy }}\"\n#export NO_PROXY=\"{{ .NoProxy }}\"\n\nuntil curl -vL https://github.com/banzaicloud/pke/releases/download/{{ .PKEVersion }}/pke-{{ .PKEVersion }} -o /usr/local/bin/pke; do sleep 10; done\nchmod +x /usr/local/bin/pke\nexport PATH=$PATH:/usr/local/bin/\n\nPRIVATE_IP=$(hostname -I | cut -d\" \" -f 1)\n\npke install worker --pipeline-url=\"{{ .PipelineURL }}\" \\\n--pipeline-insecure=\"{{ .PipelineURLInsecure }}\" \\\n--pipeline-token=\"{{ .PipelineToken }}\" \\\n--pipeline-org-id={{ .OrgID }} \\\n--pipeline-cluster-id={{ .ClusterID}} \\\n--pipeline-nodepool={{ .NodePoolName }} \\\n--taints={{ .Taints }} \\\n--kubernetes-cloud-provider=vsphere \\\n--kubernetes-api-server={{ .PublicAddress }}:6443 \\\n--kubernetes-infrastructure-cidr=$PRIVATE_IP/32 \\\n--kubernetes-version={{ .KubernetesVersion }} \\\n--kubernetes-pod-network-cidr=\"\"",
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
		"guestinfo.userdata":                     "H4sIAAAAAAAA/3yT3U7jPBCGz3MV8yUcgD45IS10aXaz2vCzglVUClS7i4SE3HjamLq28U+hFRe/SgCBWsRJNeP3GTeeeSeqhPKMVEpO+DSY3DOZwWL+QA0Sh9aRXveAaKVESnbToFbWSTrHTxBt0KJZ4O0bO6HCYmC8rOYsCwg8kQAg+i8Zc5nYuonxUSvj4HQ0Gt4OL8//XufhN6lgQYXH7+EacfU5Mjj/SA8AvHRcQOWNALIooXZO2yxJptzVfhxXap6MqVxR3jYk0TNMDAqkFm3C1IMUirJkN+7HaaORNgKiIPHWJEJVVLQP0jP8CkyBFYga0t0mkRgAVPVcMfj/8YOCAODl44fF6DTfan6zNax5wfDy7HcxOrk9G+Zb26/9BXIGT1B5B4SFEAKZQLrT0HqGwKV1VAh4UGaGBgjRXKPgEok3Ig9fe4CPDo2komDMoLXJKxXCTQDvq7i0WHmDedgOdUN3aoYyD6MoaqMoijYQZaaEs7yzfl4Jbx2aVuv111WpGDYey1ujvaiOculs/pLN/BiNRIeWPHtaG7XgDE2+sLpGg5sc1Zy0bjV52u/Eae8g7nbjTj/r7e11N3EuJ4ZaZ3zlvEFScWbyrbehJN3OZs0CjeVK5mmc7sfpB4BWjEh0zYSebwzDwFs0tlmV5wV658sAwNqaUO9qZfgKGZnh0mbNpRA2irEUiqIoDruDFT1Kl1XnpEmPi4visDkuLo6q8d1qLO3PUjvkU8FZqa7l8ovtd6/vDu/3++rXyQCkIga1WP549+fNkgQAADey2TjrmcqgKMt8uyjLHRicD4urqz/HWVGWwb8AAAD///a958JYBAAA",
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
