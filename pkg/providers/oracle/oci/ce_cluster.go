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
	"fmt"
	"time"

	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/go-errors/errors"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
	"github.com/spf13/viper"
)

// CreateCluster creates an OKE cluster specified in the request
func (ce *ContainerEngine) CreateCluster(request containerengine.CreateClusterRequest) (clusterOCID string, err error) {

	response, err := ce.client.CreateCluster(context.Background(), request)
	if err != nil {
		return clusterOCID, err
	}

	workReqResp, err := ce.waitUntilWorkRequestComplete(*ce.client, response.OpcWorkRequestId)
	if err != nil {
		return clusterOCID, err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return clusterOCID, fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	if workReqResp.WorkRequest.Status == containerengine.WorkRequestStatusSucceeded {
		clusterOCID = *ce.getResourceID(workReqResp.Resources, containerengine.WorkRequestResourceActionTypeCreated, "CLUSTER")
	}

	return clusterOCID, err
}

// UpdateCluster updates an OKE cluster specified in the request
func (ce *ContainerEngine) UpdateCluster(request containerengine.UpdateClusterRequest) (clusterOCID string, err error) {

	response, err := ce.client.UpdateCluster(context.Background(), request)
	if err != nil {
		return clusterOCID, err
	}

	workReqResp, err := ce.waitUntilWorkRequestComplete(*ce.client, response.OpcWorkRequestId)
	if err != nil {
		return clusterOCID, err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return clusterOCID, fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	if workReqResp.WorkRequest.Status == containerengine.WorkRequestStatusSucceeded {
		clusterOCID = *ce.getResourceID(workReqResp.Resources, containerengine.WorkRequestResourceActionTypeUpdated, "CLUSTER")
	}

	return clusterOCID, err
}

// DeleteCluster removes an OKE cluster specified in the request
func (ce *ContainerEngine) DeleteCluster(request containerengine.DeleteClusterRequest) (err error) {

	response, err := ce.client.DeleteCluster(context.Background(), request)
	if err != nil {
		return err
	}

	workReqResp, err := ce.waitUntilWorkRequestComplete(*ce.client, response.OpcWorkRequestId)
	if err != nil {
		return err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	return nil
}

// GetCluster gets a Cluster by id
func (ce *ContainerEngine) GetCluster(id *string) (cluster containerengine.Cluster, err error) {

	response, err := ce.client.GetCluster(context.Background(), containerengine.GetClusterRequest{
		ClusterId: id,
	})
	if err != nil {
		return cluster, err
	}

	return response.Cluster, nil
}

// GetClusterByName gets a Cluster by name within a Compartment
func (ce *ContainerEngine) GetClusterByName(name string) (cluster containerengine.ClusterSummary, err error) {

	clusters, err := ce.GetClustersByName(name)
	if err != nil {
		return cluster, err
	}

	if len(clusters) < 1 {
		return cluster, err
	}

	return clusters[0], nil
}

// GetClustersByName gets all Clusters by name within a Compartment
func (ce *ContainerEngine) GetClustersByName(name string) (clusters []containerengine.ClusterSummary, err error) {

	request := containerengine.ListClustersRequest{
		CompartmentId: common.String(ce.CompartmentOCID),
		Name:          common.String(name),
	}

	response, err := ce.client.ListClusters(context.Background(), request)
	if err != nil {
		return clusters, err
	}

	for _, item := range response.Items {
		clusters = append(clusters, item)
	}

	return clusters, err
}

// GetClusters gets all Clusters within the Compartment
func (ce *ContainerEngine) GetClusters() (clusters []containerengine.ClusterSummary, err error) {

	request := containerengine.ListClustersRequest{
		CompartmentId: common.String(ce.CompartmentOCID),
	}
	request.Limit = common.Int(20)

	listFunc := func(request containerengine.ListClustersRequest) (containerengine.ListClustersResponse, error) {
		return ce.client.ListClusters(context.Background(), request)
	}

	for r, err := listFunc(request); ; r, err = listFunc(request) {
		if err != nil {
			return clusters, err
		}

		for _, item := range r.Items {
			clusters = append(clusters, item)
		}

		if r.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = r.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return clusters, err
}

// WaitingForClusterNodePoolActiveState waits until every node in the pool is in ACTIVE state
func (ce *ContainerEngine) WaitingForClusterNodePoolActiveState(clusterID *string) error {

	ce.oci.logger.Info("Waiting for all nodepools state to be ACTIVE on all nodes")

	maxAttempts := viper.GetInt(pipConfig.OKEWaitAttemptsForNodepoolActive)
	sleepSeconds := viper.GetInt(pipConfig.OKESleepSecondsForNodepoolActive)

	for i := 0; i <= maxAttempts; i++ {

		time.Sleep(time.Duration(sleepSeconds) * time.Second)

		nodePools, err := ce.GetNodePools(clusterID)
		if err != nil {
			return err
		}

		ok := true
		for _, np := range nodePools {
			if !ce.IsNodePoolActive(np.Id) {
				ok = false
			}
		}

		if ok {
			return nil
		}
	}

	return errors.New("timeout during waiting for nodepools to activate")
}

// FilterClustersByNotInState filter cluster list by cluster state
func (ce *ContainerEngine) FilterClustersByNotInState(clusters []containerengine.ClusterSummary, state containerengine.ClusterSummaryLifecycleStateEnum) (filteredClusters []containerengine.ClusterSummary) {

	for _, cluster := range clusters {
		if cluster.LifecycleState != state {
			filteredClusters = append(filteredClusters, cluster)
		}
	}

	return filteredClusters
}
