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
const pkeVersion = "0.4.14"

func getDefaultImageID(region, kubernetesVersion string) (string, error) {

	constraint113, err := semver.NewConstraint("~1.13.0")
	if err != nil {
		return "", errors.Wrap(err, "could not create semver constraint for Kubernetes version 1.13+")
	}

	constraint114, err := semver.NewConstraint("~1.14.0")
	if err != nil {
		return "", errors.Wrap(err, "could not create semver constraint for Kubernetes version 1.14+")
	}

	constraint115, err := semver.NewConstraint("~1.15.0")
	if err != nil {
		return "", errors.Wrap(err, "could not create semver constraint for Kubernetes version 1.15+")
	}

	kubeVersion, err := semver.NewVersion(kubernetesVersion)
	if err != nil {
		return "", errors.WithDetails(err, "could not create semver from Kubernetes version", "kubernetesVersion", kubernetesVersion)
	}

	switch {
	case constraint113.Check(kubeVersion):
		return map[string]string{
			"ap-east-1":      "ami-04cd6dc5c53a1f08c", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0ed7959a76acaa682", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0e81b2d55656a1191", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-09ede0029b72b3c33", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-011932d1e814b2ff6", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-087968379b6d38ad9", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-05cc925b34abdcbce", // Canada (Central).
			"eu-central-1":   "ami-08870347bf63fc0a9", // EU (Frankfurt).
			"eu-north-1":     "ami-0652a3bb23e943a7b", // EU (Stockholm).
			"eu-west-1":      "ami-05508eabfd52a5730", // EU (Ireland).
			"eu-west-2":      "ami-01f5d06890666ef1d", // EU (London).
			"eu-west-3":      "ami-0b0d81786e9237908", // EU (Paris).
			"me-south-1":     "ami-04960cff5909f3ded", // Middle East (Bahrain).
			"sa-east-1":      "ami-089207a5c493f704e", // South America (Sao Paulo)
			"us-east-1":      "ami-0a877f2c3f30f65bd", // US East (N. Virginia).
			"us-east-2":      "ami-02c3b3314bc411937", // US East (Ohio).
			"us-west-1":      "ami-0cfc4d351b908d353", // US West (N. California).
			"us-west-2":      "ami-040319fe85c2056a8", // US West (Oregon).
		}[region], nil
	case constraint114.Check(kubeVersion):
		return map[string]string{
			"ap-east-1":      "ami-05b7d0a24532c4fd7", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0e1b4c30b002f8e0d", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-08f4975c69ebfd30b", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-09ab46415bc4d60e6", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-08d5d96e915dcbd6e", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-053244ef9703cd00d", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-086454b3b6fede54a", // Canada (Central).
			"eu-central-1":   "ami-088dbc498c9bd9170", // EU (Frankfurt).
			"eu-north-1":     "ami-0520b1493fdd02f30", // EU (Stockholm).
			"eu-west-1":      "ami-04761c847f867ca28", // EU (Ireland).
			"eu-west-2":      "ami-0116519a435ccba1e", // EU (London).
			"eu-west-3":      "ami-057fe3a4eb5f3a315", // EU (Paris).
			"me-south-1":     "ami-004f42337db5f4bfa", // Middle East (Bahrain).
			"sa-east-1":      "ami-05b071461b07f2392", // South America (Sao Paulo)
			"us-east-1":      "ami-0bc17d9b8be975338", // US East (N. Virginia).
			"us-east-2":      "ami-0bad13677d32b0959", // US East (Ohio).
			"us-west-1":      "ami-0a3892fa4e09e0c84", // US West (N. California).
			"us-west-2":      "ami-02904d39ae2ed2a7e", // US West (Oregon).
		}[region], nil
	case constraint115.Check(kubeVersion):
		return map[string]string{
			"ap-east-1":      "ami-0c906c5a886224f2c", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-048cd136c9d3752d6", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-0c2517ca3fd4e157e", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0186d86b776b8dc8b", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-0af43abdd3d123f8a", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-02ba2b7be8eb99b34", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0644ae9cb4750efd5", // Canada (Central).
			"eu-central-1":   "ami-0ccf38993187a12ff", // EU (Frankfurt).
			"eu-north-1":     "ami-01ba2867145e54525", // EU (Stockholm).
			"eu-west-1":      "ami-0e91b1f208d945645", // EU (Ireland).
			"eu-west-2":      "ami-0fc75cd793034847b", // EU (London).
			"eu-west-3":      "ami-0a362008b150c6f60", // EU (Paris).
			"me-south-1":     "ami-0813b61c2db464b11", // Middle East (Bahrain).
			"sa-east-1":      "ami-01f61154ea7d72f00", // South America (Sao Paulo)
			"us-east-1":      "ami-09882d44c73fabe37", // US East (N. Virginia).
			"us-east-2":      "ami-03a405f0e64a8cfe0", // US East (Ohio).
			"us-west-1":      "ami-062b015f7ca2803f2", // US West (N. California).
			"us-west-2":      "ami-09e2678e579e5b06f", // US West (Oregon).
		}[region], nil
	default:
		return map[string]string{
			"ap-east-1":      "ami-0c9680acbf35f26de", // Asia Pacific (Hong Kong).
			"ap-northeast-1": "ami-0f13e8123146595b9", // Asia Pacific (Tokyo).
			"ap-northeast-2": "ami-021015b95e7bdfbbe", // Asia Pacific (Seoul).
			"ap-southeast-1": "ami-0382298e181ef5686", // Asia Pacific (Mumbai).
			"ap-southeast-2": "ami-068231c38bc1a60f3", // Asia Pacific (Singapore).
			"ap-south-1":     "ami-016ec067d44808c4f", // Asia Pacific (Sydney).
			"ca-central-1":   "ami-0e06edb0102874198", // Canada (Central).
			"eu-central-1":   "ami-0ec8d2a455affc7e4", // EU (Frankfurt).
			"eu-north-1":     "ami-067696d723418ef5e", // EU (Stockholm).
			"eu-west-1":      "ami-0214421b4d7aaecdd", // EU (Ireland).
			"eu-west-2":      "ami-08576e40ab2877d2a", // EU (London).
			"eu-west-3":      "ami-0cb72921b642a83ec", // EU (Paris).
			"me-south-1":     "ami-0f5484e06a055b46d", // Middle East (Bahrain).
			"sa-east-1":      "ami-08d90516f7c661b6b", // South America (Sao Paulo).
			"us-east-1":      "ami-07079058aa890ee37", // US East (N. Virginia).
			"us-east-2":      "ami-0faf98ec1c0e28a7e", // US East (Ohio).
			"us-west-1":      "ami-0bef95b814eae1fc7", // US West (N. California).
			"us-west-2":      "ami-0ca6e0198325b7be7", // US West (Oregon).
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
	OIDCEnabled                 bool
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
	if input.OIDCEnabled {
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
