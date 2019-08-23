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

	"emperror.dev/errors"
	"github.com/Masterminds/semver"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const CreateClusterWorkflowName = "pke-create-cluster"
const pkeVersion = "0.4.12"

func getDefaultImageID(region, kubernetesVersion string) (string, error) {

	constraint113, err := semver.NewConstraint("~1.13.0")
	if err != nil {
		return "", errors.Wrap(err, "could not create semver constraint for Kubernetes version 1.13+")
	}

	constraint114, err := semver.NewConstraint("~1.14.0")
	if err != nil {
		return "", errors.Wrap(err, "could not create semver constraint for Kubernetes version 1.13+")
	}

	kubeVersion, err := semver.NewVersion(kubernetesVersion)
	if err != nil {
		return "", errors.WithDetails(err, "could not create semver from Kubernetes version", "kubernetesVersion", kubernetesVersion)
	}

	switch {
	case constraint113.Check(kubeVersion):
		return map[string]string{
			"ap-northeast-1": "ami-02c668cbe82c52654",
			"ap-northeast-2": "ami-09a528e2d122f85dd",
			"ap-south-1":     "ami-0166fe96b7d5dd415",
			"ap-southeast-1": "ami-0a0dc38e1843a6652",
			"ap-southeast-2": "ami-08b4248beea1afd65",
			"ca-central-1":   "ami-0509103cc485dab39",
			"eu-central-1":   "ami-0ae595ec7057e4b0a",
			"eu-north-1":     "ami-09c5ee8e94a1dee05",
			"eu-west-1":      "ami-07eca1d40a370537e",
			"eu-west-2":      "ami-069b8d46ebd9cc918",
			"eu-west-3":      "ami-0cb47cb1e061df828",
			"sa-east-1":      "ami-0402072eb28dd87f1",
			"us-east-1":      "ami-05431b38d9884a58f",
			"us-east-2":      "ami-09a6009e74e5d5616",
			"us-west-1":      "ami-05732d634490519e4",
			"us-west-2":      "ami-0b971d5cf3d965807",
		}[region], nil
	case constraint114.Check(kubeVersion):
		return map[string]string{
			"ap-northeast-1": "ami-008a8db9911c3ab9d",
			"ap-northeast-2": "ami-0e560ec23c16ecc50",
			"ap-south-1":     "ami-08b549add41f232cc",
			"ap-southeast-1": "ami-078974aeb0b8b8ae0",
			"ap-southeast-2": "ami-0073123d17aef15b4",
			"ca-central-1":   "ami-0f69911233e05029a",
			"eu-central-1":   "ami-0d4b6d1c3e2fc46d8",
			"eu-north-1":     "ami-0ef8145828cbc2f39",
			"eu-west-1":      "ami-0a95b955c65b32fff",
			"eu-west-2":      "ami-0bf3be45c19287c71",
			"eu-west-3":      "ami-0b11a2bd7cf2da96d",
			"sa-east-1":      "ami-0a9fc6325c2f5181a",
			"us-east-1":      "ami-0055d58addfe009d0",
			"us-east-2":      "ami-0758a65bfe14b4e62",
			"us-west-1":      "ami-0bd1e43e4a7280c2c",
			"us-west-2":      "ami-06797d1f3e822ba1f",
		}[region], nil
	default:
		return map[string]string{
			"ap-northeast-1": "ami-0b5d11ef377baca40",
			"ap-northeast-2": "ami-0aac730552ec22cb1",
			"ap-south-1":     "ami-002d6833390304363",
			"ap-southeast-1": "ami-03e54e36d04bed301",
			"ap-southeast-2": "ami-002db8ca98bab2a7e",
			"ca-central-1":   "ami-06dbcc8c9c07ecf17",
			"eu-central-1":   "ami-00181e08e0300ee8f",
			"eu-north-1":     "ami-0e2566173c70f9043",
			"eu-west-1":      "ami-07ee7cde289dd3798",
			"eu-west-2":      "ami-001a496865332c672",
			"eu-west-3":      "ami-035002f73b31a49b7",
			"sa-east-1":      "ami-08b3b0b44b1efcc62",
			"us-east-1":      "ami-0af44092dcbe1c0c7",
			"us-east-2":      "ami-09688d5398a56d7c4",
			"us-west-1":      "ami-084bff033df1b4cf8",
			"us-west-2":      "ami-0d1a9391f028146bc",
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

	// Get default security group of the VPC
	var vpcDefaultSecurityGroupID string

	activityInput := GetVpcDefaultSecurityGroupActivityInput{
		AWSActivityInput: awsActivityInput,
		ClusterID:        input.ClusterID,
		VpcID:            vpcOutput["VpcId"],
	}
	err := workflow.ExecuteActivity(ctx, GetVpcDefaultSecurityGroupActivityName, activityInput).Get(ctx, &vpcDefaultSecurityGroupID)
	if err != nil {
		return err
	}

	if vpcDefaultSecurityGroupID == "" {
		return errors.Errorf("couldn't get the default security group of the VPC %q", vpcOutput["VpcId"])
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

	masterNodeSubnetID := strings.Split(vpcOutput["SubnetIds"], ",")[0]
	if len(master.Subnets) > 0 {
		masterNodeSubnetID = master.Subnets[0]
	}
	masterInput := CreateMasterActivityInput{
		ClusterID:                 input.ClusterID,
		VPCID:                     vpcOutput["VpcId"],
		VPCDefaultSecurityGroupID: vpcDefaultSecurityGroupID,
		SubnetID:                  masterNodeSubnetID,
		MultiMaster:               multiMaster,
		MasterInstanceProfile:     rolesOutput["MasterInstanceProfile"],
		ExternalBaseUrl:           input.PipelineExternalURL,
		ExternalBaseUrlInsecure:   input.PipelineExternalURLInsecure,
		Pool:                      master,
		SSHKeyName:                keyOut.KeyName,
		AvailabilityZone:          master.AvailabilityZones[0],
	}

	if multiMaster {
		// Create NLB
		var activityOutput CreateNLBActivityOutput
		activityInput := &CreateNLBActivityInput{
			AWSActivityInput: awsActivityInput,
			ClusterID:        input.ClusterID,
			ClusterName:      input.ClusterName,
			VPCID:            vpcOutput["VpcId"],
			SubnetIds:        []string{masterNodeSubnetID},
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
		futures := make([]workflow.Future, len(nodePools))

		for i, np := range nodePools {
			if !np.Master {
				subnetID := strings.Split(vpcOutput["SubnetIds"], ",")[0]

				createWorkerPoolActivityInput := CreateWorkerPoolActivityInput{
					ClusterID:                 input.ClusterID,
					Pool:                      np,
					WorkerInstanceProfile:     rolesOutput["WorkerInstanceProfile"],
					VPCID:                     vpcOutput["VpcId"],
					VPCDefaultSecurityGroupID: vpcDefaultSecurityGroupID,
					SubnetID:                  subnetID,
					ClusterSecurityGroup:      masterOutput["ClusterSecurityGroup"],
					ExternalBaseUrl:           input.PipelineExternalURL,
					ExternalBaseUrlInsecure:   input.PipelineExternalURLInsecure,
					SSHKeyName:                keyOut.KeyName,
				}

				futures[i] = workflow.ExecuteActivity(ctx, CreateWorkerPoolActivityName, createWorkerPoolActivityInput)
			}
		}

		errs := make([]error, len(futures))
		for i, future := range futures {
			if future != nil {
				errs[i] = errors.Wrapf(future.Get(ctx, nil), "couldn't create nodepool %q", nodePools[i].Name)
			}
		}

		return errors.Combine(errs...)
	}
}
