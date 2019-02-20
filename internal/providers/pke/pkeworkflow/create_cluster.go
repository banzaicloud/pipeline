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

func getDefaultImageID(region string) string {
	return map[string]string{
		"eu-north-1":     "ami-b133bccf",
		"ap-south-1":     "ami-1780a878",
		"eu-west-3":      "ami-262e9f5b",
		"eu-west-2":      "ami-00846a67",
		"eu-west-1":      "ami-3548444c",
		"ap-northeast-2": "ami-bf9c36d1",
		"ap-northeast-1": "ami-8e8847f1",
		"sa-east-1":      "ami-cb5803a7",
		"ca-central-1":   "ami-e802818c",
		"ap-southeast-1": "ami-8e0205f2",
		"ap-southeast-2": "ami-d8c21dba",
		"eu-central-1":   "ami-dd3c0f36",
		"us-east-1":      "ami-77ec9308",
		"us-east-2":      "ami-9c0638f9",
		"us-west-1":      "ami-4826c22b",
		"us-west-2":      "ami-3ecc8f46",
	}[region]
}

type CreateClusterWorkflowInput struct {
	OrganizationID      uint
	ClusterID           uint
	ClusterUID          string
	ClusterName         string
	SecretID            string
	Region              string
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

	// Generate CA certificates
	err := workflow.ExecuteActivity(
		ctx,
		GenerateCertificatesActivityName,
		GenerateCertificatesActivityInput{ClusterID: input.ClusterID},
	).Get(ctx, nil)
	if err != nil {
		return err
	}

	createAWSRolesActivityInput := CreateAWSRolesActivityInput{
		ClusterID: input.ClusterID,
		Region:    "us-east-1",
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
			Region:    "us-east-1",
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

	var keyOut UploadSSHKeyPairActivityOutput
	UploadSSHKeyPairActivityInput := UploadSSHKeyPairActivityInput{
		ClusterID: input.ClusterID,
	}
	if err := workflow.ExecuteActivity(ctx, UploadSSHKeyPairActivityName, UploadSSHKeyPairActivityInput).Get(ctx, &keyOut); err != nil {
		return err
	}

	var masterAvailabilityZone string
	var master NodePool
	for _, np := range nodePools {
		if np.Master {
			master = np
			if len(np.AvailabilityZones) <= 0 || np.AvailabilityZones[0] == "" {
				return errors.New(fmt.Sprintf("missing availability zone for nodepool %q", np.Name))
			}
			masterAvailabilityZone = np.AvailabilityZones[0]
			break
		}
	}

	var masterStackID string
	// TODO refactor network things
	createMasterActivityInput := CreateMasterActivityInput{
		ClusterID:             input.ClusterID,
		AvailabilityZone:      masterAvailabilityZone,
		VPCID:                 vpcOutput["VpcId"],
		SubnetID:              strings.Split(vpcOutput["SubnetIds"], ",")[0],
		EIPAllocationID:       eip.AllocationId,
		MasterInstanceProfile: rolesOutput["MasterInstanceProfile"],
		ExternalBaseUrl:       input.PipelineExternalURL,
		Pool:                  master,
		SSHKeyName:            keyOut.KeyName,
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
		if !np.Master {

			createWorkerPoolActivityInput := CreateWorkerPoolActivityInput{
				ClusterID:             input.ClusterID,
				Pool:                  np,
				WorkerInstanceProfile: rolesOutput["WorkerInstanceProfile"],
				VPCID:                 vpcOutput["VpcId"],
				SubnetID:              strings.Split(vpcOutput["SubnetIds"], ",")[1],
				ClusterSecurityGroup:  clusterSecurityGroup,
				ExternalBaseUrl:       input.PipelineExternalURL,
				SSHKeyName:            keyOut.KeyName,
			}

			err = workflow.ExecuteActivity(ctx, CreateWorkerPoolActivityName, createWorkerPoolActivityInput).Get(ctx, nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type TokenGenerator interface {
	GenerateClusterToken(orgID, clusterID uint) (string, string, error)
}

type CreateClusterActivityInput struct {
	ClusterID uint
}
