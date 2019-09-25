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

package action

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	pkgEC2 "github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"logur.dev/logur"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

// EksSubnet describe the properties of subnet
type EksSubnet struct {
	// The AWS ID of the subnet
	SubnetID string

	// The CIDR of the subnet
	Cidr string

	// The availability zone of the subnet
	AvailabilityZone string
}

func getVPCStackTags(clusterName string) []*cloudformation.Tag {
	return getStackTags(clusterName, "vpc")
}

func getSubnetStackTags(clusterName string) []*cloudformation.Tag {
	return getStackTags(clusterName, "subnet")
}

// --

var _ utils.RevocableAction = (*CreateVPCAndRolesAction)(nil)

// CreateVPCAndRolesAction describes the properties of a VPC creation
type CreateVPCAndRolesAction struct {
	context   *EksClusterCreateUpdateContext
	stackName string
	log       logrus.FieldLogger
}

// NewCreateVPCAndRolesAction creates a new CreateVPCAndRolesAction
func NewCreateVPCAndRolesAction(log logrus.FieldLogger, creationContext *EksClusterCreateUpdateContext, stackName string) *CreateVPCAndRolesAction {
	return &CreateVPCAndRolesAction{
		context:   creationContext,
		stackName: stackName,
		log:       log,
	}
}

// GetName returns the name of this CreateVPCAndRolesAction
func (a *CreateVPCAndRolesAction) GetName() string {
	return "CreateVPCAndRolesAction"
}

// ExecuteAction executes this CreateVPCAndRolesAction
func (a *CreateVPCAndRolesAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Infoln("EXECUTE CreateVPCAndRolesAction, stack name:", a.stackName)

	a.log.Infoln("getting CloudFormation template for creating VPC for EKS cluster")
	templateBody, err := eks.GetVPCTemplate()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get CloudFormation template for VPC")
	}

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: aws.String(a.context.ClusterName),
		},
	}

	if aws.StringValue(a.context.VpcID) != "" {
		a.log.Infoln("skip creating new VPC, using VPC: ", *a.context.VpcID)

		stackParams = append(stackParams,
			&cloudformation.Parameter{
				ParameterKey:   aws.String("VpcId"),
				ParameterValue: a.context.VpcID,
			})

		if aws.StringValue(a.context.RouteTableID) != "" {
			stackParams = append(stackParams,
				&cloudformation.Parameter{
					ParameterKey:   aws.String("RouteTableId"),
					ParameterValue: a.context.RouteTableID,
				})
		}

	} else if aws.StringValue(a.context.VpcCidr) != "" {
		stackParams = append(stackParams,
			&cloudformation.Parameter{
				ParameterKey:   aws.String("VpcBlock"),
				ParameterValue: a.context.VpcCidr,
			})
	}

	cloudformationSrv := cloudformation.New(a.context.Session)

	createStackInput := &cloudformation.CreateStackInput{
		ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
		DisableRollback:    aws.Bool(true),
		Capabilities: []*string{
			aws.String(cloudformation.CapabilityCapabilityIam),
			aws.String(cloudformation.CapabilityCapabilityNamedIam),
		},
		StackName:        aws.String(a.stackName),
		Parameters:       stackParams,
		Tags:             getVPCStackTags(a.context.ClusterName),
		TemplateBody:     aws.String(templateBody),
		TimeoutInMinutes: aws.Int64(10),
	}
	_, err = cloudformationSrv.CreateStack(createStackInput)
	if err != nil {
		return nil, errors.WrapIf(err, "create stack failed")
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(a.stackName)}
	err = cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput)
	if err != nil {
		return nil, pkgCloudformation.NewAwsStackFailure(err, a.stackName, cloudformationSrv)
	}

	describeStacksOutput, err := cloudformationSrv.DescribeStacks(describeStacksInput)
	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		switch aws.StringValue(output.OutputKey) {
		case "VpcId":
			a.context.VpcID = output.OutputValue
		case "RouteTableId":
			a.context.RouteTableID = output.OutputValue
		}
	}
	return nil, nil
}

