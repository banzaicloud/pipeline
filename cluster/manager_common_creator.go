package cluster

import (
	"context"

	"github.com/banzaicloud/pipeline/pkg/cluster"
)

type commonCreator struct {
	request *cluster.CreateClusterRequest
	cluster CommonCluster
}

// NewCommonClusterCreator returns a new cluster creator instance.
func NewCommonClusterCreator(request *cluster.CreateClusterRequest, cluster CommonCluster) *commonCreator {
	return &commonCreator{
		request: request,
		cluster: cluster,
	}
}

// Validate implements the clusterCreator interface.
func (c *commonCreator) Validate(ctx context.Context) error {
	return c.cluster.ValidateCreationFields(c.request)
}

// Prepare implements the clusterCreator interface.
func (c *commonCreator) Prepare(ctx context.Context) (CommonCluster, error) {
	return c.cluster, c.cluster.Persist(cluster.Creating, cluster.CreatingMessage)
}

// Create implements the clusterCreator interface.
func (c *commonCreator) Create(ctx context.Context) error {
	return c.cluster.CreateCluster()
}
