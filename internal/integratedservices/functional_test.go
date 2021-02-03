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
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1/dns"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	integratedServiceDNS "github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/secret"
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
//
// make test-integrated-service-up
// make test-integrated-service-worker &
// pid=$!
// make test-integrated-service
// kill $pid
// make test-integrated-service-down

func (s *Suite) TestActivateBanzaiDNSWithoutSecret() {
	ctx, cancel := context.WithCancel(context.Background())
	s.T().Cleanup(func() {
		cancel()
	})

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	org := uint(r.Uint32())
	user := uint(r.Uint32())

	ctx = auth.SetCurrentOrganizationID(ctx, org)

	cluster, err := importCluster(s.kubeconfig, fmt.Sprintf("is-test-%d", org), org, user)
	s.Require().NoError(err)

	s.T().Logf("imported cluster id: %d", cluster.GetID())

	importedFakeCluster := importedCluster{KubeCluster: cluster}

	m := integratedServiceDNS.NewIntegratedServicesManager(
		importedFakeCluster,
		importedFakeCluster,
		s.config.Cluster.DNS.Config)

	var integratedServicesService integratedservices.Service
	if s.v2 {
		// TODO implement workflow check to see when it failed, until then we have to skip
		s.T().Skip()
		integratedServicesService, err = s.integratedServiceServiceCreaterV2(importedFakeCluster.KubeConfig, m)
	} else {
		integratedServicesService, err = s.integratedServiceServiceCreater(m)
	}
	s.Require().NoError(err)

	spec := map[string]interface{}{
		"clusterDomain": "asd",
		"externalDns": map[string]interface{}{
			"provider": map[string]string{
				"name": "banzaicloud-dns",
			},
		},
	}

	err = integratedServicesService.Activate(ctx, cluster.GetID(), integratedServiceDNS.IntegratedServiceName, spec)
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
					s.T().Fatal("unexpected active status")
				case integratedservices.IntegratedServiceStatusError:
					s.T().Logf("integrated service activation failed, but this is expected")
					return true
				}
				s.T().Logf("is status %s", i.Status)
			}
		}
		return false
	}, time.Second*30, time.Second*2)

	details, err := integratedServicesService.Details(ctx, cluster.GetID(), integratedServiceDNS.IntegratedServiceName)
	s.Require().NoError(err)

	originalTypedSpec := &dns.ServiceSpec{}
	s.Require().NoError(services.BindIntegratedServiceSpec(spec, originalTypedSpec))

	returnedTypedSpec := &dns.ServiceSpec{}
	s.Require().NoError(services.BindIntegratedServiceSpec(details.Spec, returnedTypedSpec))

	// Check that details contains the same spec as it was when created
	s.Require().Equal(originalTypedSpec, returnedTypedSpec)
}

