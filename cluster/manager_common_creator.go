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
	"time"

	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/pkg/errors"
	"go.uber.org/cadence/client"
)

type commonCreator struct {
	request *pkgCluster.CreateClusterRequest
	cluster CommonCluster
}

// NewCommonClusterCreator returns a new cluster creator instance.
func NewCommonClusterCreator(request *pkgCluster.CreateClusterRequest, cluster CommonCluster) *commonCreator {
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
	return c.cluster, c.cluster.Persist(pkgCluster.Creating, pkgCluster.CreatingMessage)
}

// Create implements the clusterCreator interface.
func (c *commonCreator) Create(ctx context.Context) error {
	return c.cluster.CreateCluster()
}

type TokenGenerator interface {
	GenerateClusterToken(orgID pkgAuth.OrganizationID, clusterID pkgCluster.ClusterID) (string, string, error)
}

// NewClusterCreator returns a new PKE or Common cluster creator instance depending on the cluster.
func NewClusterCreator(request *pkgCluster.CreateClusterRequest, cluster CommonCluster, workflowClient client.Client) clusterCreator {
	common := NewCommonClusterCreator(request, cluster)
	if _, ok := cluster.(createPKEClusterer); !ok {
		return common
	}

	return &pkeCreator{
		workflowClient: workflowClient,

		commonCreator: *common,
	}
}

type createPKEClusterer interface {
	SetCurrentWorkflowID(workflowID string) error
}

type pkeCreator struct {
	workflowClient client.Client

	commonCreator
}

// Create implements the clusterCreator interface.
func (c *pkeCreator) Create(ctx context.Context) error {
	var externalBaseURL string
	var ok bool
	if externalBaseURL, ok = ctx.Value("ExternalBaseURL").(string); !ok {
		return errors.New("externalBaseURL missing from context")
	}
	input := pkeworkflow.CreateClusterWorkflowInput{
		ClusterID:           c.cluster.GetID(),
		PipelineExternalURL: externalBaseURL,
	}
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
	}
	exec, err := c.workflowClient.ExecuteWorkflow(ctx, workflowOptions, pkeworkflow.CreateClusterWorkflowName, input)
	if err != nil {
		return err
	}

	err = c.cluster.(createPKEClusterer).SetCurrentWorkflowID(exec.GetID())
	if err != nil {
		return err
	}

	err = exec.Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
