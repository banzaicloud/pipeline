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

	corev1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/workflow"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	pkgPKE "github.com/banzaicloud/pipeline/pkg/cluster/pke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

type Logger = common.Logger

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
	LoadBalancerIPRange         string
}

func (f nodeTemplateFactory) getNode(np pke.NodePool, number int) workflow.Node {
	node := workflow.Node{
		Name:                   pke.GetVMName(f.ClusterName, np.Name, number),
		VCPU:                   np.VCPU,
		RAM:                    np.RAM,
		SSHPublicKey:           f.SSHPublicKey,
		AdminUsername:          np.AdminUsername,
		UserDataScriptTemplate: workerUserDataScriptTemplate,
		TemplateName:           np.TemplateName,
		NodePoolName:           np.Name,
		Master:                 np.HasRole(pkgPKE.RoleMaster),
	}

	k8sMasterMode := "default"
	taints := ""

	if np.HasRole(pkgPKE.RoleMaster) {
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

		if np.Size > 1 {
			k8sMasterMode = "ha"
		}
	}

	if np.HasRole(pkgPKE.RolePipelineSystem) {
		if !f.SingleNodePool {
			taints = fmt.Sprintf("%s=%s:%s", pkgCommon.NodePoolNameTaintKey, np.Name, corev1.TaintEffectPreferNoSchedule)
		}
	}

	// HttpProxy settings will be set in workflow
	node.UserDataScriptParams = map[string]string{
		"ClusterID":            strconv.FormatUint(uint64(f.ClusterID), 10),
		"ClusterName":          f.ClusterName,
		"NodePoolName":         np.Name,
		"Taints":               taints,
		"OrgID":                strconv.FormatUint(uint64(f.OrganizationID), 10),
		"PipelineURL":          f.PipelineExternalURL,
		"PipelineURLInsecure":  strconv.FormatBool(f.PipelineExternalURLInsecure),
		"PipelineToken":        "<not yet set>",
		"PKEVersion":           pkeVersion,
		"KubernetesVersion":    f.KubernetesVersion,
		"KubernetesMasterMode": k8sMasterMode,
		"NoProxy":              f.NoProxy,
		"LoadBalancerIPRange":  f.LoadBalancerIPRange,
	}
	return node
}

func handleClusterError(logger Logger, store pke.ClusterStore, status string, clusterID uint, err error) error {
	if clusterID != 0 && err != nil {
		if err := store.SetStatus(clusterID, status, pkgCadence.UnwrapError(err).Error()); err != nil {
			logger.Error("failed to set cluster error status: " + err.Error())
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
