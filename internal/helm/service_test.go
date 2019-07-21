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

package helm

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
)

const organizationName = "banzaicloud"

type clusterServiceStub struct {
	cluster Cluster
}

func (s *clusterServiceStub) GetCluster(ctx context.Context, clusterID uint) (*Cluster, error) {
	return &s.cluster, nil
}

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	kubeConfigFile := os.Getenv("KUBECONFIG")
	if kubeConfigFile == "" {
		t.Skip("skipping as Kubernetes config was not provided")
	}

	kubeConfigBytes, err := ioutil.ReadFile(kubeConfigFile)
	require.NoError(t, err)

	clusterService := &clusterServiceStub{
		cluster: Cluster{
			OrganizationName: organizationName,
			KubeConfig:       kubeConfigBytes,
		},
	}
	service := NewHelmService(clusterService, commonadapter.NewNoopLogger())

	err = service.InstallDeployment(
		context.Background(),
		1,
		"default",
		"banzaicloud-stable/banzaicloud-docs",
		"helm-service-test",
		[]byte{},
		"0.1.1",
		true,
	)
	require.NoError(t, err)

	values := map[string]interface{}{
		"replicaCount": 2,
	}

	valuesBytes, err := yaml.Marshal(values)
	require.NoError(t, err)

	err = service.UpdateDeployment(
		context.Background(),
		1,
		"default",
		"banzaicloud-stable/banzaicloud-docs",
		"helm-service-test",
		valuesBytes,
		"0.1.1",
	)
	require.NoError(t, err)

	// Wait for update to finish
	time.Sleep(5 * time.Second)

	err = service.DeleteDeployment(
		context.Background(),
		1,
		"helm-service-test",
	)
	require.NoError(t, err)
}
