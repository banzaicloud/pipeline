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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon/autoscaling"
	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
	"github.com/banzaicloud/pipeline/utils"
)

var _ utils.RevocableAction = (*CreateUpdateNodePoolStackAction)(nil)

func getNodePoolStackTags(clusterName string) []*cloudformation.Tag {
	return getStackTags(clusterName, "nodepool")
}

// CreateUpdateNodePoolStackAction describes the properties of a node pool creation
type CreateUpdateNodePoolStackAction struct {
	context          *EksClusterCreateUpdateContext
	isCreate         bool
	nodePools        []*model.AmazonNodePoolsModel
	log              logrus.FieldLogger
	waitAttempts     int
	waitInterval     time.Duration
	headNodePoolName string
	nodePoolTemplate string
	subnetMapping    map[string][]*EksSubnet
}

// NewCreateUpdateNodePoolStackAction creates a new CreateUpdateNodePoolStackAction
func NewCreateUpdateNodePoolStackAction(
	log logrus.FieldLogger,
	isCreate bool,
	creationContext *EksClusterCreateUpdateContext,
	waitAttempts int,
	waitInterval time.Duration,
	nodePoolTemplate string,
	subnetMapping map[string][]*EksSubnet,
	headNodePoolName string,
	nodePools ...*model.AmazonNodePoolsModel) *CreateUpdateNodePoolStackAction {
	return &CreateUpdateNodePoolStackAction{
		context:          creationContext,
		isCreate:         isCreate,
		nodePools:        nodePools,
		log:              log,
		waitAttempts:     waitAttempts,
		waitInterval:     waitInterval,
		headNodePoolName: headNodePoolName,
		nodePoolTemplate: nodePoolTemplate,
		subnetMapping:    subnetMapping,
	}
}

func (a *CreateUpdateNodePoolStackAction) generateStackName(nodePool *model.AmazonNodePoolsModel) string {
	return GenerateNodePoolStackName(a.context.ClusterName, nodePool.Name)
}

// GetName return the name of this action
func (a *CreateUpdateNodePoolStackAction) GetName() string {
	return "CreateUpdateNodePoolStackAction"
}

// WaitForASGToBeFulfilled waits until an ASG has the desired amount of healthy nodes
func (a *CreateUpdateNodePoolStackAction) WaitForASGToBeFulfilled(ctx context.Context, nodePool *model.AmazonNodePoolsModel) error {
	return WaitForASGToBeFulfilled(ctx, a.context.Session, a.log, a.context.ClusterName, nodePool.Name, a.waitAttempts, a.waitInterval)
}

// WaitForASGToBeFulfilled waits until an ASG has the desired amount of healthy nodes
func WaitForASGToBeFulfilled(
	ctx context.Context,
	awsSession *session.Session,
	logger logrus.FieldLogger,
	clusterName string,
	nodePoolName string,
	waitAttempts int,
	waitInterval time.Duration) error {

	m := autoscaling.NewManager(awsSession, autoscaling.MetricsEnabled(true), autoscaling.Logger{
		FieldLogger: logger,
	})
	asgName := GenerateNodePoolStackName(clusterName, nodePoolName)
	log := logger.WithField("asg-name", asgName)
	log.WithFields(logrus.Fields{
		"attempts": waitAttempts,
		"interval": waitInterval,
	}).Info("EXECUTE WaitForASGToBeFulfilled")

	ticker := time.NewTicker(waitInterval)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-ticker.C:
			if i <= waitAttempts {
				waitAttempts++

				asGroup, err := m.GetAutoscalingGroupByStackName(asgName)
				if err != nil {
					if aerr, ok := err.(awserr.Error); ok {
						if aerr.Code() == "ValidationError" || aerr.Code() == "ASGNotFoundInResponse" {
							continue
						}
					}
					return errors.WrapIfWithDetails(err, "could not get ASG", "asg-name", asgName)
				}

				ok, err := asGroup.IsHealthy()
				if err != nil {
					if autoscaling.IsErrorFinal(err) {
						return errors.WrapIfWithDetails(err, nodePoolName, "nodePoolName", nodePoolName, "asgName", *asGroup.AutoScalingGroupName)
					}
					log.Debug(err)
					continue
				}
				if ok {
					log.Debug("ASG is healthy")
					return nil
				}
			} else {
				return errors.Errorf("waiting for ASG to be fulfilled timed out after %d x %s", waitAttempts, waitInterval)
			}
		case <-ctx.Done(): // wait for ASG fulfillment cancelled
			return nil
		}
	}

}

// ExecuteAction executes the CreateUpdateNodePoolStackAction in parallel for each node pool
func (a *CreateUpdateNodePoolStackAction) ExecuteAction(input interface{}) (output interface{}, err error) {

	errorChan := make(chan error, len(a.nodePools))
	defer close(errorChan)

	for _, nodePool := range a.nodePools {
		go a.createUpdateNodePool(nodePool, errorChan)
	}

	var errs []error
	// wait for goroutines to finish
	for i := 0; i < len(a.nodePools); i++ {
		errs = append(errs, <-errorChan)
	}

	return nil, errors.Combine(errs...)
}

