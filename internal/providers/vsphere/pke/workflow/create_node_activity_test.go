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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/vim25/types"
)

func TestCreateNodeActivity_GenerateVMConfig(t *testing.T) {
	input := CreateNodeActivityInput{
		OrganizationID:   2,
		ClusterID:        269,
		SecretID:         "592cc302663c0021ffa92186bf3c1a579a97e5e01f8b28f766855e5b121f8bda",
		ClusterName:      "vmware-test-638",
		ResourcePoolName: "resource-pool",
		FolderName:       "test",
		DatastoreName:    "DatastoreCluster",
		Node: Node{
			AdminUsername: "",
			VCPU:          2,
			RamMB:         1024,
			Name:          "vmware-sancyx-638-pool1-01",
			SSHPublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcbjmOHfkmyHbzrDFYTQRTzFtBuJcx9r80j2N1EbPrf5XmGShlxiToFxuNCJRahjoqa3I03HFOS10KX4rRQvi6X9utOST5Kwbt4JJ1y7LSgLMom/UBU3fvvjwv/Sb2Q9uLvFA/VNZTKiTW/2iGwl5QcY3+PNedWY6ZobW/t5xU8SNOhusvrBWcXnGeiwV96KdyhLD5QKtO/UO50zfXom8ZkWf7CBKdJBgMJHMwrvDpWJiX25cCIxCHHTmPs4x5CrkfKtrFQGDqL1Y837nd5SRJCygzd8xxIcO6R+0A2NvFEDxO4qXJXbfzbnsFLpteiglidLoYny7s93YjBq59oJEN no-reply@banzaicloud.com \n",
			UserDataScriptParams: map[string]string{
				"ClusterID":            "269",
				"ClusterName":          "vmware-sancyx-638",
				"KubernetesMasterMode": "default",
				"KubernetesVersion":    "1.15.3",
				"NodePoolName":         "pool1",
				"OrgID":                "2",
				"PKEVersion":           "0.4.14",
				"PipelineToken":        "###token###",
				"PipelineURL":          "https://43aa900f.ngrok.io/pipeline",
				"PipelineURLInsecure":  "false",
				"PublicAddress":        "192.168.22.29",
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

	assert.Equal(t, vmSpec, &types.VirtualMachineConfigSpec{
		ExtraConfig: []types.BaseOptionValue{
			&types.OptionValue{Key: "disk.enableUUID", Value: "true"},
			&types.OptionValue{Key: "guestinfo.userdata.encoding", Value: "gzip+base64"},
			&types.OptionValue{Key: "guestinfo.userdata", Value: "H4sIAAAAAAAA/3yUW2/iOBTH3/MpvGEeZjQyIQmhwG5WG6CUS0q4pAWqkapcHOImsVPbCQHNh1+FtppRW80DyMf/3zk68fnbjSClRQgDSiJ8kKLnkPRBmR09hiD3SHCqYEfvwpzSVIUtVYopF8TL0B+hnCGOWIkef9GRl3IksYIEWdiXIPgJJQAafyk+JgqP6zWqcsoEmLju8nG5dnZ7U/6HUFB6aYH+ld8Rmz8jC+czXQKgIAKnIChYCmAJYiFy3lcU3yNnD18OohnQTAnpkaTUC7mSJ6j+wVaz3VTbAFKgFJwpKQ289NJ7nqC/QUgBTxHKgdqqA4IkAII4oyH4Xn2SIAHw2ufScifml/q//w6rm12up/eWe/04XZpfvr4dJYBT8BMEhQAwlIEMYATUbzWdJwhgwoWXpuBIWYIYgDDHOUoxQbBgqSm/fW9b97xeqxU1yYHRpImp8sbJ4IcEfs/DhKOgYMiULxP8oAuaIGLKjUbjsmo0Gh8Qyg4Qh6b2fj9ICy4Qu2id3nuV0BDVhjIvrnpVhYeJ4OZrlBQ+YgQJxOGLiXNGSxwiZpY8jxFDHzkvx/BiTWaqPa2pdrpNTWtqvX6n3dY/4phEzOOCFYEoGIIBDpn55ddYFF37mFMixjElptpUjeYnNXMaQoJEPaKXgrIsFRwxXl+Ll8vymx8lADiPoVeImDJ8RiFM0In366JArhXGPWBZljXQF2dvqJ4C7boOR9bKGtTb1moY+E+ZM4mS7DTxz2w03rurtXsei0ExC6oe67aetIV67S9ZZOyym02cVtil46pYDGdrL36iz54+bemTsbNRW/Ndm61XJe7seoVwNq4xP/qiPZuppyt7c7BvaabcDe70qCyfjqWy8bVVr7DLsaXcLx7cOXa3ioZvjqmxCvb69+UChdt954H6W0UY1V13s3DigpdssA125Abh432vMw9PsT0yVnPhKHeO0TpHO5p1H5JtdDUczMPZ4HA7m9weWTnKtzO804xgOK2Gk4mbLXm7MoYsieaCjVc3o2db3Xf1KxIam/VseDqcw25VTQOns/7esrRFOb4eVU77eTfb+dHZJ3xs5wLhQ4pDm+7J6Yr39P3T4Nno0dn1AhAKGcrT03/vXg/wg9RPES9C2geWbZtfLdv+BhbO0tpstqO+ZdvS/wEAAP//yL9/FXUFAAA="},
			&types.OptionValue{Key: "guestinfo.banzaicloud-pipeline-managed", Value: "true"},
			&types.OptionValue{Key: "guestinfo.banzaicloud-cluster", Value: "vmware-sancyx-638-pool1-01"},
			&types.OptionValue{Key: "guestinfo.banzaicloud-nodepool", Value: "pool1"},
		},
	})
}
