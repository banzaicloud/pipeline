// Copyright © 2018 Banzai Cloud
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
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

type clusterEvents interface {
	// ClusterCreated event is emitted when a cluster creation workflow finishes.
	ClusterCreated(clusterID pkgCluster.ClusterID)

	// ClusterDeleted event is emitted when a cluster is completely deleted.
	ClusterDeleted(orgID uint, clusterName string)
}

type nopClusterEvents struct {
}

func NewNopClusterEvents() *nopClusterEvents {
	return &nopClusterEvents{}
}

func (*nopClusterEvents) ClusterCreated(clusterID pkgCluster.ClusterID) {
}

func (*nopClusterEvents) ClusterDeleted(orgID uint, clusterName string) {
}

type eventBus interface {
	Publish(topic string, args ...interface{})
}

type clusterEventBus struct {
	eb eventBus
}

const (
	clusterCreatedTopic = "cluster_created"
	clusterDeletedTopic = "cluster_deleted"
)

func NewClusterEvents(eb eventBus) *clusterEventBus {
	return &clusterEventBus{
		eb: eb,
	}
}

func (c *clusterEventBus) ClusterCreated(clusterID pkgCluster.ClusterID) {
	c.eb.Publish(clusterCreatedTopic, clusterID)
}

func (c *clusterEventBus) ClusterDeleted(orgID uint, clusterName string) {
	c.eb.Publish(clusterDeletedTopic, orgID, clusterName)
}
