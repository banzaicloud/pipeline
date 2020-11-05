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
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/testing_frameworks/integration"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

type HealthCheckActivityTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestActivityEnvironment

	controlPlane *integration.ControlPlane

	client kubernetes.Interface
}

func testHealthCheckActivity(t *testing.T) {
	if os.Getenv("TEST_ASSET_KUBE_APISERVER") == "" || os.Getenv("TEST_ASSET_ETCD") == "" {
		t.Skip("control plane binaries are missing")
	}

	suite.Run(t, new(HealthCheckActivityTestSuite))
}

func (s *HealthCheckActivityTestSuite) SetupSuite() {
	s.controlPlane = &integration.ControlPlane{}

	err := s.controlPlane.Start()
	s.Require().NoError(err)
}

func (s *HealthCheckActivityTestSuite) TearDownSuite() {
	_ = s.controlPlane.Stop()
}

func (s *HealthCheckActivityTestSuite) SetupTest() {
	s.env = s.NewTestActivityEnvironment()

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: s.controlPlane.APIURL().String()}},
	).ClientConfig()
	s.Require().NoError(err)

	client, err := k8sclient.NewClientFromConfig(config)
	s.Require().NoError(err)

	s.client = client
}

func (s *HealthCheckActivityTestSuite) Test_Execute() {
	clientFactory := new(MockClientFactory)
	clientFactory.On("FromSecret", mock.Anything, "brn:1:secret:secret").Return(s.client, nil)

	healthChecker := new(MockHealthChecker)
	healthChecker.On("Check", mock.Anything, mock.Anything).Return(nil)

	healthCheckTestActivity := NewHealthCheckActivity(healthChecker, clientFactory)

	s.env.RegisterActivityWithOptions(healthCheckTestActivity.Execute, activity.RegisterOptions{Name: HealthCheckActivityName})

	_, err := s.env.ExecuteActivity(
		HealthCheckActivityName,
		HealthCheckActivityInput{
			SecretID: "brn:1:secret:secret",
		},
	)

	s.Require().NoError(err)
}
