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
	"time"

	pkgCluster "github.com/banzaicloud/pipeline/cluster"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const CreateClusterWorkflowName = "pke-create-cluster"

type CreateClusterWorkflowInput struct {
	ClusterID uint
}

func CreateClusterWorkflow(ctx workflow.Context, input uint) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	generateCertificatesActivityInput := GenerateCertificatesActivityInput{
		ClusterID: input,
	}

	err := workflow.ExecuteActivity(ctx, GenerateCertificatesActivityName, generateCertificatesActivityInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	createClusterActivityInput := CreateClusterActivityInput{
		ClusterID: input,
	}

	err = workflow.ExecuteActivity(ctx, CreateClusterActivityName, createClusterActivityInput).Get(ctx, nil)
	if err != nil {
		return err
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

const CreateClusterActivityName = "pke-create-cluster-activity"

type CreateClusterActivity struct {
	clusterManager *pkgCluster.Manager
	tokenGenerator TokenGenerator
}

func NewCreateClusterActivity(clusterManager *pkgCluster.Manager, tokenGenerator TokenGenerator) *CreateClusterActivity {
	return &CreateClusterActivity{
		clusterManager: clusterManager,
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
	c, err := a.clusterManager.GetClusterByIDOnly(ctx, input.ClusterID)
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

const GenerateCertificatesActivityName = "pke-generate-certificates-activity"

type GenerateCertificatesActivity struct {
	clusterManager *pkgCluster.Manager
}

func NewGenerateCertificatesActivity(clusterManager *pkgCluster.Manager) *GenerateCertificatesActivity {
	return &GenerateCertificatesActivity{
		clusterManager: clusterManager,
	}
}

type GenerateCertificatesActivityInput struct {
	ClusterID uint
}

func (a *GenerateCertificatesActivity) Execute(ctx context.Context, input GenerateCertificatesActivityInput) error {
	c, err := a.clusterManager.GetClusterByIDOnly(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	// Generate certificates
	req := &secret.CreateSecretRequest{
		Name: fmt.Sprintf("cluster-%d-ca", c.GetID()),
		Type: pkgSecret.PKESecretType,
		Tags: []string{
			fmt.Sprintf("clusterUID:%s", c.GetUID()),
			pkgSecret.TagBanzaiReadonly,
			pkgSecret.TagBanzaiHidden,
		},
	}
	_, err = secret.Store.GetOrCreate(c.GetOrganizationId(), req)
	if err != nil {
		return err
	}

	return nil
}
