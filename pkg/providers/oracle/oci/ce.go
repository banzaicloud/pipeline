package oci

import (
	"context"
	"io/ioutil"
	"strings"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
	"github.com/oracle/oci-go-sdk/example/helpers"
)

// ContainerEngine
type ContainerEngine struct {
	oci             *OCI
	client          *containerengine.ContainerEngineClient
	CompartmentOCID string
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
func (c *ContainerEngine) getResourceID(resources []containerengine.WorkRequestResource, actionType containerengine.WorkRequestResourceActionTypeEnum, entityType string) *string {
	for _, resource := range resources {
		if resource.ActionType == actionType && strings.ToUpper(*resource.EntityType) == entityType {
			return resource.Identifier
		}
	}

	return nil
}

// wait until work request finish
func (c *ContainerEngine) waitUntilWorkRequestComplete(client containerengine.ContainerEngineClient, workReuqestID *string) (containerengine.GetWorkRequestResponse, error) {

	// retry GetWorkRequest call until TimeFinished is set
	shouldRetryFunc := func(r common.OCIOperationResponse) bool {
		return r.Response.(containerengine.GetWorkRequestResponse).TimeFinished == nil
	}

	getWorkReq := containerengine.GetWorkRequestRequest{
		WorkRequestId:   workReuqestID,
		RequestMetadata: helpers.GetRequestMetadataWithCustomizedRetryPolicy(shouldRetryFunc),
	}

	return client.GetWorkRequest(context.Background(), getWorkReq)
}

// GetAvailableKubernetesVersions gets available K8S versions
func (c *ContainerEngine) GetAvailableKubernetesVersions() (versions Strings, err error) {

	request := containerengine.GetClusterOptionsRequest{
		ClusterOptionId: common.String("all"),
	}

	r, err := c.client.GetClusterOptions(context.Background(), request)

	return Strings{
		strings: r.KubernetesVersions,
	}, err
}

// GetK8SConfig generates and downloads K8S config
func (c *ContainerEngine) GetK8SConfig(OCID string) ([]byte, error) {

	response, err := c.client.CreateKubeconfig(context.Background(), containerengine.CreateKubeconfigRequest{
		ClusterId: &OCID,
	})

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(response.Content)
}
