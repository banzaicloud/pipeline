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
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const CreateClusterWorkflowName = "pke-create-cluster"
const pkeVersion = "0.4.9"

func getDefaultImageID(region, kubernetesVersion string) (string, error) {
	constraint112, err := semver.NewConstraint("~1.12.0")
	if err != nil {
		return "", emperror.Wrap(err, "could not create semver constraint for Kubernetes version 1.12+")
	}

	constraint113, err := semver.NewConstraint("~1.13.0")
	if err != nil {
		return "", emperror.Wrap(err, "could not create semver constraint for Kubernetes version 1.13+")
	}

	kubeVersion, err := semver.NewVersion(kubernetesVersion)
	if err != nil {
		return "", emperror.WrapWith(err, "could not create semver from Kubernetes version", "kubernetesVersion", kubernetesVersion)
	}

	switch {
	case constraint112.Check(kubeVersion):
		return map[string]string{
			"ap-northeast-1": "ami-08a85b1563efcdbfa",
			"ap-northeast-2": "ami-01ad53644d5e714e0",
			"ap-south-1":     "ami-0552dbb025034dd47",
			"ap-southeast-1": "ami-0a4e5de01c309fcdb",
			"ap-southeast-2": "ami-00b39bd845ecf13f8",
			"ca-central-1":   "ami-05b3f0a65b7cc5edb",
			"eu-central-1":   "ami-05b74dc857dc64612",
			"eu-north-1":     "ami-00ccc3f51ebcd20d1",
			"eu-west-1":      "ami-0e47f158914d41337",
			"eu-west-2":      "ami-04213a12ded35d40a",
			"eu-west-3":      "ami-0df179038236c5fe1",
			"sa-east-1":      "ami-0557d1fefc68884c0",
			"us-east-1":      "ami-03d6c616f020991c6",
			"us-east-2":      "ami-017ff3156c58d64de",
			"us-west-1":      "ami-0ed166e4d66056cba",
			"us-west-2":      "ami-0ee06a6e5ea34c447",
		}[region], nil
	case constraint113.Check(kubeVersion):
		return map[string]string{
			"ap-northeast-1": "ami-0e51ebcaab2ee7f64",
			"ap-northeast-2": "ami-03f97043746d70ea8",
			"ap-south-1":     "ami-0743f092ee208761e",
			"ap-southeast-1": "ami-021810a22ae6a6972",
			"ap-southeast-2": "ami-0707af9819a36d29b",
			"ca-central-1":   "ami-01d9e44948e56a6c1",
			"eu-central-1":   "ami-04943b3a24081a11a",
			"eu-north-1":     "ami-0a81717b1f805ac46",
			"eu-west-1":      "ami-08bd41dfe9b6ace0c",
			"eu-west-2":      "ami-0bae46bc373695303",
			"eu-west-3":      "ami-067a77cf9189870df",
			"sa-east-1":      "ami-02ed55f4ee9db122a",
			"us-east-1":      "ami-0259f84ec3cbc54a1",
			"us-east-2":      "ami-047712e13d40b4739",
			"us-west-1":      "ami-0e56d84477e285018",
			"us-west-2":      "ami-0209e5179cf144bb1",
		}[region], nil
	default:
		return map[string]string{
			"ap-northeast-1": "ami-050580615eb00d744",
			"ap-northeast-2": "ami-051b65659a2c549b0",
			"ap-south-1":     "ami-03adffe261d08c4ec",
			"ap-southeast-1": "ami-0c0f3a44506a4f470",
			"ap-southeast-2": "ami-06d552a20a61ab8fe",
			"ca-central-1":   "ami-07b0387c0bc3bf4d0",
			"eu-central-1":   "ami-0dc9154691d8a1757",
			"eu-north-1":     "ami-044edb04df20f127b",
			"eu-west-1":      "ami-0be08db35d79874b9",
			"eu-west-2":      "ami-062ce851cb781d581",
			"eu-west-3":      "ami-0f78066d649b69b51",
			"sa-east-1":      "ami-08401edb5361125d5",
			"us-east-1":      "ami-09b34d885e47bb377",
			"us-east-2":      "ami-030f8c953c69c25c0",
			"us-west-1":      "ami-0d87a1f4e1743e1d6",
			"us-west-2":      "ami-0dbd115d30cda6652",
		}[region], nil

	}
}

