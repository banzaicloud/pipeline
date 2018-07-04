package oci

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
)

// GetCluster gets cluster by it's OCID
func (c *ContainerEngine) GetCluster(OCID string) (cluster containerengine.Cluster, err error) {

	response, err := c.client.GetCluster(context.Background(), containerengine.GetClusterRequest{
		ClusterId: &OCID,
	})
	if err != nil {
		return cluster, err
	}

	return response.Cluster, nil
}

// DeleteCluster removes an OKE cluster specified in the request
func (c *ContainerEngine) DeleteCluster(request containerengine.DeleteClusterRequest) (err error) {

	response, err := c.client.DeleteCluster(context.Background(), request)
	if err != nil {
		return err
	}

	workReqResp, err := c.waitUntilWorkRequestComplete(*c.client, response.OpcWorkRequestId)
	if err != nil {
		return err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	return nil
}

// CreateCluster creates an OKE cluster specified in the request
func (c *ContainerEngine) CreateCluster(request containerengine.CreateClusterRequest) (clusterOCID string, err error) {

	response, err := c.client.CreateCluster(context.Background(), request)
	if err != nil {
		return clusterOCID, err
	}

	workReqResp, err := c.waitUntilWorkRequestComplete(*c.client, response.OpcWorkRequestId)
	if err != nil {
		return clusterOCID, err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return clusterOCID, fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	if workReqResp.WorkRequest.Status == containerengine.WorkRequestStatusSucceeded {
		clusterOCID = *c.getResourceID(workReqResp.Resources, containerengine.WorkRequestResourceActionTypeCreated, "CLUSTER")
	}

	return clusterOCID, err
}

// UpdateCluster updates an OKE cluster specified in the request
func (c *ContainerEngine) UpdateCluster(request containerengine.UpdateClusterRequest) (clusterOCID string, err error) {

	response, err := c.client.UpdateCluster(context.Background(), request)
	if err != nil {
		return clusterOCID, err
	}

	workReqResp, err := c.waitUntilWorkRequestComplete(*c.client, response.OpcWorkRequestId)
	if err != nil {
		return clusterOCID, err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return clusterOCID, fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	if workReqResp.WorkRequest.Status == containerengine.WorkRequestStatusSucceeded {
		clusterOCID = *c.getResourceID(workReqResp.Resources, containerengine.WorkRequestResourceActionTypeUpdated, "CLUSTER")
	}

	return clusterOCID, err
}

// WaitingForClusterNodePoolActiveState waits until every node in the pool is in ACTIVE state
func (c *ContainerEngine) WaitingForClusterNodePoolActiveState(OCID string) error {

	c.oci.logger.Info("Waiting for all nodepools state to be ACTIVE on all nodes")

	for i := 0; i <= 60; i++ {

		time.Sleep(time.Duration(20) * time.Second)

		nodePools, err := c.ListClusterNodePools(OCID)
		if err != nil {
			return err
		}

		ok := true
		for _, np := range nodePools {
			if !c.IsNodePoolActive(*np.Id) {
				ok = false
			}
		}

		if ok {
			return nil
		}
	}

	return fmt.Errorf("Timeout during waiting for nodepools to activate")
}

// ListClusterByName gets clusters name
func (c *ContainerEngine) ListClusterByName(name string) (clusters []containerengine.ClusterSummary, err error) {
	request := containerengine.ListClustersRequest{
		CompartmentId: common.String(c.CompartmentOCID),
		Name:          common.String(name),
	}

	return c.listClusters(request)
}

// FilterClustersByNotInState filter cluster list by cluster state
func (c *ContainerEngine) FilterClustersByNotInState(clusters []containerengine.ClusterSummary, state containerengine.ClusterSummaryLifecycleStateEnum) (filteredClusters []containerengine.ClusterSummary) {

	for _, cluster := range clusters {
		if cluster.LifecycleState != state {
			filteredClusters = append(filteredClusters, cluster)
		}
	}

	return filteredClusters
}

func (c *ContainerEngine) listClusters(request containerengine.ListClustersRequest) (clusters []containerengine.ClusterSummary, err error) {

	request.Limit = common.Int(50)

	listFunc := func(request containerengine.ListClustersRequest) (containerengine.ListClustersResponse, error) {
		return c.client.ListClusters(context.Background(), request)
	}

	clusters = make([]containerengine.ClusterSummary, 0)
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
