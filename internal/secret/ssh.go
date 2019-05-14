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

package secret

import (
	"github.com/banzaicloud/pipeline/secret"
)

// GetOrCreateSSHKeyPair gets or creates a SSH key pair in the secret store for the cluster.
// It returns the SSH key pair and its secret ID in the secret store or an error.
func GetOrCreateSSHKeyPair(secrets interface {
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
	Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)
}, cluster interface {
	GetID() uint
	GetName() string
	GetOrganizationID() uint
	GetSSHSecretID() string
	GetUID() string
}) (*secret.SSHKeyPair, string, error) {

	sshSecretID := cluster.GetSSHSecretID()
	if sshSecretID == "" {
		return CreateSSHKeyPair(secrets, cluster.GetOrganizationID(), cluster.GetID(), cluster.GetName(), cluster.GetUID())
	}
	sshKeyPair, err := GetSSHKeyPair(secrets, cluster.GetOrganizationID(), sshSecretID)
	return sshKeyPair, sshSecretID, err
}

// GetSSHKeyPair return the SSH key pair stored in the secret store under the specified organization and secret ID
func GetSSHKeyPair(secrets interface {
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
}, organizationID uint, sshSecretID string) (*secret.SSHKeyPair, error) {

	sir, err := secrets.Get(organizationID, sshSecretID)
	if err != nil {
		return nil, err
	}
	return secret.NewSSHKeyPair(sir), nil
}

// CreateSSHKeyPair creates and stores a new SSH key pair for a cluster in the secret store with the specified parameters
func CreateSSHKeyPair(secrets interface {
	Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)
}, organizationID uint, clusterID uint, clusterName string, clusterUID string) (sshKeyPair *secret.SSHKeyPair, sshSecretID string, err error) {

	sshKeyPair, err = secret.GenerateSSHKeyPair()
	if err != nil {
		return
	}
	sshSecretID, err = secret.StoreSSHKeyPair(sshKeyPair, organizationID, clusterID, clusterName, clusterUID) // TODO: refactor StoreSSHKeyPair to use secrets
	return
}
