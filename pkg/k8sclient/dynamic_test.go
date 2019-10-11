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

package k8sclient

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/testing_frameworks/integration"
)

type DynamicFileClientTestSuite struct {
	suite.Suite

	controlPlane *integration.ControlPlane

	client DynamicFileClient
}

func testDynamicFileClient(t *testing.T) {
	if os.Getenv("TEST_ASSET_KUBE_APISERVER") == "" || os.Getenv("TEST_ASSET_ETCD") != "" {
		t.Skip("control plane binaries are missing")
	}

	suite.Run(t, new(DynamicFileClientTestSuite))
}

func (s *DynamicFileClientTestSuite) SetupSuite() {
	s.controlPlane = &integration.ControlPlane{}

	err := s.controlPlane.Start()
	s.Require().NoError(err)
}

func (s *DynamicFileClientTestSuite) TearDownSuite() {
	s.controlPlane.Stop()
}

func (s *DynamicFileClientTestSuite) SetupTest() {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: s.controlPlane.APIURL().String()}},
	).ClientConfig()
	s.Require().NoError(err)

	runtimeClient, err := client.New(config, client.Options{})
	s.Require().NoError(err)

	s.client = NewDynamicFileClient(runtimeClient)
}

func (s *DynamicFileClientTestSuite) Test_Create() {
	yaml := `apiVersion: v1
kind: Namespace
metadata:
  name: test
`

	err := s.client.Create(context.Background(), []byte(yaml))
	s.Require().NoError(err)
}
