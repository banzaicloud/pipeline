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
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
)

const DeleteVPCActivityName = "pke-delete-vpc-activity"

type DeleteVPCActivity struct {
	clusters Clusters
}

func NewDeleteVPCActivity(clusters Clusters) *DeleteVPCActivity {
	return &DeleteVPCActivity{
		clusters: clusters,
	}
}

type DeleteVPCActivityInput struct {
	ClusterID uint
}

func (a *DeleteVPCActivity) Execute(ctx context.Context, input DeleteVPCActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}
	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return errors.New(fmt.Sprintf("can't delete VPC for cluster type %t", c))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return errors.WrapIf(err, "failed to connect to AWS")
	}

	cfClient := cloudformation.New(client)

	clusterName := c.GetName()
	stackName := "pke-vpc-" + clusterName
	stackInput := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(sdkAmazon.NewNormalizedClientRequestToken(activity.GetInfo(ctx).WorkflowExecution.ID)),
		StackName:          &stackName,
	}

	_, err = cfClient.DeleteStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		default:
			return err
		}
	}

	return nil
}

const WaitForDeleteVPCActivityName = "wait-for-pke-delete-vpc-activity"

type WaitForDeleteVPCActivity struct {
	DeleteVPCActivity
}

func NewWaitForDeleteVPCActivity(clusters Clusters) *WaitForDeleteVPCActivity {
	return &WaitForDeleteVPCActivity{
		DeleteVPCActivity{
			clusters: clusters,
		},
	}
}

func (a *WaitForDeleteVPCActivity) Execute(ctx context.Context, input DeleteVPCActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}
	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return errors.New(fmt.Sprintf("can't delete VPC for cluster type %t", c))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return errors.WrapIf(err, "failed to connect to AWS")
	}

	cfClient := cloudformation.New(client)

	clusterName := c.GetName()
	stackName := "pke-vpc-" + clusterName

	err = cfClient.WaitUntilStackDeleteCompleteWithContext(ctx,
		&cloudformation.DescribeStacksInput{StackName: &stackName},
		request.WithWaiterRequestOptions(WithHeartBeatOption(ctx)))

	return errors.WrapIf(pkgCloudformation.NewAwsStackFailure(err, stackName, "", cfClient), "failure while waiting for vpc stack deletion to complete")
}
