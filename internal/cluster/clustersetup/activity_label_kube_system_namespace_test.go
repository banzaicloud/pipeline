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

package clustersetup

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/testing_frameworks/integration"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// nolint: gochecknoglobals
var labelKubeSystemNamespaceTestActivity = LabelKubeSystemNamespaceActivity{}

func testLabelKubeSystemNamespaceActivityExecute(ctx context.Context, input LabelKubeSystemNamespaceActivityInput) error {
	return labelKubeSystemNamespaceTestActivity.Execute(ctx, input)
}

// nolint: gochecknoinits
func init() {
	activity.RegisterWithOptions(testLabelKubeSystemNamespaceActivityExecute, activity.RegisterOptions{Name: LabelKubeSystemNamespaceActivityName})
}

type LabelKubeSystemNamespaceActivityTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestActivityEnvironment

	controlPlane *integration.ControlPlane

	client kubernetes.Interface
}

func testLabelKubeSystemNamespaceActivity(t *testing.T) {
	if os.Getenv("TEST_ASSET_KUBE_APISERVER") == "" || os.Getenv("TEST_ASSET_ETCD") == "" {
		t.Skip("control plane binaries are missing")
	}

	suite.Run(t, new(LabelKubeSystemNamespaceActivityTestSuite))
}

func (s *LabelKubeSystemNamespaceActivityTestSuite) SetupSuite() {
	s.controlPlane = &integration.ControlPlane{}

	err := s.controlPlane.Start()
	s.Require().NoError(err)
}

func (s *LabelKubeSystemNamespaceActivityTestSuite) TearDownSuite() {
	_ = s.controlPlane.Stop()
}

func (s *LabelKubeSystemNamespaceActivityTestSuite) SetupTest() {
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

func (s *LabelKubeSystemNamespaceActivityTestSuite) Test_Execute() {
	clientFactory := new(cluster.MockClientFactory)
	clientFactory.On("FromSecret", mock.Anything, "secret").Return(s.client, nil)

	labelKubeSystemNamespaceTestActivity = NewLabelKubeSystemNamespaceActivity(clientFactory)

	_, err := s.env.ExecuteActivity(
		LabelKubeSystemNamespaceActivityName,
		LabelKubeSystemNamespaceActivityInput{
			ConfigSecretID: "secret",
		},
	)

	s.Require().NoError(err)

	namespace, err := s.client.CoreV1().Namespaces().Get("kube-system", metav1.GetOptions{})
	s.Require().NoError(err)

	s.Assert().Equal(
		map[string]string{
			"name": "kube-system",
		},
		namespace.Labels,
	)

	clientFactory.AssertExpectations(s.T())
}
