// Copyright © 2020 Banzai Cloud
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
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
	"github.com/banzaicloud/pipeline/pkg/sdk/cadence/lib/pipeline/processlog"
)

const (
	CreateClusterWorkflowName = "pke-create-cluster"
	errorSignalName           = "node-bootstrap-failed"
	readySignalName           = "node-ready"
)

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
}

type CreateClusterWorkflow struct {
	DefaultNodeVolumeSize int
	GlobalRegion          string
	processLogger         processlog.ProcessLogger
}

func NewCreateClusterWorkflow(defaultNodeVolumeSize int, globalRegion string) CreateClusterWorkflow {
	return CreateClusterWorkflow{
		DefaultNodeVolumeSize: defaultNodeVolumeSize,
		GlobalRegion:          globalRegion,
		processLogger:         processlog.New(),
	}
}

func (w CreateClusterWorkflow) Execute(ctx workflow.Context, input CreateClusterWorkflowInput) (err error) {
	clusterID := brn.New(input.OrganizationID, brn.ClusterResourceType, fmt.Sprint(input.ClusterID))
	process := w.processLogger.StartProcess(ctx, clusterID.String())
	defer func() {
		process.Finish(ctx, err)
	}()

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    20 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", ErrReasonStackFailed},
		},
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
		activityInput.AWSActivityInput.Region = w.GlobalRegion
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
		activityInput.AWSActivityInput.Region = w.GlobalRegion

		err := workflow.ExecuteActivity(ctx, WaitCFCompletionActivityName, activityInput).Get(ctx, &rolesOutput)
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

	err = forEachNodePool(nodePools, func(nodePool *NodePool) (err error) {
		nodePool.ImageID, nodePool.VolumeSize, err = SelectImageAndVolumeSize(
			ctx,
			awsActivityInput,
			input.ClusterID,
			nodePool.InstanceType,
			nodePool.ImageID,
			nodePool.VolumeSize,
			w.DefaultNodeVolumeSize,
		)

		return err
	})
	if err != nil {
		return err
	}

	// Collect all AZs
	var master NodePool
	for _, np := range nodePools {
		if np.Master {
			master = np
			if len(np.AvailabilityZones) == 0 || np.AvailabilityZones[0] == "" {
				return errors.NewWithDetails("missing availability zone for nodepool %q", np.Name)
			}
			break
		}
	}
	// Collect relevant AZs from NodePools without subnets
	availabilityZoneSet := make(map[string]struct{})
	for _, np := range nodePools {
		// We only look AZ when no subnet is set
		if len(np.Subnets) == 0 {
			for _, az := range np.AvailabilityZones {
				availabilityZoneSet[az] = struct{}{}
			}
		}
	}
	// Create AZ and Address map
	availabilityZoneMap := make(map[string]string, len(availabilityZoneSet))
	id := 0
	for zone := range availabilityZoneSet {
		availabilityZoneMap[zone] = fmt.Sprintf("192.168.%d.0/20", id*16)
		id++
	}

	var vpcStackID string

	// Create VPC
	{
		activityInput := CreateVPCActivityInput{
			AWSActivityInput: awsActivityInput,
			ClusterID:        input.ClusterID,
			ClusterName:      input.ClusterName,
			VPCID:            input.VPCID,
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
	err = workflow.ExecuteActivity(ctx, GetVpcDefaultSecurityGroupActivityName, activityInput).Get(ctx, &vpcDefaultSecurityGroupID)
	if err != nil {
		return err
	}

	if vpcDefaultSecurityGroupID == "" {
		return errors.Errorf("couldn't get the default security group of the VPC %q", vpcOutput["VpcId"])
	}

	// Create subnets map nodepools with subnet?
	// We need a Zone - SubnetID map
	subnetIDMap := make(map[string]string)
	{
		var createSubnetFutures []workflow.Future
		for zone, ip := range availabilityZoneMap {
			activityInput := CreateSubnetActivityInput{

				AWSActivityInput: awsActivityInput,
				ClusterID:        input.ClusterID,
				ClusterName:      input.ClusterName,
				VpcID:            vpcOutput["VpcId"],
				RouteTableID:     vpcOutput["RouteTableId"],
				Cidr:             ip,
				AvailabilityZone: zone,
			}
			ctx := workflow.WithActivityOptions(ctx, ao)
			createSubnetFutures = append(createSubnetFutures, workflow.ExecuteActivity(ctx, CreateSubnetActivityName, activityInput))
		}

		// wait for info about newly created subnets
		errs := make([]error, len(createSubnetFutures))
		for i, future := range createSubnetFutures {
			var activityOutput CreateSubnetActivityOutput

			errs[i] = future.Get(ctx, &activityOutput)
			if errs[i] == nil {
				subnetIDMap[activityOutput.AvailabilityZone] = activityOutput.SubnetID
			}
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

	subnetIDs := master.Subnets
	if len(master.Subnets) == 0 {
		for _, az := range master.AvailabilityZones {
			subnetIDs = append(subnetIDs, subnetIDMap[az])
		}
	}

	masterInput := CreateMasterActivityInput{
		ClusterID:                 input.ClusterID,
		VPCID:                     vpcOutput["VpcId"],
		VPCDefaultSecurityGroupID: vpcDefaultSecurityGroupID,
		SubnetIDs:                 subnetIDs,
		MultiMaster:               multiMaster,
		MasterInstanceProfile:     rolesOutput["MasterInstanceProfile"],
		ExternalBaseUrl:           input.PipelineExternalURL,
		ExternalBaseUrlInsecure:   input.PipelineExternalURLInsecure,
		Pool:                      master,
		SSHKeyName:                keyOut.KeyName,
	}

	if multiMaster {
		// Create NLB
		var activityOutput CreateNLBActivityOutput
		activityInput := &CreateNLBActivityInput{
			AWSActivityInput: awsActivityInput,
			ClusterID:        input.ClusterID,
			ClusterName:      input.ClusterName,
			VPCID:            vpcOutput["VpcId"],
			SubnetIds:        subnetIDs,
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
			Subnets:         subnetIDs,
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

	if err := waitForMasterReadySignal(ctx, 1*time.Hour); err != nil {
		return err
	}

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
				subnetIDs := np.Subnets
				if len(np.Subnets) == 0 {
					for _, az := range np.AvailabilityZones {
						subnetIDs = append(subnetIDs, subnetIDMap[az])
					}
				}

				createWorkerPoolActivityInput := CreateWorkerPoolActivityInput{
					ClusterID:                 input.ClusterID,
					Pool:                      np,
					WorkerInstanceProfile:     rolesOutput["WorkerInstanceProfile"],
					VPCID:                     vpcOutput["VpcId"],
					VPCDefaultSecurityGroupID: vpcDefaultSecurityGroupID,
					SubnetIDs:                 subnetIDs,
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

type decodableError struct {
	Message string
}

func (d decodableError) Error() string {
	return d.Message
}

func waitForMasterReadySignal(ctx workflow.Context, timeout time.Duration) error {
	signalChan := workflow.GetSignalChannel(ctx, readySignalName)
	errSignalChan := workflow.GetSignalChannel(ctx, errorSignalName)
	signalTimeoutTimer := workflow.NewTimer(ctx, timeout)
	signalTimeout := false

	var signalValue decodableError
	signalSelector := workflow.NewSelector(ctx).AddReceive(errSignalChan, func(c workflow.Channel, more bool) {
		c.Receive(ctx, &signalValue)
	}).AddReceive(signalChan, func(c workflow.Channel, more bool) {
		c.Receive(ctx, nil)
	}).AddFuture(signalTimeoutTimer, func(workflow.Future) {
		signalTimeout = true
	})

	signalSelector.Select(ctx) // wait for signal

	if signalTimeout {
		return fmt.Errorf("timeout while waiting for signal")
	}
	if signalValue.Error() != "" {
		return errors.Wrap(signalValue, "failed to start master node")
	}
	return nil
}
