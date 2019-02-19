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
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const CreateClusterWorkflowName = "pke-create-cluster"

type CreateClusterWorkflowInput struct {
	ClusterID           uint
	PipelineExternalURL string
}

func CreateClusterWorkflow(ctx workflow.Context, input CreateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
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

	var rolesOutput map[string]string
	var rolesStackID string
	err = workflow.ExecuteActivity(ctx, CreateAWSRolesActivityName, createAWSRolesActivityInput).Get(ctx, &rolesStackID)
	if err != nil {
		return err
	}

	if rolesStackID != "" {
		waitCFCompletionActivityInput := WaitCFCompletionActivityInput{
			ClusterID: input.ClusterID,
			StackID:   rolesStackID,
		}

		err = workflow.ExecuteActivity(ctx, WaitCFCompletionActivityName, waitCFCompletionActivityInput).Get(ctx, &rolesOutput)
		if err != nil {
			return err
		}
	}

	var vpcOutput map[string]string
	var vpcStackID string
	createVACActivityInput := CreateVPCActivityInput{ClusterID: input.ClusterID}
	err = workflow.ExecuteActivity(ctx, CreateVPCActivityName, createVACActivityInput).Get(ctx, &vpcStackID)
	if err != nil {
		return err
	}
	if vpcStackID != "" {
		waitCFCompletionActivityInput := WaitCFCompletionActivityInput{
			ClusterID: input.ClusterID,
			StackID:   vpcStackID,
		}

		err = workflow.ExecuteActivity(ctx, WaitCFCompletionActivityName, waitCFCompletionActivityInput).Get(ctx, &vpcOutput)
		if err != nil {
			return err
		}
	}

	createElasticIPActivityInput := &CreateElasticIPActivityInput{
		ClusterID: input.ClusterID,
	}
	var eip CreateElasticIPActivityOutput
	if err := workflow.ExecuteActivity(ctx, CreateElasticIPActivityName, createElasticIPActivityInput).Get(ctx, &eip); err != nil {
		return err
	}

	updateClusterNetworkActivityInput := &UpdateClusterNetworkActivityInput{
		ClusterID:       input.ClusterID,
		APISeverAddress: eip.PublicIp,
		VPCID:           vpcOutput["VpcId"],
		Subnets:         vpcOutput["SubnetIds"],
	}
	if err := workflow.ExecuteActivity(ctx, UpdateClusterNetworkActivityName, updateClusterNetworkActivityInput).Get(ctx, nil); err != nil {
		return err
	}

	var nodePools []NodePool
	listNodePoolsActivityInput := ListNodePoolsActivityInput{
		ClusterID: input.ClusterID,
	}

	if err := workflow.ExecuteActivity(ctx, ListNodePoolsActivityName, listNodePoolsActivityInput).Get(ctx, &nodePools); err != nil {
		return err
	}

	var keyName string
	UploadSSHKeyPairActivityInput := UploadSSHKeyPairActivityInput{
		ClusterID: input.ClusterID,
	}
	if err := workflow.ExecuteActivity(ctx, UploadSSHKeyPairActivityName, UploadSSHKeyPairActivityInput).Get(ctx, &keyName); err != nil {
		return err
	}

	var masterInstanceType, masterAvailabilityZone string
	for _, np := range nodePools {
		if np.Master {
			masterInstanceType = np.InstanceType
			if len(np.AvailabilityZones) <= 0 {
				return errors.New(fmt.Sprintf("missing availability zone from nodepool %q", np.Name))
			}
			masterAvailabilityZone = np.AvailabilityZones[0]
			break
		}
	}
	if masterInstanceType == "" || masterAvailabilityZone == "" {
		return errors.New("missing instance type or availability zone for master node")
	}

	var masterStackID string
	// TODO refactor network things
	createMasterActivityInput := CreateMasterActivityInput{
		ClusterID:             input.ClusterID,
		InstanceType:          masterInstanceType,
		AvailabilityZone:      masterAvailabilityZone,
		VPCID:                 vpcOutput["VpcId"],
		SubnetID:              strings.Split(vpcOutput["SubnetIds"], ",")[0],
		EIPAllocationID:       eip.AllocationId,
		MasterInstanceProfile: rolesOutput["MasterInstanceProfile"],
		ExternalBaseUrl:       input.PipelineExternalURL,
	}
	if err := workflow.ExecuteActivity(ctx, CreateMasterActivityName, createMasterActivityInput).Get(ctx, &masterStackID); err != nil {
		return err
	}

	var masterOutput map[string]string
	if masterStackID != "" {
		waitCFCompletionActivityInput := WaitCFCompletionActivityInput{
			ClusterID: input.ClusterID,
			StackID:   masterStackID,
		}

		err = workflow.ExecuteActivity(ctx, WaitCFCompletionActivityName, waitCFCompletionActivityInput).Get(ctx, &masterOutput)
		if err != nil {
			return err
		}
	}

	clusterSecurityGroup := masterOutput["ClusterSecurityGroup"]

	signalName := "master-ready"
	signalChan := workflow.GetSignalChannel(ctx, signalName)

	s := workflow.NewSelector(ctx)
	s.AddReceive(signalChan, func(c workflow.Channel, more bool) {
		c.Receive(ctx, nil)
		workflow.GetLogger(ctx).Info("Received signal!", zap.String("signal", signalName))
	})
	s.Select(ctx)

	for _, np := range nodePools {
		if !np.Master && np.Worker {

			createWorkerPoolActivityInput := CreateWorkerPoolActivityInput{
				ClusterID:             input.ClusterID,
				Pool:                  np,
				WorkerInstanceProfile: rolesOutput["WorkerInstanceProfile"],
				VPCID:                 vpcOutput["VpcId"],
				SubnetID:              strings.Split(vpcOutput["SubnetIds"], ",")[1],
				ClusterSecurityGroup:  clusterSecurityGroup,
				ExternalBaseUrl:       input.PipelineExternalURL,
			}

			err = workflow.ExecuteActivity(ctx, CreateWorkerPoolActivityName, createWorkerPoolActivityInput).Get(ctx, nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func generateCertificates(ctx workflow.Context, clusterID uint) error {
	generateCertificatesActivityInput := GenerateCertificatesActivityInput{
		ClusterID: clusterID,
	}

	return workflow.ExecuteActivity(ctx, GenerateCertificatesActivityName, generateCertificatesActivityInput).Get(ctx, nil)
}

type TokenGenerator interface {
	GenerateClusterToken(orgID, clusterID uint) (string, string, error)
}

type CreateClusterActivityInput struct {
	ClusterID uint
}
