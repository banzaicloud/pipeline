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
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/goph/emperror"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

type Clusters interface {
	GetCluster(ctx context.Context, id uint) (Cluster, error)
}

type Cluster interface {
	GetID() uint
	GetUID() string
	GetOrganizationId() uint
	UpdateStatus(string, string) error
}

type AWSCluster interface {
	GetAWSClient() (*session.Session, error)
	Cluster
}

const CreateClusterWorkflowName = "pke-create-cluster"

type CreateClusterWorkflowInput struct {
	ClusterID uint
}

func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	err := generateCertificates(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	createAWSRolesActivityInput := CreateAWSRolesActivityInput{
		ClusterID: input.ClusterID,
	}

	var rolesStackID string
	err = workflow.ExecuteActivity(ctx, CreateAWSRolesActivityName, createAWSRolesActivityInput).Get(ctx, &rolesStackID)
	if err != nil {
		return err
	}

	if rolesStackID != "" {
		// TODO: wait for stack
	}

	signalName := "master-ready"
	signalChan := workflow.GetSignalChannel(ctx, signalName)

	s := workflow.NewSelector(ctx)
	s.AddReceive(signalChan, func(c workflow.Channel, more bool) {
		c.Receive(ctx, nil)
		workflow.GetLogger(ctx).Info("Received signal!", zap.String("signal", signalName))
	})
	s.Select(ctx)

	return nil
}

func generateCertificates(ctx workflow.Context, clusterID uint) error {
	generateCertificatesActivityInput := GenerateCertificatesActivityInput{
		ClusterID: clusterID,
	}

	return workflow.ExecuteActivity(ctx, GenerateCertificatesActivityName, generateCertificatesActivityInput).Get(ctx, nil)
}

const CreateClusterActivityName = "pke-create-cluster-activity"

type CreateClusterActivity struct {
	clusters       Clusters
	tokenGenerator TokenGenerator
}

func NewCreateClusterActivity(clusters Clusters, tokenGenerator TokenGenerator) *CreateClusterActivity {
	return &CreateClusterActivity{
		clusters:       clusters,
		tokenGenerator: tokenGenerator,
	}
}

type TokenGenerator interface {
	GenerateClusterToken(orgID, clusterID uint) (string, string, error)
}

type CreateClusterActivityInput struct {
	ClusterID uint
}

func (a *CreateClusterActivity) Execute(ctx context.Context, input CreateClusterActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	// prepare input for real AWS flow
	_, _, err = a.tokenGenerator.GenerateClusterToken(c.GetOrganizationId(), c.GetID())
	if err != nil {
		return emperror.Wrap(err, "can't generate Pipeline token")
	}
	//client, err := c.GetAWSClient()
	//if err != nil {
	//	return err
	//}
	//cloudformationSrv := cloudformation.New(client)
	//err = CreateMasterCF(cloudformationSrv)
	//if err != nil {
	//	return emperror.Wrap(err, "can't create master CF template")
	//}
	//token := "XXX" // TODO masked from dumping valid tokens to log
	//for _, nodePool := range c.model.NodePools {
	//cmd := c.GetBootstrapCommand(nodePool.Name, externalBaseURL, token)
	//c.log.Debugf("TODO: start ASG with command %s", cmd)
	//}

	c.UpdateStatus("CREATING", "Waiting for Kubeconfig from master node.")

	return nil
}
