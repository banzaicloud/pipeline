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

package adapter

import (
	"os"

	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
)

type dummyVspherePKEClusterStore struct {
	clusters map[uint]*pke.PKEOnVsphereCluster
	id       uint
}

func NewDummyVspherePKEClusterStore() pke.VsphereClusterStore {
	return dummyVspherePKEClusterStore{clusters: make(map[uint]*pke.PKEOnVsphereCluster), id: uint(os.Getpid())}
}

func (s dummyVspherePKEClusterStore) CreateNodePool(clusterID uint, nodePool pke.NodePool) error {
	s.clusters[clusterID].NodePools = append(s.clusters[clusterID].NodePools, nodePool)
	return nil
}

func (s dummyVspherePKEClusterStore) Create(params pke.CreateParams) (c pke.PKEOnVsphereCluster, err error) {
	s.id += 1
	c.ID = s.id
	c.Name = params.Name
	c.OrganizationID = params.OrganizationID
	c.CreatedBy = params.CreatedBy
	c.SecretID = params.SecretID
	c.SSHSecretID = params.SSHSecretID
	c.Kubernetes.RBAC = params.RBAC
	c.Kubernetes.OIDC.Enabled = params.OIDC
	c.Kubernetes.Version = params.KubernetesVersion
	c.ScaleOptions = params.ScaleOptions
	c.NodePools = params.NodePools
	//c.Features = params.Features
	c.HTTPProxy = params.HTTPProxy
	s.clusters[s.id] = &c
	return
}

func (s dummyVspherePKEClusterStore) DeleteNodePool(clusterID uint, nodePoolName string) error {
	return nil
}

func (s dummyVspherePKEClusterStore) Delete(clusterID uint) error {
	return nil
}

func (s dummyVspherePKEClusterStore) GetByID(clusterID uint) (pke.PKEOnVsphereCluster, error) {
	return *s.clusters[clusterID], nil
}

func (s dummyVspherePKEClusterStore) SetStatus(clusterID uint, status, message string) error {
	return nil
}

func (s dummyVspherePKEClusterStore) SetActiveWorkflowID(clusterID uint, workflowID string) error {
	return nil
}

func (s dummyVspherePKEClusterStore) SetConfigSecretID(clusterID uint, secretID string) error {
	return nil
}

func (s dummyVspherePKEClusterStore) SetSSHSecretID(clusterID uint, secretID string) error {
	return nil
}

func (s dummyVspherePKEClusterStore) GetConfigSecretID(clusterID uint) (string, error) {
	return s.clusters[clusterID].K8sSecretID, nil
}

func (s dummyVspherePKEClusterStore) SetFeature(clusterID uint, feature string, state bool) error {
	return nil
}
