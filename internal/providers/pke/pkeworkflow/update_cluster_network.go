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
	"fmt"
	"strings"

	"github.com/pkg/errors"

	internalPke "github.com/banzaicloud/pipeline/internal/providers/pke"
)

const UpdateClusterNetworkActivityName = "pke-update-cluster-network-activity"

type UpdateClusterNetworkActivity struct {
	clusters Clusters
}

func NewUpdateClusterNetworkActivity(clusters Clusters) *UpdateClusterNetworkActivity {
	return &UpdateClusterNetworkActivity{
		clusters: clusters,
	}
}

type UpdateClusterNetworkActivityInput struct {
	ClusterID       uint
	APISeverAddress string
	VPCID           string
	Subnets         string
}

func (a *UpdateClusterNetworkActivity) Execute(ctx context.Context, input UpdateClusterNetworkActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return errors.New(fmt.Sprintf("can't update Network for cluster type %t", c))
	}

	subnets := strings.Split(input.Subnets, ",")
	err = awsCluster.SaveNetworkCloudProvider(string(internalPke.CNPAmazon), input.VPCID, subnets)
	if err != nil {
		return err
	}

	return awsCluster.SaveNetworkApiServerAddress(input.APISeverAddress, "")
}
