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

package workflow

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"
)

func TestGetCreateOrUpdateSecurityGroupParams(t *testing.T) {
	t.Run("typical input", func(t *testing.T) {
		input := CreateNSGActivityInput{
			OrganizationID:    1,
			SecretID:          "0123456789abcdefghijklmnopqrstuvwxyz",
			ClusterName:       "test-cluster",
			ResourceGroupName: "test-rg",
			SecurityGroup: SecurityGroup{
				Location: "test-location",
				Name:     "test-nsg",
				Rules: []SecurityRule{
					{
						Access:               "Allow",
						Description:          "Test security rule 1",
						Destination:          "test-destination",
						DestinationPortRange: "test-dest-port-range",
						Direction:            "Inbound",
						Name:                 "test-security-rule-1",
						Priority:             42,
						Protocol:             "Tcp",
						Source:               "test-source",
						SourcePortRange:      "test-src-port-range",
					},
					{
						Access:               "Deny",
						Description:          "Test security rule 2",
						Destination:          "test-destination",
						DestinationPortRange: "test-dest-port-range",
						Direction:            "Outbound",
						Name:                 "test-security-rule-2",
						Priority:             21,
						Protocol:             "Udp",
						Source:               "test-source",
						SourcePortRange:      "test-src-port-range",
					},
				},
			},
		}
		expected := network.SecurityGroup{
			Location: to.StringPtr("test-location"),
			SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
				SecurityRules: &[]network.SecurityRule{
					{
						Name: to.StringPtr("test-security-rule-1"),
						SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
							Access:                   network.SecurityRuleAccessAllow,
							Description:              to.StringPtr("Test security rule 1"),
							DestinationAddressPrefix: to.StringPtr("test-destination"),
							DestinationPortRange:     to.StringPtr("test-dest-port-range"),
							Direction:                network.SecurityRuleDirectionInbound,
							Priority:                 to.Int32Ptr(42),
							Protocol:                 network.SecurityRuleProtocolTCP,
							SourceAddressPrefix:      to.StringPtr("test-source"),
							SourcePortRange:          to.StringPtr("test-src-port-range"),
						},
					},
					{
						Name: to.StringPtr("test-security-rule-2"),
						SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
							Access:                   network.SecurityRuleAccessDeny,
							Description:              to.StringPtr("Test security rule 2"),
							DestinationAddressPrefix: to.StringPtr("test-destination"),
							DestinationPortRange:     to.StringPtr("test-dest-port-range"),
							Direction:                network.SecurityRuleDirectionOutbound,
							Priority:                 to.Int32Ptr(21),
							Protocol:                 network.SecurityRuleProtocolUDP,
							SourceAddressPrefix:      to.StringPtr("test-source"),
							SourcePortRange:          to.StringPtr("test-src-port-range"),
						},
					},
				},
			},
			Tags: map[string]*string{
				"kubernetesCluster-test-cluster": to.StringPtr("owned"),
				"io.banzaicloud.pipeline.uuid":   to.StringPtr(""),
			},
		}
		result := input.getCreateOrUpdateSecurityGroupParams()
		assert.Equal(t, expected, result)
	})
}
