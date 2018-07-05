# \ClustersApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ClusterPostHooks**](ClustersApi.md#ClusterPostHooks) | **Put** /api/v1/orgs/{orgId}/clusters/{id}/posthooks | Run posthook functions
[**CreateCluster**](ClustersApi.md#CreateCluster) | **Post** /api/v1/orgs/{orgId}/clusters | Create cluster
[**DeleteCluster**](ClustersApi.md#DeleteCluster) | **Delete** /api/v1/orgs/{orgId}/clusters/{id} | Delete cluster
[**GetAPIEndpoint**](ClustersApi.md#GetAPIEndpoint) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/apiendpoint | Get API endpoint
[**GetCluster**](ClustersApi.md#GetCluster) | **Get** /api/v1/orgs/{orgId}/clusters/{id} | Get cluster status
[**GetClusterConfig**](ClustersApi.md#GetClusterConfig) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/config | Get a cluster config
[**GetClusterStatus**](ClustersApi.md#GetClusterStatus) | **Head** /api/v1/orgs/{orgId}/clusters/{id} | Get cluster status
[**HelmInit**](ClustersApi.md#HelmInit) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/helminit | Initialize Helm
[**InstallSecrets**](ClustersApi.md#InstallSecrets) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/secrets | Install secrets into cluster
[**ListClusters**](ClustersApi.md#ListClusters) | **Get** /api/v1/orgs/{orgId}/clusters | List clusters
[**ListEndpoints**](ClustersApi.md#ListEndpoints) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/endpoints | List service public endpoints
[**ListNodes**](ClustersApi.md#ListNodes) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/nodes | List cluser nodes
[**UpdateCluster**](ClustersApi.md#UpdateCluster) | **Put** /api/v1/orgs/{orgId}/clusters/{id} | Update cluster
[**UpdateMonitoring**](ClustersApi.md#UpdateMonitoring) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/monitoring | Update monitoring


# **ClusterPostHooks**
> ClusterPostHooks(ctx, orgId, id, runPostHook)
Run posthook functions

Run posthook functions

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **runPostHook** | [**RunPostHook**](RunPostHook.md)|  | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreateCluster**
> CreateClusterResponse202 CreateCluster(ctx, orgId, createClusterRequest)
Create cluster

Create a new K8S cluster in the cloud

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **createClusterRequest** | [**CreateClusterRequest**](CreateClusterRequest.md)|  | 

### Return type

[**CreateClusterResponse202**](CreateClusterResponse_202.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteCluster**
> ClusterDelete200 DeleteCluster(ctx, orgId, id)
Delete cluster

Deleting a K8S cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**ClusterDelete200**](ClusterDelete_200.md)

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

# **GetClusterConfig**
> string GetClusterConfig(ctx, orgId, id)
Get a cluster config

Getting a K8S cluster config file

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

# **InstallSecrets**
> []InstallSecretsResponseItem InstallSecrets(ctx, orgId, id, installSecretsRequest)
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

[**[]InstallSecretsResponseItem**](InstallSecretsResponseItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
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

# **ListEndpoints**
> ListEndpointsResponse ListEndpoints(ctx, orgId, id)
List service public endpoints

List service public endpoints

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**ListEndpointsResponse**](ListEndpointsResponse.md)

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

# **UpdateCluster**
> UpdateCluster(ctx, orgId, id, updateClusterRequest)
Update cluster

Updating an existing K8S cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **updateClusterRequest** | [**UpdateClusterRequest**](UpdateClusterRequest.md)|  | 

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

