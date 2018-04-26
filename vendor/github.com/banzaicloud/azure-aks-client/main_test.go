package main_test

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/banzaicloud/azure-aks-client/client"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/azure-aks-client/utils"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/constants"
	"net/http"
	"reflect"
	"testing"
)

const (
	k8sVersion = "1.8.2"

	location1 = "eastus"
	location2 = "westus2"

	vmSize1 = "Standard_B2ms"
	vmSize2 = "Basic_A2"

	name              = "test_name"
	nameTooLong       = "testnametestnametestnametestnametestnametestname"
	nameBad1          = "testName"
	id                = "testid"
	provisioningState = "Succeeded"
	fqdn              = "fqdn"
	rg                = "rg"

	roleName = "testRoleName"
)

var (
	agentCount      = 1
	agentCountInt32 = int32(agentCount)
	agentName       = "agentName"
)

var kubeconfig = []byte("testkubeconfig")

var mc = containerservice.ManagedCluster{
	Response: autorest.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	},
	ManagedClusterProperties: &containerservice.ManagedClusterProperties{
		ProvisioningState: utils.S(provisioningState),
		Fqdn:              utils.S(fqdn),
	},
	ID:       utils.S(id),
	Name:     utils.S(name),
	Location: utils.S(location1),
}

type TestCluster struct {
}

func (t *TestCluster) CreateOrUpdate(request *cluster.CreateClusterRequest, managedCluster *containerservice.ManagedCluster) (*azure.ResponseWithValue, error) {
	return createResponse, nil
}

func (t *TestCluster) Delete(resourceGroup, name string) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusAccepted,
	}, nil
}

func (t *TestCluster) Get(resourceGroup, name string) (containerservice.ManagedCluster, error) {
	return mc, nil
}

func (t *TestCluster) List() ([]containerservice.ManagedCluster, error) {
	cl, err := t.Get(rg, name)
	if err != nil {
		return nil, err
	}
	return []containerservice.ManagedCluster{
		cl,
	}, nil
}

func (t *TestCluster) GetAccessProfiles(resourceGroup, name, roleName string) (containerservice.ManagedClusterAccessProfile, error) {
	return containerservice.ManagedClusterAccessProfile{
		Response: autorest.Response{
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		},
		AccessProfile: &containerservice.AccessProfile{
			KubeConfig: &kubeconfig,
		},
		ID:       utils.S(id),
		Name:     utils.S(name),
		Location: utils.S(location1),
	}, nil
}

func (t *TestCluster) ListLocations() (subscriptions.LocationListResult, error) {
	return subscriptions.LocationListResult{
		Value: &[]subscriptions.Location{
			{
				Name: utils.S(location1),
			},
			{
				Name: utils.S(location2),
			},
		},
	}, nil
}

func (t *TestCluster) ListVmSizes(location string) (result compute.VirtualMachineSizeListResult, err error) {
	return compute.VirtualMachineSizeListResult{
		Value: &[]compute.VirtualMachineSize{
			{
				Name: utils.S(vmSize1),
			},
			{
				Name: utils.S(vmSize2),
			},
		},
	}, nil
}

func (t *TestCluster) ListVersions(locations, resourceType string) (result containerservice.OrchestratorVersionProfileListResult, err error) {
	return containerservice.OrchestratorVersionProfileListResult{
		OrchestratorVersionProfileProperties: &containerservice.OrchestratorVersionProfileProperties{
			Orchestrators: &[]containerservice.OrchestratorVersionProfile{
				{
					OrchestratorType:    utils.S(string(compute.Kubernetes)),
					OrchestratorVersion: utils.S(k8sVersion),
				},
			},
		},
	}, nil
}

func (t *TestCluster) GetClientId() string     { return "testClientId" }
func (t *TestCluster) GetClientSecret() string { return "testClientSecret" }

