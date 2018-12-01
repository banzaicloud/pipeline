// Copyright © 2018 Banzai Cloud
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

package ec2

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// DescribeInstanceById returns detailed information about the instance
func DescribeInstanceById(client *ec2.EC2, instanceId string) (*ec2.Instance, error) {
	result, err := client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			&instanceId,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result.Reservations) == 1 {
		reservation := result.Reservations[0]
		if len(reservation.Instances) == 1 {
			return reservation.Instances[0], nil
		}
	}

	return nil, awserr.New("404", "instance not found", nil)
}
