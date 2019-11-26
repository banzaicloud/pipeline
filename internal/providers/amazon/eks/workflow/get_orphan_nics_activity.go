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
	"github.com/aws/aws-sdk-go/service/ec2"

	"go.uber.org/cadence/activity"

	zapadapter "logur.dev/adapter/zap"

	pkgEC2 "github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
)

const GetOrphanNICsActivityName = "eks-get-orphan-nics"

type GetOrphanNICsActivity struct {
	awsSessionFactory *AWSSessionFactory
}

type GetOrphanNICsActivityInput struct {
	EKSActivityInput

	VpcID            string
	SecurityGroupIDs []string
}

type GetOrphanNICsActivityOutput struct {
	NicList []string
}

func NewGetOrphanNICsActivity(awsSessionFactory *AWSSessionFactory) *GetOrphanNICsActivity {
	return &GetOrphanNICsActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *GetOrphanNICsActivity) Execute(ctx context.Context, input GetOrphanNICsActivityInput) (*GetOrphanNICsActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
	)

	if input.VpcID == "" || len(input.SecurityGroupIDs) == 0 {
		return nil, nil
	}

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	netSvc := pkgEC2.NewNetworkSvc(ec2.New(awsSession), zapadapter.New(logger.Desugar()))

	// collect orphan ENIs
	// CNI plugin applies the following tags to ENIs https://aws.amazon.com/blogs/opensource/vpc-cni-plugin-v1-1-available/
	tagsFilter := map[string][]string{
		"node.k8s.amazonaws.com/instance_id": nil,
	}
	nics, err := netSvc.GetUnusedNetworkInterfaces(input.VpcID, input.SecurityGroupIDs, tagsFilter)
	if err != nil {
		return nil, errors.WrapIf(err, "searching for unused network interfaces failed")
	}

	logger.Infof("NIC's used by cluster: '%s'", nics)

	output := GetOrphanNICsActivityOutput{
		NicList: nics,
	}

	return &output, nil
}
