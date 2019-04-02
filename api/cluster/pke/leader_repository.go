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
	"strings"

	"github.com/banzaicloud/bank-vaults/pkg/vault"
	"github.com/goph/emperror"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
)

type leaderSecretData struct {
	Hostname string `json:"hostname" mapstructure:"hostname"`
	IP       string `json:"ip,omitempty" mapstructure:"ip"`
}

type leaderSecretOptions struct {
	CAS uint `json:"cas" mapstructure:"cas"`
}

type leaderSecret struct {
	Data    leaderSecretData    `json:"data" mapstructure:"data"`
	Options leaderSecretOptions `json:"options" mapstructure:"options"`
}

// VaultLeaderRepository implements a LeaderRepository over Vault secret store
type VaultLeaderRepository struct {
	logical *vaultapi.Logical
}

// NewVaultLeaderRepository returns a new VaultLeaderRepository
func NewVaultLeaderRepository() (VaultLeaderRepository, error) {
	role := "pipeline"
	client, err := vault.NewClient(role)
	if err != nil {
		return VaultLeaderRepository{}, emperror.Wrap(err, "failed to create new Vault client")
	}
	return NewVaultLeaderRepositoryFromClient(client), nil
}

// NewVaultLeaderRepositoryFromClient returns a new VaultLeaderRepository
func NewVaultLeaderRepositoryFromClient(client *vault.Client) VaultLeaderRepository {
	return VaultLeaderRepository{
		logical: client.Vault().Logical(),
	}
}

// GetLeader returns information about the leader of the specified cluster
func (r VaultLeaderRepository) GetLeader(organizationID, clusterID uint) (leaderInfo *LeaderInfo, err error) {
	path := getSecretPath(organizationID, clusterID)

	secret, err := r.logical.Read(path)
	if err = emperror.Wrap(err, "failed to read secret"); err != nil {
		return
	}

	if secret == nil {
		// secret not found
		return
	}

	var lsd leaderSecretData
	if err = emperror.Wrap(mapstructure.Decode(secret.Data["data"], &lsd), "failed to decode secret data"); err != nil {
		return
	}

	leaderInfo = &LeaderInfo{
		Hostname: lsd.Hostname,
		IP:       lsd.IP,
	}
	return
}

// SetLeader writes the given leader info for the specified cluster to the repository if it's not set yet
func (r VaultLeaderRepository) SetLeader(organizationID, clusterID uint, leaderInfo LeaderInfo) error {
	path := getSecretPath(organizationID, clusterID)
	ls := leaderSecret{
		Data: leaderSecretData{
			Hostname: leaderInfo.Hostname,
			IP:       leaderInfo.IP,
		},
		Options: leaderSecretOptions{
			CAS: 0, // only allow write if the key doesn't exist
		},
	}

	data := make(map[string]interface{})
	if err := mapstructure.Decode(ls, &data); err != nil {
		return emperror.Wrap(err, "failed to decode leader secret")
	}

	_, err := r.logical.Write(path, data)
	if err != nil && strings.Contains(err.Error(), "* check-and-set parameter did not match the current version") {
		return emperror.Wrap(leaderSetError{}, "failed to write leader secret")
	}
	return err
}

func (r VaultLeaderRepository) DeleteLeader(organizationID, clusterID uint) error {
	path := getMetadataPath(organizationID, clusterID)
	_, err := r.logical.Delete(path)

	return err
}

func getSecretPath(organizationID, clusterID uint) string {
	return fmt.Sprintf("leaderelection/data/orgs/%d/clusters/%d/leader", organizationID, clusterID)
}

func getMetadataPath(organizationID, clusterID uint) string {
	return fmt.Sprintf("leaderelection/metadata/orgs/%d/clusters/%d/leader", organizationID, clusterID)
}

type leaderSetError struct{}

func (leaderSetError) Error() string {
	return "Cluster leader is already set."
}

func (leaderSetError) LeaderSet() bool {
	return true
}
