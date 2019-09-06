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

package driver

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"emperror.dev/emperror"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	pkgPKE "github.com/banzaicloud/pipeline/pkg/cluster/pke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
)

type nodePoolTemplateFactory struct {
	ClusterID                   uint
	ClusterName                 string
	KubernetesVersion           string
	Location                    string
	OrganizationID              uint
	PipelineExternalURL         string
	PipelineExternalURLInsecure bool
	ResourceGroupName           string
	RouteTableName              string
	SingleNodePool              bool
	SSHPublicKey                string
	TenantID                    string
	VirtualNetworkName          string
	OIDCClientID                string
	OIDCIssuerURL               string
}

func (f nodePoolTemplateFactory) getTemplates(np NodePool) (workflow.VirtualMachineScaleSetTemplate, workflow.SubnetTemplate, []workflow.RoleAssignmentTemplate) {
	var bapn string
	var inpn string
	var taints string

	azureRoleName := "Contributor"

	nsgn := f.ClusterName + "-worker-nsg"

	userDataScriptTemplate := workerUserDataScriptTemplate

	k8sMasterMode := "default"

	if np.hasRole(pkgPKE.RoleMaster) {
		bapn = pke.GetBackendAddressPoolName()
		inpn = pke.GetInboundNATPoolName()

		azureRoleName = "Owner"

		nsgn = f.ClusterName + "-master-nsg"

		if f.SingleNodePool {
			taints = "," // do not taint single node pool cluster's master node
		} else {
			taints = MasterNodeTaint
		}

		userDataScriptTemplate = masterUserDataScriptTemplate

		if f.OIDCIssuerURL != "" {
			userDataScriptTemplate += fmt.Sprintf(` \
--kubernetes-oidc-issuer-url=%q \
--kubernetes-oidc-client-id=%q`,
				f.OIDCIssuerURL,
				f.OIDCClientID,
			)
		}

		if np.Count > 1 {
			k8sMasterMode = "ha"
		}
	}

	if np.hasRole(pkgPKE.RolePipelineSystem) {
		if !f.SingleNodePool {
			taints = fmt.Sprintf("%s=%s:%s", pkgCommon.NodePoolNameTaintKey, np.Name, corev1.TaintEffectPreferNoSchedule)
		}
	}

	vmssName := pke.GetVMSSName(f.ClusterName, np.Name)

	cnsgn := nsgn
	if !f.SingleNodePool {
		// Ingress traffic flow target. In case of multiple NSGs workers can only receive traffic.
		cnsgn = f.ClusterName + "-worker-nsg"
	}

	return workflow.VirtualMachineScaleSetTemplate{
			AdminUsername: "azureuser",
			Image: workflow.Image{
				Offer:     "CentOS-CI",
				Publisher: "OpenLogic",
				SKU:       "7-CI",
				Version:   "7.6.20190306",
			},
			InstanceCount:                uint(np.Count),
			InstanceType:                 np.InstanceType,
			BackendAddressPoolName:       bapn,
			OutputBackendAddressPoolName: pke.GetOutboundBackendAddressPoolName(),
			InboundNATPoolName:           inpn,
			Location:                     f.Location,
			Name:                         vmssName,
			NetworkSecurityGroupName:     nsgn,
			NodePoolName:                 np.Name,
			SSHPublicKey:                 f.SSHPublicKey,
			SubnetName:                   np.Subnet.Name,
			UserDataScriptParams: map[string]string{
				"ClusterID":             strconv.FormatUint(uint64(f.ClusterID), 10),
				"ClusterName":           f.ClusterName,
				"InfraCIDR":             np.Subnet.CIDR,
				"LoadBalancerSKU":       "standard",
				"NodePoolName":          np.Name,
				"Taints":                taints,
				"NSGName":               cnsgn,
				"OrgID":                 strconv.FormatUint(uint64(f.OrganizationID), 10),
				"PipelineURL":           f.PipelineExternalURL,
				"PipelineURLInsecure":   strconv.FormatBool(f.PipelineExternalURLInsecure),
				"PipelineToken":         "<not yet set>",
				"PKEVersion":            pkeVersion,
				"KubernetesVersion":     f.KubernetesVersion,
				"KubernetesMasterMode":  k8sMasterMode,
				"PublicAddress":         "<not yet set>",
				"RouteTableName":        f.RouteTableName,
				"SubnetName":            np.Subnet.Name,
				"TenantID":              f.TenantID,
				"VnetName":              f.VirtualNetworkName,
				"VnetResourceGroupName": f.ResourceGroupName,
			},
			UserDataScriptTemplate: userDataScriptTemplate,
			Zones:                  np.Zones,
		}, workflow.SubnetTemplate{
			Name:           np.Subnet.Name,
			CIDR:           np.Subnet.CIDR,
			RouteTableName: f.RouteTableName,
		}, []workflow.RoleAssignmentTemplate{
			{
				Name:     uuid.Must(uuid.NewV1()).String(),
				VMSSName: vmssName,
				RoleName: azureRoleName,
			},
		}
}

func handleClusterError(logger logrus.FieldLogger, store pke.AzurePKEClusterStore, status string, clusterID uint, err error) error {
	if clusterID != 0 && err != nil {
		if err := store.SetStatus(clusterID, status, err.Error()); err != nil {
			logger.Errorf("failed to set cluster error status: %s", err.Error())
		}
	}
	return err
}

type notExistsYetError struct{}

func (notExistsYetError) Error() string {
	return "this resource does not exist yet"
}

func (notExistsYetError) NotFound() bool {
	return true
}

func getSubnetCIDR(ctx context.Context, client pkgAzure.SubnetsClient, resourceGroupName, virtualNetworkName, subnetName string) (string, error) {
	subnet, err := client.Get(ctx, resourceGroupName, virtualNetworkName, subnetName, "")
	if subnet.StatusCode == http.StatusNotFound {
		return "", notExistsYetError{}
	} else if err != nil {
		return "", emperror.Wrap(err, "failed to get subnet")
	}
	return to.String(subnet.AddressPrefix), nil
}
