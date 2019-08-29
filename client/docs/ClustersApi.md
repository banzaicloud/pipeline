# \ClustersApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ClusterPostHooks**](ClustersApi.md#ClusterPostHooks) | **Put** /api/v1/orgs/{orgId}/clusters/{id}/posthooks | Run posthook functions
[**CreateCluster**](ClustersApi.md#CreateCluster) | **Post** /api/v1/orgs/{orgId}/clusters | Create cluster
[**DeleteCluster**](ClustersApi.md#DeleteCluster) | **Delete** /api/v1/orgs/{orgId}/clusters/{id} | Delete cluster
[**DeleteLeaderElection**](ClustersApi.md#DeleteLeaderElection) | **Delete** /api/v1/orgs/{orgId}/clusters/{id}/pke/leader | Delete cluster leader
[**DeleteNamespace**](ClustersApi.md#DeleteNamespace) | **Delete** /api/v1/orgs/{orgId}/clusters/{id}/namespaces/{namespace} | Delete namespace from a cluster
[**GetAPIEndpoint**](ClustersApi.md#GetAPIEndpoint) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/apiendpoint | Get API endpoint
[**GetCluster**](ClustersApi.md#GetCluster) | **Get** /api/v1/orgs/{orgId}/clusters/{id} | Get cluster status
[**GetClusterBootstrap**](ClustersApi.md#GetClusterBootstrap) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/bootstrap | Get cluster bootstrap info
[**GetClusterConfig**](ClustersApi.md#GetClusterConfig) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/config | Get a cluster config
[**GetClusterStatus**](ClustersApi.md#GetClusterStatus) | **Head** /api/v1/orgs/{orgId}/clusters/{id} | Get cluster status
[**GetLeaderElection**](ClustersApi.md#GetLeaderElection) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/pke/leader | Query cluster leader
[**GetOIDCClusterConfig**](ClustersApi.md#GetOIDCClusterConfig) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/oidcconfig | Get a cluster config with OIDC login
[**GetPKECommands**](ClustersApi.md#GetPKECommands) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/pke/commands | List bootstrap commands for namespaces
[**GetPodDetails**](ClustersApi.md#GetPodDetails) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/pods | Get pod details
[**GetReadyPKENode**](ClustersApi.md#GetReadyPKENode) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/pke/ready | Query reported node readiness information
[**HelmInit**](ClustersApi.md#HelmInit) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/helminit | Initialize Helm
[**InstallSecret**](ClustersApi.md#InstallSecret) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/secrets/{secretName} | Install a particular secret into a cluster with optional remapping
[**InstallSecrets**](ClustersApi.md#InstallSecrets) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/secrets | Install secrets into cluster
[**ListClusterEndpoints**](ClustersApi.md#ListClusterEndpoints) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/endpoints | List service public endpoints
[**ListClusterSecrets**](ClustersApi.md#ListClusterSecrets) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/secrets | List secrets which belongs to cluster
[**ListClusters**](ClustersApi.md#ListClusters) | **Get** /api/v1/orgs/{orgId}/clusters | List clusters
[**ListNodepoolLabels**](ClustersApi.md#ListNodepoolLabels) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/nodepools/labels | List cluser nodepool labels
[**ListNodes**](ClustersApi.md#ListNodes) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/nodes | List cluser nodes
[**MergeSecret**](ClustersApi.md#MergeSecret) | **Patch** /api/v1/orgs/{orgId}/clusters/{id}/secrets/{secretName} | Merge a particular secret with an existing one with optional remapping
[**PostLeaderElection**](ClustersApi.md#PostLeaderElection) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/pke/leader | Apply as new cluster leader
[**PostReadyPKENode**](ClustersApi.md#PostReadyPKENode) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/pke/ready | Report to Pipeline that a new node is ready (to be called by PKE installer)
[**UpdateCluster**](ClustersApi.md#UpdateCluster) | **Put** /api/v1/orgs/{orgId}/clusters/{id} | Update cluster
[**UpdateMonitoring**](ClustersApi.md#UpdateMonitoring) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/monitoring | Update monitoring


# **ClusterPostHooks**
> ClusterPostHooks(ctx, orgId, id, body)
Run posthook functions

Run posthook functions

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **body** | **map[string]interface{}**|  | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreateCluster**
> CreateClusterResponse202 CreateCluster(ctx, orgId, body)
Create cluster

Create a new K8S cluster in the cloud

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **body** | **map[string]interface{}**|  | 

### Return type

[**CreateClusterResponse202**](CreateClusterResponse_202.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteCluster**
> ClusterDelete200 DeleteCluster(ctx, orgId, id, optional)
Delete cluster

Deleting a K8S cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
 **optional** | ***DeleteClusterOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DeleteClusterOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **force** | **optional.Bool**| Ignore errors during deletion | [default to false]

### Return type

[**ClusterDelete200**](ClusterDelete_200.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteLeaderElection**
> DeleteLeaderElection(ctx, orgId, id)
Delete cluster leader

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteNamespace**
> DeleteNamespace(ctx, orgId, id, namespace)
Delete namespace from a cluster

Delete namespace from a cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **namespace** | **string**| Kubernetes namespace | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetAPIEndpoint**
> string GetAPIEndpoint(ctx, orgId, id)
Get API endpoint

Get API endpoint

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

**string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetCluster**
> GetClusterStatusResponse GetCluster(ctx, orgId, id)
Get cluster status

Getting cluster status

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**GetClusterStatusResponse**](GetClusterStatusResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetClusterBootstrap**
> GetClusterBootstrapResponse GetClusterBootstrap(ctx, orgId, id)
Get cluster bootstrap info

Get cluster bootstrap info

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**GetClusterBootstrapResponse**](GetClusterBootstrapResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetClusterConfig**
> ClusterConfig GetClusterConfig(ctx, orgId, id)
Get a cluster config

Getting a K8S cluster config file

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**ClusterConfig**](ClusterConfig.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetClusterStatus**
> GetClusterStatus(ctx, orgId, id)
Get cluster status

Getting the K8S cluster status

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetLeaderElection**
> GetLeaderElectionResponse GetLeaderElection(ctx, orgId, id)
Query cluster leader

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**GetLeaderElectionResponse**](GetLeaderElectionResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetOIDCClusterConfig**
> ClusterConfig GetOIDCClusterConfig(ctx, orgId, id)
Get a cluster config with OIDC login

Getting a K8S cluster config file with OIDC credentials

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**ClusterConfig**](ClusterConfig.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPKECommands**
> GetPkeCommandsResponse GetPKECommands(ctx, orgId, id)
List bootstrap commands for namespaces

Get the commands to use for bootstrapping nodes of a PKE cluster in each nodepool

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**GetPkeCommandsResponse**](GetPKECommandsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPodDetails**
> []PodItem GetPodDetails(ctx, orgId, id)
Get pod details

Getting pod details

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**[]PodItem**](PodItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetReadyPKENode**
> PkeClusterReadinessResponse GetReadyPKENode(ctx, orgId, id)
Query reported node readiness information

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**PkeClusterReadinessResponse**](PKEClusterReadinessResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **HelmInit**
> HelmInitResponse HelmInit(ctx, orgId, id, helmInitRequest)
Initialize Helm

Initialize helm in the cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **helmInitRequest** | [**HelmInitRequest**](HelmInitRequest.md)|  | 

### Return type

[**HelmInitResponse**](HelmInitResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **InstallSecret**
> InstallSecretResponse InstallSecret(ctx, orgId, id, secretName, installSecretRequest)
Install a particular secret into a cluster with optional remapping

Install a particular secret into a cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **secretName** | **string**| Secret name as it will be seen in the cluster | 
  **installSecretRequest** | [**InstallSecretRequest**](InstallSecretRequest.md)|  | 

### Return type

[**InstallSecretResponse**](InstallSecretResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **InstallSecrets**
> []InstallSecretResponse InstallSecrets(ctx, orgId, id, installSecretsRequest)
Install secrets into cluster

Install secrets into cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **installSecretsRequest** | [**InstallSecretsRequest**](InstallSecretsRequest.md)|  | 

### Return type

[**[]InstallSecretResponse**](InstallSecretResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListClusterEndpoints**
> ListEndpointsResponse ListClusterEndpoints(ctx, orgId, id, optional)
List service public endpoints

List service public endpoints

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
 **optional** | ***ListClusterEndpointsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ListClusterEndpointsOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **releaseName** | **optional.String**| Selected deployment release name | 

### Return type

[**ListEndpointsResponse**](ListEndpointsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListClusterSecrets**
> []SecretItem ListClusterSecrets(ctx, orgId, id, optional)
List secrets which belongs to cluster

List secrets which belongs to cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
 **optional** | ***ListClusterSecretsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ListClusterSecretsOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **releaseName** | **optional.String**| Selected deployment release name | 

### Return type

[**[]SecretItem**](SecretItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListClusters**
> []GetClusterStatusResponse ListClusters(ctx, orgId)
List clusters

Listing all the K8S clusters from the cloud

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 

### Return type

[**[]GetClusterStatusResponse**](GetClusterStatusResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListNodepoolLabels**
> map[string][]NodepoolLabels ListNodepoolLabels(ctx, orgId, id)
List cluser nodepool labels

List cluster nodepool labels

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**map[string][]NodepoolLabels**](array.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListNodes**
> ListNodesResponse ListNodes(ctx, orgId, id)
List cluser nodes

List cluser nodes

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**ListNodesResponse**](ListNodesResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **MergeSecret**
> InstallSecretResponse MergeSecret(ctx, orgId, id, secretName, installSecretRequest)
Merge a particular secret with an existing one with optional remapping

Merge a particular secret with an existing one

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **secretName** | **string**| Secret name as it will be seen in the cluster | 
  **installSecretRequest** | [**InstallSecretRequest**](InstallSecretRequest.md)|  | 

### Return type

[**InstallSecretResponse**](InstallSecretResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PostLeaderElection**
> PostLeaderElectionResponse PostLeaderElection(ctx, orgId, id, postLeaderElectionRequest)
Apply as new cluster leader

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **postLeaderElectionRequest** | [**PostLeaderElectionRequest**](PostLeaderElectionRequest.md)|  | 

### Return type

[**PostLeaderElectionResponse**](PostLeaderElectionResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PostReadyPKENode**
> map[string]interface{} PostReadyPKENode(ctx, orgId, id, postReadyPkeNodeRequest)
Report to Pipeline that a new node is ready (to be called by PKE installer)

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **postReadyPkeNodeRequest** | [**PostReadyPkeNodeRequest**](PostReadyPkeNodeRequest.md)|  | 

### Return type

[**map[string]interface{}**](map[string]interface{}.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateCluster**
> UpdateCluster(ctx, orgId, id, body)
Update cluster

Updating an existing K8S cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **body** | **map[string]interface{}**|  | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateMonitoring**
> string UpdateMonitoring(ctx, orgId, id)
Update monitoring

Update monitoring

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

**string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

