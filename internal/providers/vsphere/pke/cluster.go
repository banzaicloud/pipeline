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

package pke

import (
	"fmt"

	"github.com/banzaicloud/pipeline/internal/cluster/clusterbase"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	pkgPKE "github.com/banzaicloud/pipeline/pkg/cluster/pke"
)

const PKEOnVsphere = "pke-on-vsphere"

type NodePool struct {
	CreatedBy     uint
	Size          int
	VCPU          int
	RAM           int // MiB
	Name          string
	Roles         []string
	AdminUsername string
	TemplateName  string
}

func (np NodePool) InstanceType() string {
	return fmt.Sprintf("%dvcpu-%dmb", np.VCPU, np.RAM)
}

func (np NodePool) HasRole(role pkgPKE.Role) bool {
	for _, r := range np.Roles {
		if r == string(role) {
			return true
		}
	}
	return false
}

type PKEOnVsphereCluster struct {
	clusterbase.ClusterBase

	NodePools           []NodePool
	ResourcePool        string
	Datastore           string
	Folder              string
	Kubernetes          intPKE.Kubernetes
	ActiveWorkflowID    string
	HTTPProxy           intPKE.HTTPProxy
	StorageSecretID     string
	LoadBalancerIPRange string
}

func (c PKEOnVsphereCluster) HasActiveWorkflow() bool {
	return c.ActiveWorkflowID != ""
}

func GetVMName(clusterName, nodePoolName string, number int) string {
	return fmt.Sprintf("%s-%s-%02d", clusterName, nodePoolName, number)
}
