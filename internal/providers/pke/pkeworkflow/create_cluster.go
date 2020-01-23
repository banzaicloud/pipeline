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

	"github.com/banzaicloud/pipeline/pkg/cloudinfo"
)

const CreateClusterWorkflowName = "pke-create-cluster"
const pkeVersion = "0.4.19"

func getDefaultImageID(region, kubernetesVersion, pkeVersion string, cloudInfoClient *cloudinfo.Client) (string, error) {
	kubeVersion, err := semver.NewVersion(kubernetesVersion)
	if err != nil {
		return "", errors.WithDetails(err, "could not create semver from Kubernetes version", "kubernetesVersion", kubernetesVersion)
	}
	_ = kubeVersion

	if cloudInfoClient != nil {
		ami, err := cloudInfoClient.PKEImageName("amazon", "pke", "ubuntu", kubeVersion.String(), pkeVersion, region)
		if err != nil {
			// fail silently
		}
		if ami != "" {
			return ami, nil
		}
	}

	// PKE 0.4.19; K8s 1.13.10; OS Ubuntu
	return map[string]string{
		"ap-east-1":      "ami-0ca8206236662e9ea", // Asia Pacific (Hong Kong).
		"ap-northeast-1": "ami-029f1fff7d250aa95", // Asia Pacific (Tokyo).
		"ap-northeast-2": "ami-0b2ea3e1fb7e0a0dc", // Asia Pacific (Seoul).
		"ap-southeast-1": "ami-00d5d224c11f12854", // Asia Pacific (Singapore).
		"ap-southeast-2": "ami-03ad7f293fb551d91", // Asia Pacific (Sydney).
		"ap-south-1":     "ami-03f5be5363911cfd7", // Asia Pacific (Mumbai).
		"ca-central-1":   "ami-0f45e7a3348941cd0", // Canada (Central).
		"eu-central-1":   "ami-01a9d881b5eef8c78", // EU (Frankfurt).
		"eu-north-1":     "ami-0152ce8be8bcc0c50", // EU (Stockholm).
		"eu-west-1":      "ami-0284019fcb7ca3121", // EU (Ireland).
		"eu-west-2":      "ami-0b4c70c59b14d97ba", // EU (London).
		"eu-west-3":      "ami-084d0cc1bce975f4b", // EU (Paris).
		"me-south-1":     "ami-0bab0d9b6142d8674", // Middle East (Bahrain).
		"sa-east-1":      "ami-025c97e09ba50cb05", // South America (Sao Paulo).
		"us-east-1":      "ami-0980e83ac34bbc3bb", // US East (N. Virginia).
		"us-east-2":      "ami-07d087f4be161fa72", // US East (Ohio).
		"us-west-1":      "ami-0c8ea5996aa18c15d", // US West (N. California).
		"us-west-2":      "ami-0d441df4104cb772b", // US West (Oregon).
	}[region], nil
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
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    20 * time.Minute,
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
