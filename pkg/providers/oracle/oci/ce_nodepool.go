package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
)

// GetNodePool gets a node pool by it's OCID
func (c *ContainerEngine) GetNodePool(OCID string) (nodepool containerengine.NodePool, err error) {

	response, err := c.client.GetNodePool(context.Background(), containerengine.GetNodePoolRequest{
		NodePoolId: &OCID,
	})

	if err != nil {
		return nodepool, err
	}

	return response.NodePool, nil
}

// CreateNodePool creates node pool specified in the request
func (c *ContainerEngine) CreateNodePool(request containerengine.CreateNodePoolRequest) (nodepoolOCID string, err error) {

	ctx := context.Background()

	response, err := c.client.CreateNodePool(ctx, request)
	if err != nil {
		return nodepoolOCID, err
	}

	workReqResp, err := c.waitUntilWorkRequestComplete(*c.client, response.OpcWorkRequestId)
	if err != nil {
		return nodepoolOCID, err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return nodepoolOCID, fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	if workReqResp.WorkRequest.Status == containerengine.WorkRequestStatusSucceeded {
		nodepoolOCID = *c.getResourceID(workReqResp.Resources, containerengine.WorkRequestResourceActionTypeCreated, "NODEPOOL")
	}

	return nodepoolOCID, err
}

// DeleteClusterNodePoolByName deletes a node pool in a cluster specified by it's name
func (c *ContainerEngine) DeleteClusterNodePoolByName(clusterID string, name string) error {

	nodePools, err := c.ListClusterNodePoolsByName(clusterID, name)
	if err != nil {
		return err
	}

	if len(nodePools) == 0 {
		return nil
	}

	if len(nodePools) != 1 {
		return fmt.Errorf("More than 1 Node Pools with name %s", name)
	}

	nodePool := nodePools[0]

	request := containerengine.DeleteNodePoolRequest{
		NodePoolId: nodePool.Id,
	}

	c.oci.GetLogger().Infof("Deleting NodePool[%s]", *nodePool.Name)
	c.DeleteNodePool(request)

	return nil
}

// DeleteNodePool deletes a node pool specified in the request
func (c *ContainerEngine) DeleteNodePool(request containerengine.DeleteNodePoolRequest) error {

	response, err := c.client.DeleteNodePool(context.Background(), request)
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

// UpdateNodePool updates a node pool specified in a request
func (c *ContainerEngine) UpdateNodePool(request containerengine.UpdateNodePoolRequest) (nodepoolOCID string, err error) {

	response, err := c.client.UpdateNodePool(context.Background(), request)
	if err != nil {
		return nodepoolOCID, err
	}

	workReqResp, err := c.waitUntilWorkRequestComplete(*c.client, response.OpcWorkRequestId)
	if err != nil {
		return nodepoolOCID, err
	}

	if workReqResp.WorkRequest.Status != containerengine.WorkRequestStatusSucceeded {
		return nodepoolOCID, fmt.Errorf("WorkReqResp status: %s", workReqResp.WorkRequest.Status)
	}

	if workReqResp.WorkRequest.Status == containerengine.WorkRequestStatusSucceeded {
		nodepoolOCID = *c.getResourceID(workReqResp.Resources, containerengine.WorkRequestResourceActionTypeUpdated, "NODEPOOL")
	}

	return nodepoolOCID, err
}

// ListClusterNodePoolsByName gets node pools by cluster OCID and name
func (c *ContainerEngine) ListClusterNodePoolsByName(clusterOCID string, name string) (nodepools []containerengine.NodePoolSummary, err error) {
	request := containerengine.ListNodePoolsRequest{
		CompartmentId: common.String(c.CompartmentOCID),
		ClusterId:     common.String(clusterOCID),
		Name:          common.String(name),
	}

	return c.listNodePools(request)
}

// ListClusterNodePools gets node pools by cluster OCID
func (c *ContainerEngine) ListClusterNodePools(clusterOCID string) (nodepools []containerengine.NodePoolSummary, err error) {
	request := containerengine.ListNodePoolsRequest{
		CompartmentId: common.String(c.CompartmentOCID),
		ClusterId:     common.String(clusterOCID),
	}

	return c.listNodePools(request)
}

// IsNodePoolActive checks whether every node is in ACTIVE not DELETED state in a node pool
func (c *ContainerEngine) IsNodePoolActive(OCID string) bool {

	np, err := c.GetNodePool(OCID)
	if err != nil {
		return false
	}

	neededCount := len(np.SubnetIds) * *np.QuantityPerSubnet

	activeNodes := 0
	for _, n := range np.Nodes {
		if n.LifecycleState == containerengine.NodeLifecycleStateDeleted {
			continue
		}
		if n.LifecycleState == containerengine.NodeLifecycleStateActive {
			activeNodes++
		} else {
			c.oci.logger.Debugf("Node state: %s (%s)", n.LifecycleState, *n.LifecycleDetails)
			break
		}
	}

	if activeNodes == neededCount {
		c.oci.logger.Infof("All nodes are in ACTIVE state in NodePool[%s]", *np.Name)
		return true
	}

	return false
}

// GetDefaultNodePoolOptions gets default node pool options
func (c *ContainerEngine) GetDefaultNodePoolOptions() (options NodePoolOptions, err error) {

	return c.GetNodePoolOptions("all")
}

// GetNodePoolOptions gets available node pool options for a specified cluster OCID
func (c *ContainerEngine) GetNodePoolOptions(clusterID string) (options NodePoolOptions, err error) {

	request := containerengine.GetNodePoolOptionsRequest{
		NodePoolOptionId: &clusterID,
	}

	r, err := c.client.GetNodePoolOptions(context.Background(), request)

	return NodePoolOptions{
		Images:             Strings{strings: r.Images},
		KubernetesVersions: Strings{strings: r.KubernetesVersions},
		Shapes:             Strings{strings: r.Shapes},
	}, err
}

func (c *ContainerEngine) listNodePools(request containerengine.ListNodePoolsRequest) (nodepools []containerengine.NodePoolSummary, err error) {

	request.Limit = common.Int(50)

	listFunc := func(request containerengine.ListNodePoolsRequest) (containerengine.ListNodePoolsResponse, error) {
		return c.client.ListNodePools(context.Background(), request)
	}

	nodepools = make([]containerengine.NodePoolSummary, 0)
	for r, err := listFunc(request); ; r, err = listFunc(request) {
		if err != nil {
			return nodepools, err
		}

		for _, item := range r.Items {
			nodepools = append(nodepools, item)
		}

		if r.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = r.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return nodepools, err
}
