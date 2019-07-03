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

	"github.com/aws/aws-sdk-go/service/ec2"
	pkgEC2 "github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/goph/logur/adapters/zapadapter"
	"github.com/pkg/errors"
	"go.uber.org/cadence/activity"
)

const GetVpcDefaultSecurityGroupActivityName = "pke-get-vpc-default-sg-activity"

type GetVpcDefaultSecurityGroupActivity struct {
	awsClientFactory *AWSClientFactory
}

type GetVpcDefaultSecurityGroupActivityInput struct {
	AWSActivityInput
	ClusterID uint
	VpcID     string
}

func NewGetVpcDefaultSecurityGroupActivity(awsClientFactory *AWSClientFactory) *GetVpcDefaultSecurityGroupActivity {
	return &GetVpcDefaultSecurityGroupActivity{
		awsClientFactory: awsClientFactory,
	}
}

func (a *GetVpcDefaultSecurityGroupActivity) Execute(ctx context.Context, input GetVpcDefaultSecurityGroupActivityInput) (string, error) {
	logger := logur.WithFields(zapadapter.New(activity.GetLogger(ctx)), map[string]interface{}{"clusterID": input.ClusterID, "vpcId": input.VpcID})

	client, err := a.awsClientFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return "", err
	}

	netSvc := pkgEC2.NewNetworkSvc(ec2.New(client), logger)
	sgID, err := netSvc.GetVpcDefaultSecurityGroup(input.VpcID)

	logger.Debug("getting VPC's default security group")
	if err != nil {
		return "", emperror.WrapWith(err, "couldn't get the default security group of the VPC", "clusterID", input.ClusterID, "vpcId", input.VpcID)
	}

	if sgID == "" {
		return "", emperror.With(errors.New("couldn't get the default security group of the VPC"), "vpcId", input.VpcID)
	}

	return sgID, nil
}