// UndoAction rolls back this CreateVPCAndRolesAction
func (a *CreateVPCAndRolesAction) UndoAction() (err error) {
	a.log.Infoln("EXECUTE UNDO CreateVPCAndRolesAction, deleting stack:", a.stackName)
	cloudformationSrv := cloudformation.New(a.context.Session)
	deleteStackInput := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
		StackName:          aws.String(a.stackName),
	}
	_, err = cloudformationSrv.DeleteStack(deleteStackInput)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == cloudformation.ErrCodeStackInstanceNotFoundException {
				return nil
			}
		}
	}

	return err
}

var _ utils.Action = (*CreateSubnetsAction)(nil)

// CreateSubnetsAction describes the properties of the action for creating subnets
type CreateSubnetsAction struct {
	context                *EksClusterCreateUpdateContext
	cloudFormationTemplate string

	log logur.Logger
}

// NewCreateSubnetsAction creates a new CreateSubnetsAction
func NewCreateSubnetsAction(logger logur.Logger, creationContext *EksClusterCreateUpdateContext, cloudFormationTemplate string) *CreateSubnetsAction {
	return &CreateSubnetsAction{
		context:                creationContext,
		cloudFormationTemplate: cloudFormationTemplate,
		log:                    logger,
	}
}

// GetName returns the name of this CreateSubnetsAction
func (a *CreateSubnetsAction) GetName() string {
	return "CreateSubnetsAction"
}

// ExecuteAction executes this CreateSubnetsAction
func (a *CreateSubnetsAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Info("EXECUTE CreateSubnetsAction")

	netSvc := pkgEC2.NewNetworkSvc(ec2.New(a.context.Session), a.log)

	var subnetIds []string
	for _, subnet := range a.context.Subnets {
		if subnet.SubnetID != "" {
			subnetIds = append(subnetIds, subnet.SubnetID)
		}
	}
	ec2Subnets, err := netSvc.GetSubnetsById(subnetIds)
	if err != nil {
		return nil, errors.WrapIf(err, "couldn't get subnet details")
	}

	outputChan := make(chan struct {
		*EksSubnet
		error
	}, len(a.context.Subnets))
	defer close(outputChan)

	waitRoutines := 0
	for _, subnet := range a.context.Subnets {
		if subnet.SubnetID == "" && subnet.Cidr != "" {
			waitRoutines++
			go a.createSubnet(subnet.Cidr, subnet.AvailabilityZone, outputChan)
		}
	}

	// wait for go routines to finish
	var errs []error
	var subnets []*EksSubnet
	for i := 0; i < waitRoutines; i++ {
		output := <-outputChan
		if output.error != nil {
			errs = append(errs, output.error)
			continue
		}
		subnets = append(subnets, output.EksSubnet)
	}

	err = errors.Combine(errs...)
	if err != nil {
		return nil, err
	}

	// pass back the details of the newly created subnets
	for _, subnet := range subnets {
		for i := range a.context.Subnets {
			if a.context.Subnets[i].SubnetID == "" && a.context.Subnets[i].Cidr == subnet.Cidr {
				a.context.Subnets[i].SubnetID = subnet.SubnetID
				break
			}
		}
	}

	// pass back the details of existing subnets
	for _, subnet := range ec2Subnets {
		for i := range a.context.Subnets {
			if a.context.Subnets[i].SubnetID != "" && a.context.Subnets[i].SubnetID == aws.StringValue(subnet.SubnetId) {
				a.context.Subnets[i].Cidr = aws.StringValue(subnet.CidrBlock)
				a.context.Subnets[i].AvailabilityZone = aws.StringValue(subnet.AvailabilityZone)
			}
		}
	}

	return nil, nil
}

