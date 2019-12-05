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
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"

	logrusadapter "logur.dev/adapter/logrus"

	"github.com/banzaicloud/pipeline/pkg/providers/amazon/autoscaling"
)

// WaitForASGToBeFulfilled waits until an ASG has the desired amount of healthy nodes
func WaitForASGToBeFulfilled(
	ctx context.Context,
	awsSession *session.Session,
	logger logrus.FieldLogger,
	clusterName string,
	nodePoolName string,
	waitAttempts int,
	waitInterval time.Duration) error {

	logurLogger := logrusadapter.New(logrus.New())
	m := autoscaling.NewManager(awsSession, autoscaling.MetricsEnabled(true), autoscaling.Logger{
		Logger: logurLogger,
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

// GenerateNodePoolStackName returns the CF Stack name for a node pool
func GenerateNodePoolStackName(clusterName, nodePoolName string) string {
	return "pipeline-eks-nodepool-" + clusterName + "-" + nodePoolName
}
