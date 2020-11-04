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

package integratedservices_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	integratedServiceDNS "github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/stretchr/testify/suite"
)

// These tests aim to verify that Integrated Service API specifications are met.
// Considering efficiency as a key aspect, tests are executed against the highest logical, but still internal layer of the
// Pipeline Integrated Service API, which means that the http machinery is completely bypassed.
// There is no user authentication, users and organizations are represented by fake entities.
// Although the Pipeline Web - and any UI component - is not required for these tests to run certain dependencies are:
// MySQL, Vault, Cadence as external dependencies (launched using docker-compose for example).
// (Dex should not be required for these tests, but there is no way to avoid it right now)
// A running Pipeline Worker configured with the same external dependencies is also required, but to make debugging easier
// we don't make an assumption on how the worker is started. It is recommended to run the worker with the same codebase
// using the same config: testconfig/config.yaml
//
// Example how to trigger this using make:
// make test-integrated-service-functional-worker & ; pid=$! ; make test-integrated-service-functional ; kill $pid

func (s *Suite) TestActivateDisabledDNSShouldFail() {
	ctx, cancel := context.WithCancel(context.Background())
	s.T().Cleanup(func() {
		cancel()
	})

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	org := uint(r.Uint32())
	user := uint(r.Uint32())

	cluster, err := importCluster(s.kubeconfigPath, "is-test", org, user)
	s.Require().NoError(err)

	s.T().Logf("imported cluster id: %d", cluster.GetID())

	m := integratedServiceDNS.NewIntegratedServicesManager(
		importedCluster{KubeCluster: cluster},
		importedCluster{KubeCluster: cluster},
		s.config.Cluster.DNS.Config)

	integratedServicesService, err := s.integratedServiceServiceCreater(m)
	s.Require().NoError(err)

	err = integratedServicesService.Activate(ctx, cluster.GetID(), integratedServiceDNS.IntegratedServiceName, map[string]interface{}{
		"clusterDomain": "asd",
		"externalDns": map[string]interface{}{
			"provider": map[string]string{
				"name": "banzaicloud-dns",
			},
		},
	})
	s.Require().NoError(err)

	s.Require().Eventually(func() bool {
		isList, err := integratedServicesService.List(ctx, cluster.GetID())
		if err != nil {
			s.T().Fatalf("%+v", err)
		}
		for _, i := range isList {
			if i.Name == integratedServiceDNS.IntegratedServiceName {
				switch i.Status {
				case integratedservices.IntegratedServiceStatusActive:
					return true
				case integratedservices.IntegratedServiceStatusError:
					s.T().Logf("integrated service activation failed, but this is expected")
					return true
				}
				s.T().Logf("is status %s", i.Status)
			}
		}
		return false
	}, time.Second*30, time.Second*2)
}

func TestFunctional(t *testing.T) {
	suite.Run(t, new(Suite))
}
