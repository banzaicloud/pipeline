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

package pkeworkflow

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
)

type Clusters interface {
	GetCluster(ctx context.Context, id uint) (Cluster, error)
}

type Cluster interface {
	GetID() uint
	GetUID() string
	GetName() string
	GetOrganizationId() uint
	UpdateStatus(string, string) error
	GetNodePools() []NodePool
	GetSshPublicKey() (string, error)
	GetLocation() string
}

type AWSCluster interface {
	GetAWSClient() (*session.Session, error)
	GetBootstrapCommand(string, string, string) (string, error)
	SaveNetworkCloudProvider(string, string, []string) error
	SaveNetworkApiServerAddress(string, string) error
}

type NodePool struct {
	Name              string
	MinCount          int
	MaxCount          int
	Count             int
	Master            bool
	Worker            bool
	InstanceType      string
	AvailabilityZones []string
	ImageID           string
	SpotPrice         string
}
