package test

import (
	"testing"
	"github.com/banzaicloud/azure-aks-client/client"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"fmt"
)

const name = "test_cluster"
const resourceGroup = "rg1"

func TestCreateCluster(t *testing.T) {

	fmt.Println(" --- [ Testing creation ] ---")

	c := cluster.CreateClusterRequest{
		Name:              name,
		Location:          "eastus",
		VMSize:            "Standard_D2_v2",
		ResourceGroup:     resourceGroup,
		AgentCount:        1,
		AgentName:         "agentpool1",
		KubernetesVersion: "1.7.7",
	}

	if resp, err := client.CreateUpdateCluster(c); err != nil {
		t.Errorf("Error is NOT <nil>: %s.", err)
	} else if resp.Value.Name != name {
		t.Errorf("Expected cluster name is %v but got %v.", name, resp.Value.Name)
	}

}

func TestPollingCluster(t *testing.T) {

	fmt.Println(" --- [ Testing polling ] ---")

	if _, err := client.PollingCluster(name, resourceGroup); err != nil {
		t.Errorf("Error is NOT <nil>: %s. Polling failed.", err)
	}

}

func TestListCluster(t *testing.T) {

	fmt.Println(" --- [ Testing listing ] ---")

	if resp, err := client.ListClusters(resourceGroup); err != nil {
		t.Errorf("Error is NOT <nil>: %s. Listing failed.", err)
	} else {

		isContains := false
		for i := 0; i < len(resp.Value.Value); i++ {
			v := resp.Value.Value[i]
			if v.Name == name {
				isContains = true
				break
			}
		}

		if !isContains {
			t.Errorf("The list not contains %v in %v", name, resourceGroup)
		}
	}

}

func TestDeleteCluster(t *testing.T) {

	fmt.Println(" --- [ Testing delete ] ---")

	if resp, ok := client.DeleteCluster(name, resourceGroup); !ok {
		t.Errorf("Delete failed: %s.", resp)
	}

}
