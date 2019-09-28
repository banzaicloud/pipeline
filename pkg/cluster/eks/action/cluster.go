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
	"time"

	"emperror.dev/errors"
	"github.com/Masterminds/semver"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/utils"
)

var _ utils.RevocableAction = (*CreateEksClusterAction)(nil)

// CreateEksClusterAction describes the properties of an EKS cluster creation
type CreateEksClusterAction struct {
	context           *EksClusterCreateUpdateContext
	kubernetesVersion string
	log               logrus.FieldLogger
}

// NewCreateEksClusterAction creates a new CreateEksClusterAction
func NewCreateEksClusterAction(log logrus.FieldLogger, creationContext *EksClusterCreateUpdateContext, kubernetesVersion string) *CreateEksClusterAction {
	return &CreateEksClusterAction{
		context:           creationContext,
		kubernetesVersion: kubernetesVersion,
		log:               log,
	}
}

// GetName returns the name of this CreateEksClusterAction
func (a *CreateEksClusterAction) GetName() string {
	return "CreateEksClusterAction"
}

// ExecuteAction executes this CreateEksClusterAction
func (a *CreateEksClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	vpcConfigRequest, ok := input.(*eks.VpcConfigRequest)

	if !ok {
		return nil, errors.New("input parameter must be a *VpcConfigRequest")
	}

	a.log.Infoln("EXECUTE CreateEksClusterAction")
	eksSvc := eks.New(a.context.Session)

	roleArn := a.context.ClusterRoleArn

	createClusterInput := &eks.CreateClusterInput{
		ClientRequestToken: aws.String(uuid.Must(uuid.NewV4()).String()),
		Name:               aws.String(a.context.ClusterName),
		ResourcesVpcConfig: vpcConfigRequest,
		RoleArn:            &roleArn,
	}

	// set Kubernetes version only if provided, otherwise the cloud provider default one will be used
	if len(a.kubernetesVersion) > 0 {
		// EKS CreateCluster API accepts only major.minor Kubernetes version
		v, err := semver.NewVersion(a.kubernetesVersion)
		if err != nil {
			return nil, errors.WrapIff(err, "invalid Kubernetes version %q", a.kubernetesVersion)
		}

		createClusterInput.Version = aws.String(fmt.Sprintf("%d.%d", v.Major(), v.Minor()))
	}

	result, err := eksSvc.CreateCluster(createClusterInput)
	if err != nil {
		return nil, errors.WrapIf(err, "failer to create EKS master")
	}

	// wait for ready status
	startTime := time.Now()
	a.log.Info("waiting for EKS cluster creation")
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(a.context.ClusterName),
	}
	err = a.waitUntilClusterCreateComplete(describeClusterInput)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()
	a.log.Infoln("EKS cluster created successfully in", endTime.Sub(startTime).String())

	return result.Cluster, nil
}

func (a *CreateEksClusterAction) waitUntilClusterCreateComplete(input *eks.DescribeClusterInput) error {
	return a.waitUntilClusterCreateCompleteWithContext(aws.BackgroundContext(), input)
}

func (a *CreateEksClusterAction) waitUntilClusterCreateCompleteWithContext(ctx aws.Context, input *eks.DescribeClusterInput, opts ...request.WaiterOption) error {
	eksSvc := eks.New(a.context.Session)

	w := request.Waiter{
		Name:        "WaitUntilClusterCreateComplete",
		MaxAttempts: 120,
		Delay:       request.ConstantWaiterDelay(30 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Cluster.Status",
				Expected: eks.ClusterStatusActive,
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Cluster.Status",
				Expected: eks.ClusterStatusDeleting,
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Cluster.Status",
				Expected: eks.ClusterStatusFailed,
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.ErrorWaiterMatch,
				Expected: "ValidationError",
			},
		},
		Logger: eksSvc.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			var inCpy *eks.DescribeClusterInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := eksSvc.DescribeClusterRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.ApplyOptions(opts...)

	return w.WaitWithContext(ctx)
}

// UndoAction rolls back this CreateEksClusterAction
func (a *CreateEksClusterAction) UndoAction() (err error) {
	a.log.Infoln("EXECUTE UNDO CreateEksClusterAction")
	eksSvc := eks.New(a.context.Session)

	deleteClusterInput := &eks.DeleteClusterInput{
		Name: aws.String(a.context.ClusterName),
	}
	_, err = eksSvc.DeleteCluster(deleteClusterInput)
	return err
}
