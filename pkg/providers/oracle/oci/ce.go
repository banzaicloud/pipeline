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

package oci

import (
	"context"
	"io/ioutil"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
)

// ContainerEngine is for managing OKE related calls of OCI
type ContainerEngine struct {
	CompartmentOCID string

	oci    *OCI
	client *containerengine.ContainerEngineClient
}

// NewContainerEngineClient creates a new ContainerEngine
func (oci *OCI) NewContainerEngineClient() (client *ContainerEngine, err error) {
	client = &ContainerEngine{}

	oClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(oci.config)
	if err != nil {
		return client, err
	}

	client.client = &oClient
	client.oci = oci
	client.CompartmentOCID = oci.CompartmentOCID

	return client, nil
}

// getResourceID return a resource ID based on the filter of resource actionType and entityType
func (ce *ContainerEngine) getResourceID(resources []containerengine.WorkRequestResource, actionType containerengine.WorkRequestResourceActionTypeEnum, entityType string) *string {
	for _, resource := range resources {
		if resource.ActionType == actionType && strings.ToUpper(*resource.EntityType) == entityType {
			return resource.Identifier
		}
	}

	return nil
}

// wait until work request finish
func (ce *ContainerEngine) waitUntilWorkRequestComplete(client containerengine.ContainerEngineClient, workReuqestID *string) (containerengine.GetWorkRequestResponse, error) {
	// retry GetWorkRequest call until TimeFinished is set
	policy := common.NewRetryPolicy(uint(180), func(r common.OCIOperationResponse) bool {
		return r.Response.(containerengine.GetWorkRequestResponse).TimeFinished == nil
	}, func(r common.OCIOperationResponse) time.Duration {
		return time.Duration(uint(10)) * time.Second
	})

	getWorkReq := containerengine.GetWorkRequestRequest{
		WorkRequestId: workReuqestID,
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: &policy,
		},
	}

	return client.GetWorkRequest(context.Background(), getWorkReq)
}

// GetAvailableKubernetesVersions gets available K8S versions
func (ce *ContainerEngine) GetAvailableKubernetesVersions() (versions Strings, err error) {
	request := containerengine.GetClusterOptionsRequest{
		ClusterOptionId: common.String("all"),
	}

	r, err := ce.client.GetClusterOptions(context.Background(), request)

	return Strings{
		strings: r.KubernetesVersions,
	}, err
}

// GetK8SConfig generates and downloads K8S config
func (ce *ContainerEngine) GetK8SConfig(OCID string) ([]byte, error) {
	response, err := ce.client.CreateKubeconfig(context.Background(), containerengine.CreateKubeconfigRequest{
		ClusterId: &OCID,
	})

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(response.Content)
}