func (t *TestCluster) LogDebug(args ...interface{})                 {}
func (t *TestCluster) LogInfo(args ...interface{})                  {}
func (t *TestCluster) LogWarn(args ...interface{})                  {}
func (t *TestCluster) LogError(args ...interface{})                 {}
func (t *TestCluster) LogFatal(args ...interface{})                 {}
func (t *TestCluster) LogPanic(args ...interface{})                 {}
func (t *TestCluster) LogDebugf(format string, args ...interface{}) {}
func (t *TestCluster) LogInfof(format string, args ...interface{})  {}
func (t *TestCluster) LogWarnf(format string, args ...interface{})  {}
func (t *TestCluster) LogErrorf(format string, args ...interface{}) {}
func (t *TestCluster) LogFatalf(format string, args ...interface{}) {}
func (t *TestCluster) LogPanicf(format string, args ...interface{}) {}

var manager = &TestCluster{}

func TestCreateOrUpdate(t *testing.T) {

	cases := []struct {
		name        string
		request     *cluster.CreateClusterRequest
		expResponse *azure.ResponseWithValue
		error
	}{
		{name: "full create", request: createRequest, expResponse: createResponse, error: nil},
		{name: "empty name", request: createRequestEmptyName, expResponse: nil, error: constants.ErrorAzureClusterNameEmpty},
		{name: "too long name", request: createRequestTooLongName, expResponse: nil, error: constants.ErrorAzureClusterNameTooLong},
		{name: "regexp name", request: createRequestWrongName, expResponse: nil, error: constants.ErrorAzureClusterNameRegexp},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if managedCluster, err := client.CreateUpdateCluster(manager, tc.request); err != nil {
				if tc.error == nil {
					t.Errorf("Error during create cluster: %s", err.Error())
					t.FailNow()
				} else if !reflect.DeepEqual(tc.error, err) {
					t.Errorf("Expected error: %s, but got: %s", tc.error, err.Error())
					t.FailNow()
				}
			} else if !reflect.DeepEqual(tc.expResponse, managedCluster) {
				t.Errorf("Expected cluster: %v, but got: %v", tc.expResponse, managedCluster)
				t.FailNow()
			}
		})
	}

}

func TestDeleteCluster(t *testing.T) {
	if err := client.DeleteCluster(manager, name, rg); err != nil {
		t.Errorf("Error during deleting cluster: %s", err.Error())
		t.FailNow()
	}
}

func TestGetClusterConfig(t *testing.T) {

	exp := &azure.Config{
		Location: location1,
		Name:     name,
		Properties: struct {
			KubeConfig string `json:"kubeConfig"`
		}{
			KubeConfig: string(kubeconfig),
		},
	}

	config, err := client.GetClusterConfig(manager, name, rg, roleName)
	if err != nil {
		t.Errorf("Error during getting config: %s", err.Error())
		t.FailNow()
	} else if !reflect.DeepEqual(exp, config) {
		t.Errorf("Expected config: %v, but got: %v", exp, config)
		t.FailNow()
	}

}

func TestListClusters(t *testing.T) {
	exp := &azure.ListResponse{
		StatusCode: http.StatusOK,
		Value: azure.Values{
			Value: []azure.Value{
				{
					Id:       id,
					Location: location1,
					Name:     name,
					Properties: azure.Properties{
						ProvisioningState: provisioningState,
						AgentPoolProfiles: nil,
						Fqdn:              fqdn,
					},
				},
			},
		},
	}

	if cl, err := client.ListClusters(manager); err != nil {
		t.Errorf("Error during listing cluster: %s", err.Error())
		t.FailNow()
	} else if !reflect.DeepEqual(exp, cl) {
		t.Errorf("Expected clusters: %v, but got: %v", exp, cl)
		t.FailNow()
	}
}

func TestGetCluster(t *testing.T) {
	exp := createResponse

	if cl, err := client.GetCluster(manager, name, rg); err != nil {
		t.Errorf("Error during getting cluster: %s", err.Error())
		t.FailNow()
	} else if !reflect.DeepEqual(exp, cl) {
		t.Errorf("Expected cluster: %v, but got: %v", exp, cl)
		t.FailNow()
	}
}