type TokenGenerator interface {
	GenerateClusterToken(orgID, clusterID uint) (string, string, error)
}

type CreateClusterWorkflowInput struct {
	OrganizationID              uint
	ClusterID                   uint
	ClusterUID                  string
	ClusterName                 string
	SecretID                    string
	Region                      string
	PipelineExternalURL         string
	PipelineExternalURLInsecure bool
	DexEnabled                  bool
	VPCID                       string
	SubnetID                    string
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
	{
		activityInput := GenerateCertificatesActivityInput{ClusterID: input.ClusterID}

		err := workflow.ExecuteActivity(ctx, GenerateCertificatesActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	// Generic AWS activity input
	awsActivityInput := AWSActivityInput{
		OrganizationID: input.OrganizationID,
		SecretID:       input.SecretID,
		Region:         input.Region,
	}

	var rolesStackID string

	// Create AWS roles
	{
		activityInput := CreateAWSRolesActivityInput{AWSActivityInput: awsActivityInput, ClusterID: input.ClusterID}
		activityInput.AWSActivityInput.Region = "us-east-1"
		err := workflow.ExecuteActivity(ctx, CreateAWSRolesActivityName, activityInput).Get(ctx, &rolesStackID)
		if err != nil {
			return err
		}
	}

	var rolesOutput map[string]string

	// Wait for roles
	{
		if rolesStackID == "" {
			return errors.New("missing AWS role stack ID")
		}

		activityInput := WaitCFCompletionActivityInput{AWSActivityInput: awsActivityInput, StackID: rolesStackID}
		activityInput.AWSActivityInput.Region = "us-east-1"

		err := workflow.ExecuteActivity(ctx, WaitCFCompletionActivityName, activityInput).Get(ctx, &rolesOutput)
		if err != nil {
			return err
		}
	}

	var vpcStackID string

	// Create VPC
	{
		activityInput := CreateVPCActivityInput{
			AWSActivityInput: awsActivityInput,
			ClusterID:        input.ClusterID,
			ClusterName:      input.ClusterName,
			VPCID:            input.VPCID,
			SubnetID:         input.SubnetID,
		}
		err := workflow.ExecuteActivity(ctx, CreateVPCActivityName, activityInput).Get(ctx, &vpcStackID)
		if err != nil {
			return err
		}
	}

	var vpcOutput map[string]string

	// Wait for VPC
	{
		if vpcStackID == "" {
			return errors.New("missing VPC stack ID")
		}

		activityInput := WaitCFCompletionActivityInput{AWSActivityInput: awsActivityInput, StackID: vpcStackID}

		err := workflow.ExecuteActivity(ctx, WaitCFCompletionActivityName, activityInput).Get(ctx, &vpcOutput)
		if err != nil {
			return err
		}
	}

	var nodePools []NodePool

	// List node pools
	{
		activityInput := ListNodePoolsActivityInput{ClusterID: input.ClusterID}
		err := workflow.ExecuteActivity(ctx, ListNodePoolsActivityName, activityInput).Get(ctx, &nodePools)
		if err != nil {
			return err
		}
	}

	var master NodePool
	for _, np := range nodePools {
		if np.Master {
			master = np
			if len(np.AvailabilityZones) <= 0 || np.AvailabilityZones[0] == "" {
				return errors.Errorf("missing availability zone for nodepool %q", np.Name)
			}
			break
		}
	}

	var keyOut UploadSSHKeyPairActivityOutput

	// Upload SSH key pair
	{
		activityInput := UploadSSHKeyPairActivityInput{
			ClusterID: input.ClusterID,
		}
		err := workflow.ExecuteActivity(ctx, UploadSSHKeyPairActivityName, activityInput).Get(ctx, &keyOut)
		if err != nil {
			return err
		}
	}

	// Create dex client for the cluster
	if input.DexEnabled {
		activityInput := CreateDexClientActivityInput{
			ClusterID: input.ClusterID,
		}
		err := workflow.ExecuteActivity(ctx, CreateDexClientActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	var externalAddress string

	multiMaster := master.MaxCount > 1

	masterInput := CreateMasterActivityInput{
		ClusterID:               input.ClusterID,
		VPCID:                   vpcOutput["VpcId"],
		SubnetID:                strings.Split(vpcOutput["SubnetIds"], ",")[0],
		MultiMaster:             multiMaster,
		MasterInstanceProfile:   rolesOutput["MasterInstanceProfile"],
		ExternalBaseUrl:         input.PipelineExternalURL,
		ExternalBaseUrlInsecure: input.PipelineExternalURLInsecure,
		Pool:                    master,
		SSHKeyName:              keyOut.KeyName,
		AvailabilityZone:        master.AvailabilityZones[0],
	}

	if multiMaster {

		// Create NLB
		var activityOutput CreateNLBActivityOutput
		activityInput := &CreateNLBActivityInput{
			AWSActivityInput: awsActivityInput,
			ClusterID:        input.ClusterID,
			ClusterName:      input.ClusterName,
			VPCID:            vpcOutput["VpcId"],
			SubnetIds:        strings.Split(vpcOutput["SubnetIds"], ","),
		}

		err := workflow.ExecuteActivity(ctx, CreateNLBActivityName, activityInput).Get(ctx, &activityOutput)
		if err != nil {
			return err
		}

		masterInput.TargetGroup = activityOutput.TargetGroup
		externalAddress = activityOutput.DNSName

	} else {

		// Create EIP
		var eip CreateElasticIPActivityOutput
		activityInput := &CreateElasticIPActivityInput{
			AWSActivityInput: awsActivityInput,
			ClusterID:        input.ClusterID,
			ClusterName:      input.ClusterName,
		}

		err := workflow.ExecuteActivity(ctx, CreateElasticIPActivityName, activityInput).Get(ctx, &eip)
		if err != nil {
			return err
		}

		masterInput.EIPAllocationID = eip.AllocationId
		externalAddress = eip.PublicIp
	}

	// Update cluster network
	{
		activityInput := &UpdateClusterNetworkActivityInput{
			ClusterID:       input.ClusterID,
			APISeverAddress: externalAddress,
			VPCID:           vpcOutput["VpcId"],
			Subnets:         vpcOutput["SubnetIds"],
		}
		err := workflow.ExecuteActivity(ctx, UpdateClusterNetworkActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	var masterStackID string
	// Create master
	{
		err := workflow.ExecuteActivity(ctx, CreateMasterActivityName, masterInput).Get(ctx, &masterStackID)
		if err != nil {
			return err
		}
	}

	var masterOutput map[string]string

	// Wait for master
	{
		if masterStackID == "" {
			return errors.New("missing stack ID")
		}

		activityInput := WaitCFCompletionActivityInput{AWSActivityInput: awsActivityInput, StackID: masterStackID}
		err := workflow.ExecuteActivity(ctx, WaitCFCompletionActivityName, activityInput).Get(ctx, &masterOutput)
		if err != nil {
			return err
		}
	}

	signalName := "master-ready"
	signalChan := workflow.GetSignalChannel(ctx, signalName)

	s := workflow.NewSelector(ctx)
	s.AddReceive(signalChan, func(c workflow.Channel, more bool) {
		c.Receive(ctx, nil)
		workflow.GetLogger(ctx).Info("Received signal!", zap.String("signal", signalName))
	})
	s.Select(ctx)

	if len(nodePools) == 1 {
		err := workflow.ExecuteActivity(ctx, SetMasterTaintActivityName, SetMasterTaintActivityInput{
			ClusterID: input.ClusterID,
		}).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	// Create nodes
	{
		for _, np := range nodePools {
			if !np.Master {
				createWorkerPoolActivityInput := CreateWorkerPoolActivityInput{
					ClusterID:               input.ClusterID,
					Pool:                    np,
					WorkerInstanceProfile:   rolesOutput["WorkerInstanceProfile"],
					VPCID:                   vpcOutput["VpcId"],
					SubnetID:                strings.Split(vpcOutput["SubnetIds"], ",")[0],
					ClusterSecurityGroup:    masterOutput["ClusterSecurityGroup"],
					ExternalBaseUrl:         input.PipelineExternalURL,
					ExternalBaseUrlInsecure: input.PipelineExternalURLInsecure,
					SSHKeyName:              keyOut.KeyName,
				}

				err := workflow.ExecuteActivity(ctx, CreateWorkerPoolActivityName, createWorkerPoolActivityInput).Get(ctx, nil)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