func (s *Suite) TestActivateGoogleDNSWithFakeSecret() {
	ctx, cancel := context.WithCancel(context.Background())
	s.T().Cleanup(func() {
		cancel()
	})

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	org := uint(r.Uint32())
	user := uint(r.Uint32())

	ctx = auth.SetCurrentOrganizationID(ctx, org)

	cluster, err := importCluster(s.kubeconfig, fmt.Sprintf("is-test-%d", org), org, user)
	s.Require().NoError(err)

	s.T().Logf("imported cluster id: %d", cluster.GetID())

	importedFakeCluster := importedCluster{KubeCluster: cluster}

	m := integratedServiceDNS.NewIntegratedServicesManager(
		importedFakeCluster,
		importedFakeCluster,
		s.config.Cluster.DNS.Config)

	var integratedServicesService integratedservices.Service
	if s.v2 {
		integratedServicesService, err = s.integratedServiceServiceCreaterV2(importedFakeCluster.KubeConfig, m)
	} else {
		integratedServicesService, err = s.integratedServiceServiceCreater(m)
	}
	s.Require().NoError(err)

	createSecretRequest := secret.CreateSecretRequest{
		Name: "fake-dns-secret",
		Type: secrettype.Google,
		Values: map[string]string{
			"type":                        "fake-type",
			"project_id":                  "fake-project_id",
			"private_key_id":              "fake-private_key_id",
			"private_key":                 "fake-private_key",
			"client_email":                "fake-client_email",
			"client_id":                   "fake-client_id",
			"auth_uri":                    "fake-auth_uri",
			"token_uri":                   "fake-token_uri",
			"auth_provider_x509_cert_url": "fake-auth_provider_x509_cert_url",
			"client_x509_cert_url":        "fake-client_x509_cert_url",
		},
	}

	fakeSecretId, err := secret.Store.Store(org, &createSecretRequest)
	s.Require().NoError(err)

	var createdOutput, updatedOutput integratedservices.IntegratedServiceOutput
	{
		spec := map[string]interface{}{
			"clusterDomain": "asd",
			"externalDns": map[string]interface{}{
				"provider": map[string]interface{}{
					"name":     "google",
					"secretId": fakeSecretId,
					"options": map[string]string{
						"project": "google",
					},
				},
			},
		}

		err = integratedServicesService.Activate(ctx, cluster.GetID(), integratedServiceDNS.IntegratedServiceName, spec)
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
						s.T().Fatal("unexpected error status")
					}
					s.T().Logf("is status %s", i.Status)
				}
			}
			return false
		}, time.Second*60, time.Second*2)

		details, err := integratedServicesService.Details(ctx, cluster.GetID(), integratedServiceDNS.IntegratedServiceName)
		s.Require().NoError(err)

		createdOutput = details.Output

		originalTypedSpec := &dns.ServiceSpec{}
		s.Require().NoError(services.BindIntegratedServiceSpec(spec, originalTypedSpec))

		returnedTypedSpec := &dns.ServiceSpec{}
		s.Require().NoError(services.BindIntegratedServiceSpec(details.Spec, returnedTypedSpec))

		// TXTOwnerID and RBACEnabled is set dynamically so unset here
		returnedTypedSpec.ExternalDNS.TXTOwnerID = ""
		returnedTypedSpec.RBACEnabled = false

		// Check that details contains the same spec as it was when created
		s.Require().Equal(originalTypedSpec, returnedTypedSpec)
	}

	// CHECK UPDATE
	{
		spec := map[string]interface{}{
			"clusterDomain": "asd",
			"externalDns": map[string]interface{}{
				"provider": map[string]interface{}{
					"name":     "google",
					"secretId": fakeSecretId,
					"options": map[string]string{
						"project": "google",
					},
				},
				"sources": []string{"service", "ingress"},
			},
		}

		err = integratedServicesService.Update(ctx, cluster.GetID(), integratedServiceDNS.IntegratedServiceName, spec)
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
						s.T().Fatal("unexpected error status")
					}
					s.T().Logf("is status %s", i.Status)
				}
			}
			return false
		}, time.Second*30, time.Second*2)

		details, err := integratedServicesService.Details(ctx, cluster.GetID(), integratedServiceDNS.IntegratedServiceName)
		s.Require().NoError(err)

		updatedOutput = details.Output

		originalTypedSpec := &dns.ServiceSpec{}
		s.Require().NoError(services.BindIntegratedServiceSpec(spec, originalTypedSpec))

		returnedTypedSpec := &dns.ServiceSpec{}
		s.Require().NoError(services.BindIntegratedServiceSpec(details.Spec, returnedTypedSpec))

		// TXTOwnerID and RBACEnabled is set dynamically so unset here
		returnedTypedSpec.ExternalDNS.TXTOwnerID = ""
		returnedTypedSpec.RBACEnabled = false

		// Check that details contains the same spec as it was when created
		s.Assert().Equal(originalTypedSpec, returnedTypedSpec)
	}

	s.Assert().Equal(createdOutput, updatedOutput)

	kubeConfig, err := importedFakeCluster.GetK8sConfig()
	s.Require().NoError(err)
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	s.Require().NoError(err)

	deploymentName := "dns-external-dns"

	s.Require().Eventually(func() bool {
		_, err = client.AppsV1().Deployments(s.config.Cluster.Namespace).
			Get(context.TODO(), deploymentName, v1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				s.FailNow(err.Error())
			}
			return false
		}
		return true
	}, time.Second*30, time.Second*2, "external-dns deployment created")

	err = integratedServicesService.Deactivate(ctx, cluster.GetID(), integratedServiceDNS.IntegratedServiceName)
	s.Require().NoError(err)

	s.Require().Eventually(func() bool {
		_, err = client.AppsV1().Deployments(s.config.Cluster.Namespace).
			Get(context.TODO(), deploymentName, v1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				s.Error(err)
			}
			return true
		}
		return false
	}, time.Second*30, time.Second*2, "external-dns deployment created")
}

func TestV1(t *testing.T) {
	suite.Run(t, new(Suite))
}

func TestV2(t *testing.T) {
	s := new(Suite)
	s.v2 = true
	suite.Run(t, s)
}
