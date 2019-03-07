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
	"context"
	"fmt"

	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/goph/emperror"
	"go.uber.org/cadence/client"
)

type commonUpdater struct {
	request                  *cluster.UpdateClusterRequest
	cluster                  CommonCluster
	userID                   uint
	scaleOptionsChanged      bool
	clusterPropertiesChanged bool
	workflowClient           client.Client
	externalBaseURL          string
}

type commonUpdateValidationError struct {
	msg string

	invalidRequest     bool
	preconditionFailed bool
}

func (e *commonUpdateValidationError) Error() string {
	return e.msg
}

func (e *commonUpdateValidationError) IsInvalid() bool {
	return e.invalidRequest
}

func (e *commonUpdateValidationError) IsPreconditionFailed() bool {
	return e.preconditionFailed
}

// NewCommonClusterUpdater returns a new cluster creator instance.
func NewCommonClusterUpdater(request *cluster.UpdateClusterRequest, cluster CommonCluster, userID uint, workflowClient client.Client, externalBaseURL string) *commonUpdater {
	return &commonUpdater{
		request:         request,
		cluster:         cluster,
		userID:          userID,
		workflowClient:  workflowClient,
		externalBaseURL: externalBaseURL,
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

	if status.Status != cluster.Running && status.Status != cluster.Warning {
		return emperror.With(
			&commonUpdateValidationError{
				msg:                fmt.Sprintf("cluster is not in %s or %s state yet", cluster.Running, cluster.Warning),
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

	c.scaleOptionsChanged = isDifferent(c.request.ScaleOptions, c.cluster.GetScaleOptions()) == nil
	c.clusterPropertiesChanged = true

	if err := c.cluster.CheckEqualityToUpdate(c.request); err != nil {
		c.clusterPropertiesChanged = false
		if !c.scaleOptionsChanged {
			return nil, &commonUpdateValidationError{
				msg:            err.Error(),
				invalidRequest: true,
			}
		}
	}

	if err := c.request.Validate(); err != nil {
		return nil, &commonUpdateValidationError{
			msg:            err.Error(),
			invalidRequest: true,
		}
	}

	return c.cluster, c.cluster.SetStatus(cluster.Updating, cluster.UpdatingMessage)
}

// Update implements the clusterUpdater interface.
func (c *commonUpdater) Update(ctx context.Context) error {
	if c.scaleOptionsChanged {
		c.cluster.SetScaleOptions(c.request.ScaleOptions)
	}

	if !c.clusterPropertiesChanged && !c.scaleOptionsChanged {
		return nil
	}

	// pre deploy NodePoolLabelSet objects for each new node pool to be created
	nodePools := getNodePoolsFromUpdateRequest(c.request)
	// to avoid overriding user specified labels, in case of of an empty label map in update request,
	// set noReturnIfNoUserLabels = true
	labelsMap, err := GetDesiredLabelsForCluster(c.cluster, nodePools, true)
	if err != nil {
		return err
	}
	if err = DeployNodePoolLabelsSet(c.cluster, labelsMap); err != nil {
		return err
	}

	if updater, ok := c.cluster.(interface {
		UpdatePKECluster(context.Context, *cluster.UpdateClusterRequest, client.Client, string) error
	}); ok {
		err = updater.UpdatePKECluster(ctx, c.request, c.workflowClient, c.externalBaseURL)
	} else {
		err = c.cluster.UpdateCluster(c.request, c.userID)
	}
	if err != nil {
		return err
	}

	if err := DeployClusterAutoscaler(c.cluster); err != nil {
		return emperror.Wrap(err, "deploying cluster autoscaler failed")
	}

	// on certain clouds like Alibaba & Ec2_Banzaicloud we still need to add node pool name labels
	if err := LabelNodesWithNodePoolName(c.cluster); err != nil {
		return emperror.Wrap(err, "adding labels to nodes failed")
	}
	return nil
}
