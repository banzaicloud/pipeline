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

package cluster

import (
	"time"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

type Cluster interface {
	GetID() uint
	GetName() string
}

// ClusterBase defines common cluster fields
type ClusterBase struct {
	CreatedBy      uint
	CreationTime   time.Time
	ID             uint
	K8sSecretID    string
	Name           string
	OrganizationID uint
	ScaleOptions   pkgCluster.ScaleOptions
	SecretID       string
	SSHSecretID    string
	Status         string
	StatusMessage  string
	UID            string
}

func (c ClusterBase) GetID() uint {
	return c.ID
}

func (c ClusterBase) GetName() string {
	return c.Name
}
