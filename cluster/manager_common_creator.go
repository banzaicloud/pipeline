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

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/providers/pke"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
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
	if err := c.cluster.Persist(); err != nil {
		return nil, err
	}

	if err := c.cluster.SetStatus(pkgCluster.Creating, pkgCluster.CreatingMessage); err != nil {
		return nil, err
	}

	return c.cluster, nil
}

// Create implements the clusterCreator interface.
func (c *commonCreator) Create(ctx context.Context) error {
	return c.cluster.CreateCluster()
}

type TokenGenerator interface {
	GenerateClusterToken(orgID uint, clusterID uint) (string, string, error)
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

		oidcEnabled: request.Properties.CreateClusterPKE.Kubernetes.OIDC.Enabled,
	}
}

type createPKEClusterer interface {
	SetCurrentWorkflowID(workflowID string) error
}

type pkeCreator struct {
	workflowClient client.Client

	commonCreator

	oidcEnabled bool
}

// Create implements the clusterCreator interface.
func (c *pkeCreator) Create(ctx context.Context) error {
	var externalBaseURL string
	var externalBaseURLInsecure bool
	var ok bool
	if externalBaseURL, ok = ctx.Value(ExternalBaseURLKey).(string); !ok {
		return errors.New("externalBaseURL missing from context")
	}
	if externalBaseURLInsecure, ok = ctx.Value(ExternalBaseURLInsecureKey).(bool); !ok {
		return errors.New("externalBaseURLInsecure missing from context")
	}

	input := pkeworkflow.CreateClusterWorkflowInput{
		OrganizationID:              uint(c.cluster.GetOrganizationId()),
		ClusterID:                   uint(c.cluster.GetID()),
		ClusterUID:                  c.cluster.GetUID(),
		ClusterName:                 c.cluster.GetName(),
		SecretID:                    string(c.cluster.GetSecretId()),
		Region:                      c.cluster.GetLocation(),
		PipelineExternalURL:         externalBaseURL,
		PipelineExternalURLInsecure: externalBaseURLInsecure,
		OIDCEnabled:                 c.oidcEnabled,
	}

	providerConfig := c.request.Properties.CreateClusterPKE.Network.ProviderConfig
	if providerConfig != nil {
		cpc := &pke.NetworkCloudProviderConfigAmazon{}
		err := mapstructure.Decode(providerConfig, &cpc)
		if err != nil {
			return err
		}

		input.VPCID = cpc.VPCID

		if len(cpc.Subnets) > 0 {
			input.SubnetID = string(cpc.Subnets[0])
		}
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
