package cluster

import (
	"context"
	"fmt"

	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/goph/emperror"
)

type commonUpdater struct {
	request *cluster.UpdateClusterRequest
	cluster CommonCluster
	userID  uint
}

type commonUpdateValidationError struct {
	msg string

	invalidRequest     bool
	preconditionFailed bool
}

func (e *commonUpdateValidationError) Error() string {
	return e.msg
}

// NewCommonClusterUpdater returns a new cluster creator instance.
func NewCommonClusterUpdater(request *cluster.UpdateClusterRequest, cluster CommonCluster, userID uint) *commonUpdater {
	return &commonUpdater{
		request: request,
		cluster: cluster,
		userID:  userID,
	}
}

// Validate implements the clusterUpdater interface.
func (c *commonUpdater) Validate(ctx context.Context) error {
	if c.cluster.GetCloud() != c.request.Cloud {
		return &commonUpdateValidationError{
			msg:            fmt.Sprintf("cloud provider [%s] does not match the cluster's cloud provider [%s]", c.request.Cloud, c.cluster.GetCloud()),
			invalidRequest: true,
		}
	}

	status, err := c.cluster.GetStatus()
	if err != nil {
		return emperror.Wrap(err, "could not get cluster status")
	}

	if status.Status != cluster.Running {
		return emperror.With(
			&commonUpdateValidationError{
				msg:                fmt.Sprintf("cluster is not in %s state yet", cluster.Running),
				preconditionFailed: true,
			},
			"status", status.Status,
		)
	}

	return nil
}

// Prepare implements the clusterUpdater interface.
func (c *commonUpdater) Prepare(ctx context.Context) (CommonCluster, error) {
	c.cluster.AddDefaultsToUpdate(c.request)

	if err := c.cluster.CheckEqualityToUpdate(c.request); err != nil {
		return nil, &commonUpdateValidationError{
			msg:            err.Error(),
			invalidRequest: true,
		}
	}

	if err := c.request.Validate(); err != nil {
		return nil, &commonUpdateValidationError{
			msg:            err.Error(),
			invalidRequest: true,
		}
	}

	return c.cluster, c.cluster.Persist(cluster.Updating, cluster.UpdatingMessage)
}

// Update implements the clusterUpdater interface.
func (c *commonUpdater) Update(ctx context.Context) error {
	return c.cluster.UpdateCluster(c.request, c.userID)
}
