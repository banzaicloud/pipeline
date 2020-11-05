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

package pkeawsworkflow

import (
	"reflect"
	"testing"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"

	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
)

func TestNewHealthCheckActivity(t *testing.T) {
	type args struct {
		awsSessionFactory AWSSessionFactory
		elbv2Factory      ELBV2APIFactory
	}
	tests := []struct {
		name string
		args args
		want *HealthCheckActivity
	}{
		{
			name: "nil AWSSessionFactory nil ELBV2APIFactory",
			args: args{
				awsSessionFactory: nil,
				elbv2Factory:      nil,
			},
			want: &HealthCheckActivity{
				awsSessionFactory: nil,
				elbv2Factory:      nil,
			},
		},
		{
			name: "nil AWSSessionFactory not nil ELBV2APIFactory",
			args: args{
				awsSessionFactory: nil,
				elbv2Factory:      NewELBV2Factory(),
			},
			want: &HealthCheckActivity{
				awsSessionFactory: nil,
				elbv2Factory:      NewELBV2Factory(),
			},
		},
		{
			name: "not nil AWSSessionFactory nil ELBV2APIFactory",
			args: args{
				awsSessionFactory: pkeworkflow.NewAWSClientFactory(nil),
				elbv2Factory:      nil,
			},
			want: &HealthCheckActivity{
				awsSessionFactory: pkeworkflow.NewAWSClientFactory(nil),
				elbv2Factory:      nil,
			},
		},
		{
			name: "not nil AWSSessionFactory not nil ELBV2APIFactory",
			args: args{
				awsSessionFactory: pkeworkflow.NewAWSClientFactory(nil),
				elbv2Factory:      NewELBV2Factory(),
			},
			want: &HealthCheckActivity{
				awsSessionFactory: pkeworkflow.NewAWSClientFactory(nil),
				elbv2Factory:      NewELBV2Factory(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHealthCheckActivity(tt.args.awsSessionFactory, tt.args.elbv2Factory); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHealthCheckActivity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthCheckActivityExecute(t *testing.T) {
	emptyString := ""
	healthy := elbv2.TargetHealthStateEnumHealthy
	unHealthy := elbv2.TargetHealthStateEnumUnhealthy
	testInput := HealthCheckActivityInput{
		OrganizationID: uint(1),
		SecretID:       "brn:test:secret",
		Region:         "us-east-2",
		ClusterID:      uint(1),
		ClusterName:    "test-cluster",
	}
	nlbName := "pke-test-cluster-nlb"
	type inputType struct {
		activity *HealthCheckActivity
		input    HealthCheckActivityInput
	}
	type outputType struct {
		expectedError  error
		expectedOutput *HealthCheckActivityOutput
	}
	tests := []struct {
		name       string
		input      inputType
		output     outputType
		setupMocks func(input inputType, output outputType)
	}{
		{
			name: "nil activity",
			input: inputType{
				activity: nil,
				input:    testInput,
			},
			output: outputType{
				expectedError:  errors.New("activity is nil"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {},
		},
		{
			name: "aws session factory error",
			input: inputType{
				activity: NewHealthCheckActivity(&MockAWSSessionFactory{}, &MockELBV2APIFactory{}),
				input:    testInput,
			},
			output: outputType{
				expectedError:  errors.New("failed to create AWS session: test error"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSSessionFactory, isOk := input.activity.awsSessionFactory.(*MockAWSSessionFactory)
				require.True(t, isOk, "test AWSSessionFactory is not a mock")

				mockAWSSessionFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(nil, errors.New("test error"))
			},
		},
		{
			name: "describing load balancers failed",
			input: inputType{
				activity: NewHealthCheckActivity(&MockAWSSessionFactory{}, &MockELBV2APIFactory{}),
				input:    testInput,
			},
			output: outputType{
				expectedError:  errors.New("all attempts failed: failed to describe load balancers: test error"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSSessionFactory, isOk := input.activity.awsSessionFactory.(*MockAWSSessionFactory)
				require.True(t, isOk, "test AWSSessionFactory is not a mock")

				mockAWSSession := &session.Session{}
				mockAWSSessionFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(mockAWSSession, nil)

				mockELBV2APIFactory, isOk := input.activity.elbv2Factory.(*MockELBV2APIFactory)
				require.True(t, isOk, "test ELBV2APIFactory is not a mock")

				mockELBV2API := &Mockelbv2API{}
				mockELBV2APIFactory.On(
					"New",
					mockAWSSession,
				).Return(mockELBV2API)

				mockELBV2API.On(
					"DescribeLoadBalancers",
					&elbv2.DescribeLoadBalancersInput{
						Names: []*string{&nlbName},
					},
				).Return(nil, errors.New("test error"))
			},
		},
		{
			name: "describing target groups failed",
			input: inputType{
				activity: NewHealthCheckActivity(&MockAWSSessionFactory{}, &MockELBV2APIFactory{}),
				input:    testInput,
			},
			output: outputType{
				expectedError:  errors.New("all attempts failed: failed to describe target groups: test error"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSSessionFactory, isOk := input.activity.awsSessionFactory.(*MockAWSSessionFactory)
				require.True(t, isOk, "test AWSSessionFactory is not a mock")

				mockAWSSession := &session.Session{}
				mockAWSSessionFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(mockAWSSession, nil)

				mockELBV2APIFactory, isOk := input.activity.elbv2Factory.(*MockELBV2APIFactory)
				require.True(t, isOk, "test ELBV2APIFactory is not a mock")

				mockELBV2API := &Mockelbv2API{}
				mockELBV2APIFactory.On(
					"New",
					mockAWSSession,
				).Return(mockELBV2API)

				mockELBV2API.On(
					"DescribeLoadBalancers",
					&elbv2.DescribeLoadBalancersInput{
						Names: []*string{&nlbName},
					},
				).Return(&elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: []*elbv2.LoadBalancer{
						{
							LoadBalancerArn: &emptyString,
						},
					},
				}, nil)

				mockELBV2API.On(
					"DescribeTargetGroups",
					&elbv2.DescribeTargetGroupsInput{
						LoadBalancerArn: &emptyString,
					},
				).Return(nil, errors.New("test error"))
			},
		},
		{
			name: "describing target health failed",
			input: inputType{
				activity: NewHealthCheckActivity(&MockAWSSessionFactory{}, &MockELBV2APIFactory{}),
				input:    testInput,
			},
			output: outputType{
				expectedError:  errors.New("all attempts failed: failed to describe target health: test error"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSSessionFactory, isOk := input.activity.awsSessionFactory.(*MockAWSSessionFactory)
				require.True(t, isOk, "test AWSSessionFactory is not a mock")

				mockAWSSession := &session.Session{}
				mockAWSSessionFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(mockAWSSession, nil)

				mockELBV2APIFactory, isOk := input.activity.elbv2Factory.(*MockELBV2APIFactory)
				require.True(t, isOk, "test ELBV2APIFactory is not a mock")

				mockELBV2API := &Mockelbv2API{}
				mockELBV2APIFactory.On(
					"New",
					mockAWSSession,
				).Return(mockELBV2API)

				mockELBV2API.On(
					"DescribeLoadBalancers",
					&elbv2.DescribeLoadBalancersInput{
						Names: []*string{&nlbName},
					},
				).Return(&elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: []*elbv2.LoadBalancer{
						{
							LoadBalancerArn: &emptyString,
						},
					},
				}, nil)

				mockELBV2API.On(
					"DescribeTargetGroups",
					&elbv2.DescribeTargetGroupsInput{
						LoadBalancerArn: &emptyString,
					},
				).Return(&elbv2.DescribeTargetGroupsOutput{
					TargetGroups: []*elbv2.TargetGroup{
						{
							TargetGroupArn: &emptyString,
						},
					},
				}, nil)

				mockELBV2API.On(
					"DescribeTargetHealth",
					&elbv2.DescribeTargetHealthInput{
						TargetGroupArn: &emptyString,
					},
				).Return(nil, errors.New("test error"))
			},
		},
		{
			name: "not all targets are healthy",
			input: inputType{
				activity: NewHealthCheckActivity(&MockAWSSessionFactory{}, &MockELBV2APIFactory{}),
				input:    testInput,
			},
			output: outputType{
				expectedError:  errors.New("not all targets of the network load balancer are healthy"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSSessionFactory, isOk := input.activity.awsSessionFactory.(*MockAWSSessionFactory)
				require.True(t, isOk, "test AWSSessionFactory is not a mock")

				mockAWSSession := &session.Session{}
				mockAWSSessionFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(mockAWSSession, nil)

				mockELBV2APIFactory, isOk := input.activity.elbv2Factory.(*MockELBV2APIFactory)
				require.True(t, isOk, "test ELBV2APIFactory is not a mock")

				mockELBV2API := &Mockelbv2API{}
				mockELBV2APIFactory.On(
					"New",
					mockAWSSession,
				).Return(mockELBV2API)

				mockELBV2API.On(
					"DescribeLoadBalancers",
					&elbv2.DescribeLoadBalancersInput{
						Names: []*string{&nlbName},
					},
				).Return(&elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: []*elbv2.LoadBalancer{
						{
							LoadBalancerArn: &emptyString,
						},
					},
				}, nil)

				mockELBV2API.On(
					"DescribeTargetGroups",
					&elbv2.DescribeTargetGroupsInput{
						LoadBalancerArn: &emptyString,
					},
				).Return(&elbv2.DescribeTargetGroupsOutput{
					TargetGroups: []*elbv2.TargetGroup{
						{
							TargetGroupArn: &emptyString,
						},
					},
				}, nil)

				mockELBV2API.On(
					"DescribeTargetHealth",
					&elbv2.DescribeTargetHealthInput{
						TargetGroupArn: &emptyString,
					},
				).Return(&elbv2.DescribeTargetHealthOutput{
					TargetHealthDescriptions: []*elbv2.TargetHealthDescription{
						{
							Target: &elbv2.TargetDescription{
								Id: &emptyString,
							},
							TargetHealth: &elbv2.TargetHealth{
								State: &healthy,
							},
						},
						{
							Target: &elbv2.TargetDescription{
								Id: &emptyString,
							},
							TargetHealth: &elbv2.TargetHealth{
								State: &unHealthy,
							},
						},
					},
				}, nil)
			},
		},
		{
			name: "target health status is not set",
			input: inputType{
				activity: NewHealthCheckActivity(&MockAWSSessionFactory{}, &MockELBV2APIFactory{}),
				input:    testInput,
			},
			output: outputType{
				expectedError:  errors.New("unable to getting state of the target health description"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSSessionFactory, isOk := input.activity.awsSessionFactory.(*MockAWSSessionFactory)
				require.True(t, isOk, "test AWSSessionFactory is not a mock")

				mockAWSSession := &session.Session{}
				mockAWSSessionFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(mockAWSSession, nil)

				mockELBV2APIFactory, isOk := input.activity.elbv2Factory.(*MockELBV2APIFactory)
				require.True(t, isOk, "test ELBV2APIFactory is not a mock")

				mockELBV2API := &Mockelbv2API{}
				mockELBV2APIFactory.On(
					"New",
					mockAWSSession,
				).Return(mockELBV2API)

				mockELBV2API.On(
					"DescribeLoadBalancers",
					&elbv2.DescribeLoadBalancersInput{
						Names: []*string{&nlbName},
					},
				).Return(&elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: []*elbv2.LoadBalancer{
						{
							LoadBalancerArn: &emptyString,
						},
					},
				}, nil)

				mockELBV2API.On(
					"DescribeTargetGroups",
					&elbv2.DescribeTargetGroupsInput{
						LoadBalancerArn: &emptyString,
					},
				).Return(&elbv2.DescribeTargetGroupsOutput{
					TargetGroups: []*elbv2.TargetGroup{
						{
							TargetGroupArn: &emptyString,
						},
					},
				}, nil)

				mockELBV2API.On(
					"DescribeTargetHealth",
					&elbv2.DescribeTargetHealthInput{
						TargetGroupArn: &emptyString,
					},
				).Return(&elbv2.DescribeTargetHealthOutput{
					TargetHealthDescriptions: []*elbv2.TargetHealthDescription{
						{
							Target: &elbv2.TargetDescription{
								Id: &emptyString,
							},
							TargetHealth: &elbv2.TargetHealth{
								State: &healthy,
							},
						},
						{
							Target: &elbv2.TargetDescription{
								Id: &emptyString,
							},
						},
					},
				}, nil)
			},
		},
		{
			name: "all targets are healthy",
			input: inputType{
				activity: NewHealthCheckActivity(&MockAWSSessionFactory{}, &MockELBV2APIFactory{}),
				input:    testInput,
			},
			output: outputType{
				expectedError:  nil,
				expectedOutput: &HealthCheckActivityOutput{nlbStatus: nil},
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSSessionFactory, isOk := input.activity.awsSessionFactory.(*MockAWSSessionFactory)
				require.True(t, isOk, "test AWSSessionFactory is not a mock")

				mockAWSSession := &session.Session{}
				mockAWSSessionFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(mockAWSSession, nil)

				mockELBV2APIFactory, isOk := input.activity.elbv2Factory.(*MockELBV2APIFactory)
				require.True(t, isOk, "test ELBV2APIFactory is not a mock")

				mockELBV2API := &Mockelbv2API{}
				mockELBV2APIFactory.On(
					"New",
					mockAWSSession,
				).Return(mockELBV2API)

				mockELBV2API.On(
					"DescribeLoadBalancers",
					&elbv2.DescribeLoadBalancersInput{
						Names: []*string{&nlbName},
					},
				).Return(&elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: []*elbv2.LoadBalancer{
						{
							LoadBalancerArn: &emptyString,
						},
					},
				}, nil)

				mockELBV2API.On(
					"DescribeTargetGroups",
					&elbv2.DescribeTargetGroupsInput{
						LoadBalancerArn: &emptyString,
					},
				).Return(&elbv2.DescribeTargetGroupsOutput{
					TargetGroups: []*elbv2.TargetGroup{
						{
							TargetGroupArn: &emptyString,
						},
					},
				}, nil)

				mockELBV2API.On(
					"DescribeTargetHealth",
					&elbv2.DescribeTargetHealthInput{
						TargetGroupArn: &emptyString,
					},
				).Return(&elbv2.DescribeTargetHealthOutput{
					TargetHealthDescriptions: []*elbv2.TargetHealthDescription{
						{
							TargetHealth: &elbv2.TargetHealth{
								State: &healthy,
							},
						},
						{
							TargetHealth: &elbv2.TargetHealth{
								State: &healthy,
							},
						},
					},
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowTestSuite := &testsuite.WorkflowTestSuite{}
			testActivityEnvironment := workflowTestSuite.NewTestActivityEnvironment()

			testActivityEnvironment.RegisterActivityWithOptions(
				tt.input.activity.Execute,
				activity.RegisterOptions{Name: tt.name},
			)

			tt.setupMocks(tt.input, tt.output)

			actualValue, actualError := testActivityEnvironment.ExecuteActivity(
				tt.name,
				tt.input.input,
			)

			var actualOutput *HealthCheckActivityOutput
			if actualValue != nil &&
				actualValue.HasValue() {
				err := actualValue.Get(&actualOutput)
				require.NoError(t, err)
			}

			if tt.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, tt.output.expectedError.Error())
			}
			require.Equal(t, tt.output.expectedOutput, actualOutput)
		})
	}
}