func (a *CreateUpdateNodePoolStackAction) createUpdateNodePool(nodePool *model.AmazonNodePoolsModel, errorChan chan error) {
	log := a.log.WithField("nodePool", nodePool.Name)

	stackName := a.generateStackName(nodePool)

	if a.isCreate {
		log.Infoln("EXECUTE CreateUpdateNodePoolStackAction, create stack name:", stackName)
	} else {
		log.Infoln("EXECUTE CreateUpdateNodePoolStackAction, update stack name:", stackName)
	}

	spotPriceParam := ""
	if p, err := strconv.ParseFloat(nodePool.NodeSpotPrice, 64); err == nil && p > 0.0 {
		spotPriceParam = nodePool.NodeSpotPrice
	}

	clusterAutoscalerEnabled := false
	terminationDetachEnabled := false

	if nodePool.Autoscaling {
		clusterAutoscalerEnabled = true
	}

	// if ScaleOptions is enabled on cluster, ClusterAutoscaler is disabled on all node pools, except head
	if a.context.ScaleEnabled {
		if nodePool.Name != a.headNodePoolName {
			clusterAutoscalerEnabled = false
			terminationDetachEnabled = true
		}
	}

	waitOnCreateUpdate := true

	cloudformationSrv := cloudformation.New(a.context.Session)

	tags := getNodePoolStackTags(a.context.ClusterName)
	var stackParams []*cloudformation.Parameter

	// create stack
	if a.isCreate {
		// do not update node labels via kubelet boostrap params as that induces node reboot or replacement
		// we only add node pool name here, all other labels will be added by NodePoolLabelSet operator
		nodeLabels := []string{
			fmt.Sprintf("%v=%v", common.LabelKey, nodePool.Name),
		}

		subnets, ok := a.subnetMapping[nodePool.Name]
		if !ok {
			errorChan <- errors.Errorf("there is no subnet mapping defined for node pool %q", nodePool.Name)
			return
		}
		var subnetIDs []string
		for _, subnet := range subnets {
			subnetIDs = append(subnetIDs, subnet.SubnetID)
		}

		log.Infoln("node pool mapped to subnets", subnetIDs)

		stackParams = []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("KeyName"),
				ParameterValue: aws.String(a.context.SSHKeyName),
			},
			{
				ParameterKey:   aws.String("NodeImageId"),
				ParameterValue: aws.String(nodePool.NodeImage),
			},
			{
				ParameterKey:   aws.String("NodeInstanceType"),
				ParameterValue: aws.String(nodePool.NodeInstanceType),
			},
			{
				ParameterKey:   aws.String("NodeSpotPrice"),
				ParameterValue: aws.String(spotPriceParam),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", nodePool.NodeMinCount)),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", nodePool.NodeMaxCount)),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingInitSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", nodePool.Count)),
			},
			{
				ParameterKey:   aws.String("ClusterName"),
				ParameterValue: aws.String(a.context.ClusterName),
			},
			{
				ParameterKey:   aws.String("NodeGroupName"),
				ParameterValue: aws.String(nodePool.Name),
			},
			{
				ParameterKey:   aws.String("ClusterControlPlaneSecurityGroup"),
				ParameterValue: a.context.SecurityGroupID,
			},
			{
				ParameterKey:   aws.String("NodeSecurityGroup"),
				ParameterValue: a.context.NodeSecurityGroupID,
			},
			{
				ParameterKey:   aws.String("VpcId"),
				ParameterValue: a.context.VpcID,
			}, {
				ParameterKey:   aws.String("Subnets"),
				ParameterValue: aws.String(strings.Join(subnetIDs, ",")),
			},
			{
				ParameterKey:   aws.String("NodeInstanceRoleId"),
				ParameterValue: a.context.NodeInstanceRoleID,
			},
			{
				ParameterKey:   aws.String("ClusterAutoscalerEnabled"),
				ParameterValue: aws.String(fmt.Sprint(clusterAutoscalerEnabled)),
			},
			{
				ParameterKey:   aws.String("TerminationDetachEnabled"),
				ParameterValue: aws.String(fmt.Sprint(terminationDetachEnabled)),
			},
			{
				ParameterKey:   aws.String("BootstrapArguments"),
				ParameterValue: aws.String(fmt.Sprintf("--kubelet-extra-args '--node-labels %v'", strings.Join(nodeLabels, ","))),
			},
		}

		createStackInput := &cloudformation.CreateStackInput{
			ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
			DisableRollback:    aws.Bool(true),
			StackName:          aws.String(stackName),
			Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
			Parameters:         stackParams,
			Tags:               tags,
			TemplateBody:       aws.String(a.nodePoolTemplate),
			TimeoutInMinutes:   aws.Int64(10),
		}
		_, err := cloudformationSrv.CreateStack(createStackInput)
		if err != nil {
			errorChan <- errors.WrapIff(err, "could not create '%s' CF stack", stackName)
			return
		}

	} else {
		// update stack

		stackParams = []*cloudformation.Parameter{
			{
				ParameterKey:     aws.String("KeyName"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("NodeImageId"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("NodeInstanceType"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("NodeSpotPrice"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", nodePool.NodeMinCount)),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", nodePool.NodeMaxCount)),
			},
			{
				ParameterKey:   aws.String("NodeAutoScalingInitSize"),
				ParameterValue: aws.String(fmt.Sprintf("%d", nodePool.Count)),
			},
			{
				ParameterKey:     aws.String("ClusterName"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("NodeGroupName"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("ClusterControlPlaneSecurityGroup"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("NodeSecurityGroup"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("VpcId"),
				UsePreviousValue: aws.Bool(true),
			}, {
				ParameterKey:     aws.String("Subnets"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("NodeInstanceRoleId"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:   aws.String("ClusterAutoscalerEnabled"),
				ParameterValue: aws.String(fmt.Sprint(clusterAutoscalerEnabled)),
			},
			{
				ParameterKey:   aws.String("TerminationDetachEnabled"),
				ParameterValue: aws.String(fmt.Sprint(terminationDetachEnabled)),
			},
			{
				ParameterKey:     aws.String("BootstrapArguments"),
				UsePreviousValue: aws.Bool(true),
			},
		}

		// we don't reuse the creation time template, since it may have changed
		updateStackInput := &cloudformation.UpdateStackInput{
			ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
			StackName:          aws.String(stackName),
			Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
			Parameters:         stackParams,
			Tags:               tags,
			TemplateBody:       aws.String(a.nodePoolTemplate),
		}

		_, err := cloudformationSrv.UpdateStack(updateStackInput)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ValidationError" && strings.HasPrefix(awsErr.Message(), awsNoUpdatesError) {
				// Get error details
				log.Warnf("nothing changed during update!")
				waitOnCreateUpdate = false
				err = nil // nolint: ineffassign
			} else {
				errorChan <- errors.WrapIff(err, "could not update '%s' CF stack", stackName)
				return
			}
		}
	}

	ctx, cancelASGWait := context.WithCancel(context.Background())
	defer cancelASGWait()

	waitChan := make(chan error)
	defer close(waitChan)

	if waitOnCreateUpdate {
		go func(ctx context.Context, nodePool *model.AmazonNodePoolsModel) {
			waitChan <- a.WaitForASGToBeFulfilled(ctx, nodePool)
		}(ctx, nodePool)
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}

	var err, asgFulfillmentErr error
	if a.isCreate {
		err = errors.WrapIff(cloudformationSrv.WaitUntilStackCreateComplete(describeStacksInput), "waiting for %q CF stack create operation to complete failed", stackName)
	} else if waitOnCreateUpdate {
		err = errors.WrapIff(cloudformationSrv.WaitUntilStackUpdateComplete(describeStacksInput), "waiting for %q CF stack update operation to complete failed", stackName)
	}

	err = pkgCloudformation.NewAwsStackFailure(err, stackName, cloudformationSrv)
	if err != nil {
		// cancelling the wait for ASG fulfillment go routine as an error occurred during waiting for the completion of the cloud formation stack operation
		cancelASGWait()
	} else if waitOnCreateUpdate {
		asgFulfillmentErr = errors.WrapIff(<-waitChan, "node pool %q ASG not fulfilled", nodePool.Name)
	}
	errorChan <- errors.Append(err, asgFulfillmentErr)
}

// UndoAction rolls back this CreateUpdateNodePoolStackAction
func (a *CreateUpdateNodePoolStackAction) UndoAction() (err error) {
	// do not delete updated stack for now
	if !a.isCreate {
		return
	}

	for _, nodepool := range a.nodePools {
		a.log.Info("EXECUTE UNDO CreateUpdateNodePoolStackAction")
		cloudformationSrv := cloudformation.New(a.context.Session)
		deleteStackInput := &cloudformation.DeleteStackInput{
			ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
			StackName:          aws.String(a.generateStackName(nodepool)),
		}
		_, deleteErr := cloudformationSrv.DeleteStack(deleteStackInput)
		if deleteErr != nil {
			if awsErr, ok := deleteErr.(awserr.Error); ok {
				if awsErr.Code() == cloudformation.ErrCodeStackInstanceNotFoundException {
					return nil
				}
			}

			err = deleteErr
		}
	}
	return
}

// GenerateNodePoolStackName returns the CF Stack name for a node pool
func GenerateNodePoolStackName(clusterName, nodePoolName string) string {
	return "pipeline-eks-nodepool-" + clusterName + "-" + nodePoolName
}
