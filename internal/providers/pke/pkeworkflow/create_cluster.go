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
			"ap-northeast-1": "ami-03c85d7b693274ab2",
			"ap-northeast-2": "ami-0c862483d46858388",
			"ap-southeast-1": "ami-0313d518301238064",
			"ap-southeast-2": "ami-07051cdb34f3ccd13",
			"ap-south-1":     "ami-0a13e0ed5cf6d56d1",
			"ca-central-1":   "ami-0046d50c09c3fa196",
			"eu-central-1":   "ami-04e248d5671469fab",
			"eu-north-1":     "ami-0fb5f2dccf1f9e280",
			"eu-west-1":      "ami-0cde1c7fb5445c1eb",
			"eu-west-2":      "ami-02c35d4bb7da231d9",
			"eu-west-3":      "ami-06fa9d5c99b3636fe",
			"sa-east-1":      "ami-0a070ff1e7d34b50b",
			"us-east-1":      "ami-0995a0271a0703eee",
			"us-east-2":      "ami-0976011aebf06a743",
			"us-west-1":      "ami-0b25be32a4e31c64f",
			"us-west-2":      "ami-05d8f2797251e0434",
		}[region], nil
	case constraint114.Check(kubeVersion):
		return map[string]string{
			"ap-northeast-1": "ami-02a69e3257cc89d5e",
			"ap-northeast-2": "ami-054caf9bfbd30b209",
			"ap-southeast-1": "ami-087faa6ef575cfbee",
			"ap-southeast-2": "ami-02970f1e1230ac498",
			"ap-south-1":     "ami-0a91bc117ce743304",
			"ca-central-1":   "ami-0886d8b66b20a286e",
			"eu-central-1":   "ami-09bf0e50559b54472",
			"eu-north-1":     "ami-06e648a4174d7089f",
			"eu-west-1":      "ami-0db9a69297f143fa5",
			"eu-west-2":      "ami-03fc86f273d9e9220",
			"eu-west-3":      "ami-04b2392b0e3c304f3",
			"sa-east-1":      "ami-029bfd553809f5afc",
			"us-east-1":      "ami-0f67d3c9300e86e84",
			"us-east-2":      "ami-05dc7aa00467ff508",
			"us-west-1":      "ami-07ec96a1f82619388",
			"us-west-2":      "ami-0faa0b56e538023d1",
		}[region], nil
	case constraint115.Check(kubeVersion):
		return map[string]string{
			"ap-northeast-1": "ami-0ff8257dbc74c33f4",
			"ap-northeast-2": "ami-05654b3b1fe2c442b",
			"ap-southeast-1": "ami-072f0303ff596ec59",
			"ap-southeast-2": "ami-0dca60ba1f804b73a",
			"ap-south-1":     "ami-037de328d01274ce7",
			"ca-central-1":   "ami-02358ac7f3c5dd608",
			"eu-central-1":   "ami-03e96adc107f88c8b",
			"eu-north-1":     "ami-0dc0929c435baf7e1",
			"eu-west-1":      "ami-0bc8422ca744df23e",
			"eu-west-2":      "ami-0885ccbf224e38d3b",
			"eu-west-3":      "ami-015cd31c41067a91e",
			"sa-east-1":      "ami-0af9ba56e4e7b51a1",
			"us-east-1":      "ami-06ebb5813a6ec2be3",
			"us-east-2":      "ami-09c84ffdb343309d6",
			"us-west-1":      "ami-0fc837638f27f5ebf",
			"us-west-2":      "ami-07e9bf3372342dc02",
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
