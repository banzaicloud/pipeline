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
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/workflow"
	pkgPKE "github.com/banzaicloud/pipeline/pkg/cluster/pke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

type nodeTemplateFactory struct {
	ClusterID                   uint
	ClusterName                 string
	KubernetesVersion           string
	OrganizationID              uint
	PipelineExternalURL         string
	PipelineExternalURLInsecure bool
	SingleNodePool              bool
	SSHPublicKey                string
	OIDCClientID                string
	OIDCIssuerURL               string
	NoProxy                     string
}

func (f nodeTemplateFactory) getNode(np NodePool, number int) workflow.Node {

	node := workflow.Node{
		Name:                   pke.GetVMName(f.ClusterName, np.Name, number),
		VCPU:                   np.VCPU,
		RamMB:                  np.RamMB,
		SSHPublicKey:           f.SSHPublicKey,
		AdminUsername:          np.AdminUsername,
		UserDataScriptTemplate: workerUserDataScriptTemplate,
		TemplateName:           np.TemplateName,
		NodePoolName:           np.Name,
	}

	k8sMasterMode := "default"
	taints := ""

	if np.hasRole(pkgPKE.RoleMaster) {

		if f.SingleNodePool {
			taints = "," // do not taint single node pool cluster's master node
		} else {
			taints = MasterNodeTaint
		}

		node.UserDataScriptTemplate = masterUserDataScriptTemplate

		// TODO use templating
		if f.OIDCIssuerURL != "" {
			node.UserDataScriptTemplate += fmt.Sprintf(` \
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

	node.UserDataScriptParams = map[string]string{
		"ClusterID":   strconv.FormatUint(uint64(f.ClusterID), 10),
		"ClusterName": f.ClusterName,
		//"InfraCIDR":             np.Subnet.CIDR, TODO find out why would this be needed
		"NodePoolName":         np.Name,
		"Taints":               taints,
		"OrgID":                strconv.FormatUint(uint64(f.OrganizationID), 10),
		"PipelineURL":          f.PipelineExternalURL,
		"PipelineURLInsecure":  strconv.FormatBool(f.PipelineExternalURLInsecure),
		"PipelineToken":        "<not yet set>",
		"PKEVersion":           pkeVersion,
		"KubernetesVersion":    f.KubernetesVersion,
		"KubernetesMasterMode": k8sMasterMode,
		"PublicAddress":        "<not yet set>",
		//"HttpProxy":            "<not yet set>",
		//"HttpsProxy":           "<not yet set>",
		//"NoProxy":              f.NoProxy,
	}
	return node
}

func handleClusterError(logger logrus.FieldLogger, store pke.VsphereClusterStore, status string, clusterID uint, err error) error {
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