func TestPollingCluster(t *testing.T) {
	exp := pollingResponse

	if cl, err := client.PollingCluster(manager, name, rg); err != nil {
		t.Errorf("Error during polling cluster: %s", err.Error())
		t.FailNow()
	} else if !reflect.DeepEqual(exp, cl) {
		t.Errorf("Expected cluster: %v, but got: %v", exp, cl)
		t.FailNow()
	}
}

func TestGetLocations(t *testing.T) {

	exp := []string{
		location1,
		location2,
	}

	if locations, err := client.GetLocations(manager); err != nil {
		t.Errorf("Error during getting locations: %s", err.Error())
		t.FailNow()
	} else if !reflect.DeepEqual(exp, locations) {
		t.Errorf("Expected locations: %v, but got: %v", exp, locations)
		t.FailNow()
	}
}

func TestGetVmSizes(t *testing.T) {

	exp := []string{
		vmSize1,
		vmSize2,
	}

	vmSizes, err := client.GetVmSizes(manager, "eastus")
	if err != nil {
		t.Errorf("Error during getting vm sizes: %s", err.Error())
		t.FailNow()
	} else if !reflect.DeepEqual(vmSizes, exp) {
		t.Errorf("Expected vm sizes: %v, but got: %v", exp, vmSizes)
		t.FailNow()
	}
}

func TestGetKubernetesVersions(t *testing.T) {

	exp := []string{
		k8sVersion,
	}

	versions, err := client.GetKubernetesVersions(manager, "eastus")
	if err != nil {
		t.Errorf("Error during getting k8s versions: %s", err.Error())
		t.FailNow()
	} else if !reflect.DeepEqual(versions, exp) {
		t.Errorf("Expected versions: %v, but got: %v", exp, versions)
		t.FailNow()
	}
}

var (
	createRequest = &cluster.CreateClusterRequest{
		Name:              name,
		Location:          location1,
		ResourceGroup:     rg,
		KubernetesVersion: k8sVersion,
		Profiles: []containerservice.AgentPoolProfile{
			{
				Name:   &agentName,
				Count:  &agentCountInt32,
				VMSize: vmSize1,
			},
		},
	}

	createRequestEmptyName = &cluster.CreateClusterRequest{
		Location:          location1,
		ResourceGroup:     rg,
		KubernetesVersion: k8sVersion,
		Profiles: []containerservice.AgentPoolProfile{
			{
				Name:   &agentName,
				Count:  &agentCountInt32,
				VMSize: vmSize1,
			},
		},
	}

	createRequestTooLongName = &cluster.CreateClusterRequest{
		Name:              nameTooLong,
		Location:          location1,
		ResourceGroup:     rg,
		KubernetesVersion: k8sVersion,
		Profiles: []containerservice.AgentPoolProfile{
			{
				Name:   &agentName,
				Count:  &agentCountInt32,
				VMSize: vmSize1,
			},
		},
	}

	createRequestWrongName = &cluster.CreateClusterRequest{
		Name:              nameBad1,
		Location:          location1,
		ResourceGroup:     rg,
		KubernetesVersion: k8sVersion,
		Profiles: []containerservice.AgentPoolProfile{
			{
				Name:   &agentName,
				Count:  &agentCountInt32,
				VMSize: vmSize1,
			},
		},
	}
)

var (
	createResponse = &azure.ResponseWithValue{
		StatusCode: http.StatusOK,
		Value: azure.Value{
			Id:       id,
			Location: location1,
			Name:     name,
			Properties: azure.Properties{
				ProvisioningState: provisioningState,
				AgentPoolProfiles: nil,
				Fqdn:              fqdn,
			},
		},
	}

	pollingResponse = &azure.ResponseWithValue{
		StatusCode: http.StatusCreated,
		Value: azure.Value{
			Id:       id,
			Location: location1,
			Name:     name,
			Properties: azure.Properties{
				ProvisioningState: provisioningState,
				AgentPoolProfiles: nil,
				Fqdn:              fqdn,
			},
		},
	}
)
