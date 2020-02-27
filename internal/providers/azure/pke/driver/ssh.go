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
	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/secret/ssh"
	"github.com/banzaicloud/pipeline/internal/secret/sshsecret"
	"github.com/banzaicloud/pipeline/src/secret"
)

// GetOrCreateSSHKeyPair creates and saves a new SSH key pair for the cluster or gets the cluster's SSH key pair if it already exists
func GetOrCreateSSHKeyPair(cluster pke.Cluster, secrets secretStore, store pke.ClusterStore) (ssh.KeyPair, error) {
	keyPair, secretID, err := sshsecret.GetOrCreateSSHKeyPair(secrets, getOrCreateSSHKeyPairClusterAdapter(cluster))
	if err != nil {
		return keyPair, errors.WrapIf(err, "failed to get or create SSH key pair")
	}
	if secretID != cluster.SSHSecretID {
		if err := store.SetSSHSecretID(cluster.ID, secretID); err != nil {
			return keyPair, errors.WrapIf(err, "failed to set cluster SSH secret ID")
		}
	}
	return keyPair, nil
}

type secretStore interface {
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
	Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)
}

type getOrCreateSSHKeyPairClusterAdapter pke.Cluster

func (a getOrCreateSSHKeyPairClusterAdapter) GetID() uint {
	return a.ID
}

func (a getOrCreateSSHKeyPairClusterAdapter) GetName() string {
	return a.Name
}

func (a getOrCreateSSHKeyPairClusterAdapter) GetOrganizationID() uint {
	return a.OrganizationID
}

func (a getOrCreateSSHKeyPairClusterAdapter) GetSSHSecretID() string {
	return a.SSHSecretID
}

func (a getOrCreateSSHKeyPairClusterAdapter) GetUID() string {
	return a.UID
}
