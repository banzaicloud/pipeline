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
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"go.uber.org/cadence/activity"
)

const WaitELBsDeletionActivityName = "eks-wait-elbs-deletion"

// WaitELBsDeletionActivity waits for the deletion of a list of ELBs identified by name
type WaitELBsDeletionActivity struct {
	awsSessionFactory *AWSSessionFactory
}

// WaitELBsDeletionActivity holds the names of the ELBs to wait for to be deleted
type WaitELBsDeletionActivityActivityInput struct {
	OrganizationID uint
	SecretID       string
	Region         string

	ClusterName string

	LoadBalancerNames []string
}

// NewWaitELBsDeletionActivity instantiates a new NewWaitELBsDeletionActivity
func NewWaitELBsDeletionActivity(awsSessionFactory *AWSSessionFactory) *WaitELBsDeletionActivity {
	return &WaitELBsDeletionActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *WaitELBsDeletionActivity) Execute(ctx context.Context, input WaitELBsDeletionActivityActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"region", input.Region,
		"cluster", input.ClusterName,
	)

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return err
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	elbService := elb.New(session)
	loadBalancersToWait := input.LoadBalancerNames

	i := 0
	for {
		var remaining []string

		select {
		case <-ticker.C:
			if i < 5 {
				i++

				logger.With("ELBs", loadBalancersToWait).Info("waiting for ELBs to be deleted")

				for _, lbName := range loadBalancersToWait {
					describeLoadBalancers := &elb.DescribeLoadBalancersInput{
						LoadBalancerNames: aws.StringSlice([]string{lbName}),
					}

					_, err := elbService.DescribeLoadBalancersWithContext(ctx, describeLoadBalancers)
					if err != nil {
						var awsErr awserr.Error
						if errors.As(err, &awsErr) && awsErr.Code() == elb.ErrCodeAccessPointNotFoundException {
							continue
						}

						if err = errors.WrapIf(err, "couldn't describe ELBs"); err != nil {
							return err
						}

					}

					remaining = append(remaining, lbName)
				}

				loadBalancersToWait = remaining
				if len(loadBalancersToWait) == 0 {
					return nil
				}
			} else {
				return errors.New("waiting for ELBs to be deleted timed out")
			}
		case <-ctx.Done():
			return nil

		}
	}
}
