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

	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
)

const PKEOnVsphere = "pke-on-vsphere"

type NodePool struct {
	CreatedBy uint
	Count     int
	VCPU      int
	RamMB     int
	Name      string
	Roles     []string
}

func (np NodePool) InstanceType() string {
	return fmt.Sprintf("%dvcpu-%dmb", np.VCPU, np.RamMB)
}

type PKEOnVsphereCluster struct {
	intCluster.ClusterBase

	NodePools        []NodePool
	ResourcePool     string
	Datastore        string
	Folder           string
	Kubernetes       intPKE.Kubernetes
	ActiveWorkflowID string
	HTTPProxy        intPKE.HTTPProxy

	Monitoring   bool
	Logging      bool
	SecurityScan bool
	TtlMinutes   uint
}

func (c PKEOnVsphereCluster) HasActiveWorkflow() bool {
	return c.ActiveWorkflowID != ""
}

func GetVMName(clusterName, nodePoolName string, number int) string {
	return fmt.Sprintf("%s-%s-%02d", clusterName, nodePoolName, number)
}
