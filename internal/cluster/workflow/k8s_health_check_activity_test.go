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
	"context"
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

// nolint: gochecknoglobals
var k8sHealthCheckTestActivity = K8sHealthCheckActivity{}

func testK8sHealthCheckActivityExecute(ctx context.Context, input K8sHealthCheckActivityInput) error {
	return k8sHealthCheckTestActivity.Execute(ctx, input)
}

// nolint: gochecknoinits
func init() {
	activity.RegisterWithOptions(testK8sHealthCheckActivityExecute, activity.RegisterOptions{Name: K8sHealthCheckActivityName})
}

type K8sHealthCheckActivityTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestActivityEnvironment

	controlPlane *integration.ControlPlane

	client kubernetes.Interface
}

func testK8sHealthCheckActivity(t *testing.T) {
	if os.Getenv("TEST_ASSET_KUBE_APISERVER") == "" || os.Getenv("TEST_ASSET_ETCD") == "" {
		t.Skip("control plane binaries are missing")
	}

	suite.Run(t, new(K8sHealthCheckActivityTestSuite))
}

func (s *K8sHealthCheckActivityTestSuite) SetupSuite() {
	s.controlPlane = &integration.ControlPlane{}

	err := s.controlPlane.Start()
	s.Require().NoError(err)
}

func (s *K8sHealthCheckActivityTestSuite) TearDownSuite() {
	_ = s.controlPlane.Stop()
}

func (s *K8sHealthCheckActivityTestSuite) SetupTest() {
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

func (s *K8sHealthCheckActivityTestSuite) Test_Execute() {
	clientFactory := new(MockClientFactory)
	clientFactory.On("FromSecret", mock.Anything, "secret").Return(s.client, nil)

	k8sHealthChecker := new(MockK8sHealthChecker)
	k8sHealthChecker.On("Check", mock.Anything, mock.Anything).Return(nil)

	k8sHealthCheckTestActivity = MakeK8sHealthCheckActivity(k8sHealthChecker, clientFactory)

	_, err := s.env.ExecuteActivity(
		K8sHealthCheckActivityName,
		K8sHealthCheckActivityInput{
			OrganizationID: 1,
			ClusterName:    "test",
			K8sSecretBRN:   "brn:1:secret:secret",
		},
	)

	s.Require().NoError(err)
}
