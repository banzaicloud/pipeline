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

package eksworkflow

import (
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
	internalAmazon "github.com/banzaicloud/pipeline/internal/providers/amazon"
	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

// AWSSessionFactory creates an AWS session.
type AWSSessionFactory interface {
	// NewSession creates an AWS session.
	NewSession(secretID string, region string) (*session.Session, error)
}

// getStackTags returns the tags that are placed onto CF template stacks.
// These tags  are propagated onto the resources created by the CF template.
// TODO: move these to a better place
func getStackTags(clusterName, stackType string, clusterTags map[string]string) []*cloudformation.Tag {
	tags := make([]*cloudformation.Tag, 0)

	for k, v := range clusterTags {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	tags = append(tags, []*cloudformation.Tag{
		{Key: aws.String("banzaicloud-pipeline-cluster-name"), Value: aws.String(clusterName)},
		{Key: aws.String("banzaicloud-pipeline-stack-type"), Value: aws.String(stackType)},
	}...)
	tags = append(tags, internalAmazon.PipelineTags()...)
	return tags
}

func getNodePoolStackTags(clusterName string, clusterTags map[string]string) []*cloudformation.Tag {
	return getStackTags(clusterName, "nodepool", clusterTags)
}

// ErrReasonStackFailed cadence custom error reason that denotes a stack operation that resulted a stack failure
// TODO: this is temporary
const ErrReasonStackFailed = "CLOUDFORMATION_STACK_FAILED"

// TODO: this is temporary
func setClusterStatus(ctx workflow.Context, clusterID uint, status, statusMessage string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    2 * time.Minute,
		WaitForCancellation:    true,
	})

	return workflow.ExecuteActivity(ctx, clusterworkflow.SetClusterStatusActivityName, clusterworkflow.SetClusterStatusActivityInput{
		ClusterID:     clusterID,
		Status:        status,
		StatusMessage: statusMessage,
	}).Get(ctx, nil)
}

// TODO: this is temporary
func packageCFError(err error, stackName string, clientRequestToken string, cloudformationClient *cloudformation.CloudFormation, errMessage string) error {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		if awsErr.Code() == request.WaiterResourceNotReadyErrorCode {
			err = pkgCloudformation.NewAwsStackFailure(err, stackName, clientRequestToken, cloudformationClient)
			err = errors.WrapIfWithDetails(err, errMessage, "stackName", stackName)
			if pkgCloudformation.IsErrorFinal(err) {
				return cadence.NewCustomError(ErrReasonStackFailed, err.Error())
			}
			return err
		}
	}
	return err
}