func (a *CreateSubnetsAction) createSubnet(cidr, az string, output chan struct {
	*EksSubnet
	error
}) {
	cloudformationSrv := cloudformation.New(a.context.Session)

	r := strings.NewReplacer(".", "-", "/", "-")
	stackName := fmt.Sprintf("pipeline-eks-subnet-%s-%s", a.context.ClusterName, r.Replace(cidr))

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("VpcId"),
			ParameterValue: a.context.VpcID,
		},
		{
			ParameterKey:   aws.String("RouteTableId"),
			ParameterValue: a.context.RouteTableID,
		},
		{
			ParameterKey:   aws.String("SubnetBlock"),
			ParameterValue: aws.String(cidr),
		},
		{
			ParameterKey:   aws.String("AvailabilityZoneName"),
			ParameterValue: aws.String(az),
		},
	}

	a.log.Debug("creating subnet", map[string]interface{}{"cidr": cidr, "availabilityZone": az})

	createStackInput := &cloudformation.CreateStackInput{
		ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
		DisableRollback:    aws.Bool(true),
		StackName:          aws.String(stackName),
		Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		Parameters:         stackParams,
		Tags:               getSubnetStackTags(a.context.ClusterName),
		TemplateBody:       aws.String(a.cloudFormationTemplate),
		TimeoutInMinutes:   aws.Int64(10),
	}

	_, err := cloudformationSrv.CreateStack(createStackInput)
	if err != nil {
		output <- struct {
			*EksSubnet
			error
		}{nil, errors.WrapIff(err, "failed to create subnet with cidr %q", cidr)}
		return
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}
	err = cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput)
	if err != nil {
		output <- struct {
			*EksSubnet
			error
		}{nil, errors.WrapIff(pkgCloudformation.NewAwsStackFailure(err, stackName, cloudformationSrv), "failed to create subnet with cidr %q", cidr)}
		return
	}

	describeStacksOutput, err := cloudformationSrv.DescribeStacks(describeStacksInput)
	if err != nil {
		output <- struct {
			*EksSubnet
			error
		}{nil, errors.WrapIf(err, "failed to retrieve subnet id")}
		return
	}

	var subnetId string
	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		switch aws.StringValue(output.OutputKey) {
		case "SubnetId":
			subnetId = aws.StringValue(output.OutputValue)
		}
	}

	a.log.Debug("subnet successfully created", map[string]interface{}{"cidr": cidr, "availabilityZone": az, "subnetId": subnetId})
	output <- struct {
		*EksSubnet
		error
	}{&EksSubnet{SubnetID: subnetId, Cidr: cidr, AvailabilityZone: az}, nil}
}

var _ utils.Action = (*DeleteOrphanNICsAction)(nil)

// DeleteOrphanNICsAction describes the properties for deleting orphaned
// ENIs left behind by the CNI driver
type DeleteOrphanNICsAction struct {
	context *EksClusterDeletionContext
	log     logur.Logger
}

// NewDeleteOrphanNICsAction creates action for deleting orphaned ENIs
func NewDeleteOrphanNICsAction(logger logur.Logger, context *EksClusterDeletionContext) *DeleteOrphanNICsAction {
	return &DeleteOrphanNICsAction{
		context: context,
		log:     logger,
	}
}

// GetName returns the name of this DeleteOrphanNICsAction
func (a *DeleteOrphanNICsAction) GetName() string {
	return "DeleteOrphanNICsAction"
}

// ExecuteAction deletes all orphaned network interfaces
func (a *DeleteOrphanNICsAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Info("EXECUTE DeleteOrphanNICsAction")

	if a.context.VpcID == "" || len(a.context.SecurityGroupIDs) == 0 {
		return nil, nil
	}

	netSvc := pkgEC2.NewNetworkSvc(ec2.New(a.context.Session), a.log)

	// collect orphan ENIs
	// CNI plugin applies the following tags to ENIs https://aws.amazon.com/blogs/opensource/vpc-cni-plugin-v1-1-available/
	tagsFilter := map[string][]string{
		"node.k8s.amazonaws.com/instance_id": nil,
	}
	nics, err := netSvc.GetUnusedNetworkInterfaces(a.context.VpcID, a.context.SecurityGroupIDs, tagsFilter)
	if err != nil {
		return nil, errors.WrapIf(err, "searching for unused network interfaces failed")
	}

	errChan := make(chan error, len(nics))
	defer close(errChan)

	for _, nic := range nics {
		go a.deleteNetworkInterface(errChan, nic)
	}

	errs := make([]error, len(nics))
	for i := 0; i < len(nics); i++ {
		errs[i] = <-errChan
	}

	return nil, errors.Combine(errs...)
}

func (a *DeleteOrphanNICsAction) deleteNetworkInterface(errChan chan<- error, nicId string) {
	netSvc := pkgEC2.NewNetworkSvc(ec2.New(a.context.Session), a.log)

	a.log.Info("deleting network interface", map[string]interface{}{"nic": nicId})

	errChan <- netSvc.DeleteNetworkInterface(nicId)
}
