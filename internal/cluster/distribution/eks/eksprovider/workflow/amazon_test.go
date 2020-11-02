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

package workflow

import (
	"testing"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
)

func TestNewSubnetsFromEKSSubnets(t *testing.T) {
	type inputType struct {
		eksSubnets                []*eksmodel.EKSSubnetModel
		optionalIncludedSubnetIDs []string
	}

	type outputType struct {
		expectedError   error
		expectedSubnets []Subnet
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "subnet ID not found -> error",
			input: inputType{
				eksSubnets:                []*eksmodel.EKSSubnetModel{},
				optionalIncludedSubnetIDs: []string{"not-existing-id"},
			},
			output: outputType{
				expectedError:   errors.New("some subnet IDs could not be found among the subnets"),
				expectedSubnets: nil,
			},
		},
		{
			caseDescription: "no included subnet IDs -> success",
			input: inputType{
				eksSubnets: []*eksmodel.EKSSubnetModel{
					{
						ID:               1,
						SubnetId:         aws.String("subnet-id-1"),
						Cidr:             aws.String("cidr-1"),
						AvailabilityZone: aws.String("availability-zone-1"),
					},
					{
						ID:               2,
						SubnetId:         aws.String("subnet-id-2"),
						Cidr:             aws.String("cidr-2"),
						AvailabilityZone: aws.String("availability-zone-2"),
					},
				},
				optionalIncludedSubnetIDs: nil,
			},
			output: outputType{
				expectedError: nil,
				expectedSubnets: []Subnet{
					{
						SubnetID:         "subnet-id-1",
						Cidr:             "cidr-1",
						AvailabilityZone: "availability-zone-1",
					},
					{
						SubnetID:         "subnet-id-2",
						Cidr:             "cidr-2",
						AvailabilityZone: "availability-zone-2",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualSubnets, actualError := NewSubnetsFromEKSSubnets(
				testCase.input.eksSubnets,
				testCase.input.optionalIncludedSubnetIDs...,
			)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedSubnets, actualSubnets)
		})
	}
}
