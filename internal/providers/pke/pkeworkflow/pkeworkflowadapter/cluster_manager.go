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

package pkeworkflowadapter

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/pkg/errors"
)

// ClusterManagerAdapter provides an adapter for pkeworkflow.Clusters.
type ClusterManagerAdapter struct {
	clusterManager *cluster.Manager
}

// NewClusterManagerAdapter creates a new ClusterManagerAdapter.
func NewClusterManagerAdapter(clusterManager *cluster.Manager) *ClusterManagerAdapter {
	return &ClusterManagerAdapter{
		clusterManager: clusterManager,
	}
}

// GetCluster returns a Cluster.
func (a *ClusterManagerAdapter) GetCluster(ctx context.Context, id uint) (pkeworkflow.Cluster, error) {
	cluster, err := a.clusterManager.GetClusterByIDOnly(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Cluster{cluster}, nil
}

type Cluster struct {
	cluster.CommonCluster
}

var _ pkeworkflow.AWSCluster = (*Cluster)(nil)

func (c *Cluster) GetID() uint {
	return uint(c.CommonCluster.GetID())
}

func (c *Cluster) GetOrganizationId() uint {
	return uint(c.CommonCluster.GetOrganizationId())
}

func (c *Cluster) GetNodePools() []pkeworkflow.NodePool {
	clusterNodePools := c.CommonCluster.(interface{ GetNodePools() []cluster.PKENodePool }).GetNodePools()
	nodePools := make([]pkeworkflow.NodePool, len(clusterNodePools), len(clusterNodePools))
	for i, np := range clusterNodePools {
		nodePools[i] = pkeworkflow.NodePool{
			Name:              np.Name,
			MinCount:          np.MinCount,
			MaxCount:          np.MaxCount,
			Count:             np.Count,
			Autoscaling:       np.Autoscaling,
			Master:            np.Master,
			Worker:            np.Worker,
			InstanceType:      np.InstanceType,
			AvailabilityZones: np.AvailabilityZones,
			ImageID:           np.ImageID,
			SpotPrice:         np.SpotPrice,
		}
	}
	return nodePools
}

func (c *Cluster) GetAWSClient() (*session.Session, error) {
	if awscluster, ok := c.CommonCluster.(pkeworkflow.AWSCluster); ok {
		return awscluster.GetAWSClient()
	}
	return nil, errors.New(fmt.Sprintf("failed to cast cluster to AWSCluster, got type: %T", c.CommonCluster))
}

func (c *Cluster) GetBootstrapCommand(nodePoolName, url, token string) (string, error) {
	if awscluster, ok := c.CommonCluster.(pkeworkflow.AWSCluster); ok {
		return awscluster.GetBootstrapCommand(nodePoolName, url, token)
	}
	return "", errors.New(fmt.Sprintf("failed to cast cluster to AWSCluster, got type: %T", c.CommonCluster))
}

func (c *Cluster) SaveNetworkCloudProvider(cloudProvider, vpcID string, subnets []string) error {
	if awscluster, ok := c.CommonCluster.(pkeworkflow.AWSCluster); ok {
		return awscluster.SaveNetworkCloudProvider(cloudProvider, vpcID, subnets)
	}
	return errors.New(fmt.Sprintf("failed to cast cluster to AWSCluster, got type: %T", c.CommonCluster))
}

func (c *Cluster) SaveNetworkApiServerAddress(host, port string) error {
	if awscluster, ok := c.CommonCluster.(pkeworkflow.AWSCluster); ok {
		return awscluster.SaveNetworkApiServerAddress(host, port)
	}
	return errors.New(fmt.Sprintf("failed to cast cluster to AWSCluster, got type: %T", c.CommonCluster))
}

func (c *Cluster) GetSshPublicKey() (string, error) {
	if pke, ok := c.CommonCluster.(*cluster.EC2ClusterPKE); ok {
		return pke.GetSshPublicKey()
	}
	return "", errors.New(fmt.Sprintf("failed to cast cluster to EC2ClusterPKE, got type: %T", c.CommonCluster))
}
