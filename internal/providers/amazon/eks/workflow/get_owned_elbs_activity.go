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

package workflow

import (
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"go.uber.org/cadence/activity"
)

const GetOwnedELBsActivityName = "eks-get-owned-elbs"

// GetOwnedELBsActivity collects all ELBs that were created by the EKS cluster
type GetOwnedELBsActivity struct {
	awsSessionFactory *AWSSessionFactory
}

// GetOwnedELBsActivityInput holds fields needed to retrieve all ELBs provisioned by
// an EKS cluster
type GetOwnedELBsActivityInput struct {
	EKSActivityInput

	VpcID string
}

type GetOwnedELBsActivityOutput struct {
	LoadBalancerNames []string
}

// NewGetOwnedELBsActivity instantiates a new GetOwnedELBsActivity
func NewGetOwnedELBsActivity(awsSessionFactory *AWSSessionFactory) *GetOwnedELBsActivity {
	return &GetOwnedELBsActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *GetOwnedELBsActivity) Execute(ctx context.Context, input GetOwnedELBsActivityInput) (*GetOwnedELBsActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"region", input.Region,
		"vpcID", input.VpcID,
		"cluster", input.ClusterName,
	)

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	elbService := elb.New(session)
	clusterTag := "kubernetes.io/cluster/" + input.ClusterName

	logger.Infof("search for ELBs with tag '%s'", clusterTag)

	var loadBalancerNames []*string
	describeLoadBalancers := &elb.DescribeLoadBalancersInput{}
	var output GetOwnedELBsActivityOutput

	err = elbService.DescribeLoadBalancersPagesWithContext(ctx, describeLoadBalancers,
		func(page *elb.DescribeLoadBalancersOutput, lastPage bool) bool {

			for _, lb := range page.LoadBalancerDescriptions {
				if aws.StringValue(lb.VPCId) == input.VpcID {
					loadBalancerNames = append(loadBalancerNames, lb.LoadBalancerName)
				}
			}

			return lastPage
		})

	if err != nil {
		return nil, errors.WrapIf(err, "couldn't describe ELBs")
	}

	if len(loadBalancerNames) == 0 {
		return &output, nil
	}

	// according to https://docs.aws.amazon.com/elasticloadbalancing/2012-06-01/APIReference/API_DescribeTags.html
	// tags can be queried for up to 20 ELBs in one call
	maxELBNames := 20
	for low := 0; low < len(loadBalancerNames); low += maxELBNames {
		high := low + maxELBNames

		if high > len(loadBalancerNames) {
			high = len(loadBalancerNames)
		}

		describeTagsInput := &elb.DescribeTagsInput{
			LoadBalancerNames: loadBalancerNames[low:high],
		}

		describeTagsOutput, err := elbService.DescribeTagsWithContext(ctx, describeTagsInput)
		if err != nil {
			return nil, errors.WrapIf(err, "couldn't describe ELB tags")
		}

		for _, tagDescription := range describeTagsOutput.TagDescriptions {
			for _, tag := range tagDescription.Tags {
				if aws.StringValue(tag.Key) == clusterTag {
					output.LoadBalancerNames = append(output.LoadBalancerNames, aws.StringValue(tagDescription.LoadBalancerName))
				}
			}
		}

	}

	logger.Infof("ELBs owned by cluster: '%s'", output.LoadBalancerNames)

	return &output, nil
}
