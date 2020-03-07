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

package workflow

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
)

const SaveNetworkDetailsActivityName = "eks-save-network-details"

type SaveNetworkDetailsActivity struct {
	manager Clusters
}

func NewSaveNetworkDetailsActivity(manager Clusters) SaveNetworkDetailsActivity {
	return SaveNetworkDetailsActivity{
		manager: manager,
	}
}

type SaveNetworkDetailsInput struct {
	ClusterID uint

	VpcID              string
	NodeInstanceRoleID string
	Subnets            []Subnet
}

func (a SaveNetworkDetailsActivity) Execute(ctx context.Context, input SaveNetworkDetailsInput) error {
	cluster, err := a.manager.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	if eksCluster, ok := cluster.(interface {
		GetModel() *eksmodel.EKSClusterModel
	}); ok {
		modelCluster := eksCluster.GetModel()
		modelCluster.NodeInstanceRoleId = input.NodeInstanceRoleID
		modelCluster.VpcId = aws.String(input.VpcID)

		// persist the id of the newly created subnets
		for _, subnet := range input.Subnets {
			for _, subnetModel := range modelCluster.Subnets {
				if (aws.StringValue(subnetModel.SubnetId) != "" && aws.StringValue(subnetModel.SubnetId) == subnet.SubnetID) ||
					(aws.StringValue(subnetModel.SubnetId) == "" && aws.StringValue(subnetModel.Cidr) != "" && aws.StringValue(subnetModel.Cidr) == subnet.Cidr) {
					sub := subnet
					subnetModel.SubnetId = &sub.SubnetID
					subnetModel.Cidr = &sub.Cidr
					subnetModel.AvailabilityZone = &sub.AvailabilityZone
					break
				}
			}
		}
	}

	return cluster.Persist()
}
