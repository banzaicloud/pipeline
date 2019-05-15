# \ClustergroupsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdDelete**](ClustergroupsApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdDelete) | **Delete** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId} | Delete Cluster Group
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdGet**](ClustergroupsApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdGet) | **Get** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId} | Get Cluster Group
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdPut**](ClustergroupsApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdPut) | **Put** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId} | Update Cluster Group
[**ApiV1OrgsOrgidClustergroupsGet**](ClustergroupsApi.md#ApiV1OrgsOrgidClustergroupsGet) | **Get** /api/v1/orgs/{orgid}/clustergroups | List Cluster Groups of an Organization
[**ApiV1OrgsOrgidClustergroupsPost**](ClustergroupsApi.md#ApiV1OrgsOrgidClustergroupsPost) | **Post** /api/v1/orgs/{orgid}/clustergroups | Create Cluster Group


# **ApiV1OrgsOrgidClustergroupsClusterGroupIdDelete**
> string ApiV1OrgsOrgidClustergroupsClusterGroupIdDelete(ctx, orgid, clusterGroupId)
Delete Cluster Group

delete a cluster group, disable all enabled features, delete related deployments

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 
  **clusterGroupId** | **int32**| Cluster Group ID | 

### Return type

**string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiV1OrgsOrgidClustergroupsClusterGroupIdGet**
> ApiClusterGroup ApiV1OrgsOrgidClustergroupsClusterGroupIdGet(ctx, orgid, clusterGroupId)
Get Cluster Group

retrieve info about a cluster group, members, status of member clusters, features

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 
  **clusterGroupId** | **int32**| Cluster Group ID | 

### Return type

[**ApiClusterGroup**](api.ClusterGroup.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiV1OrgsOrgidClustergroupsClusterGroupIdPut**
> ApiUpdateResponse ApiV1OrgsOrgidClustergroupsClusterGroupIdPut(ctx, orgid, clusterGroupId, apiUpdateRequest)
Update Cluster Group

update name & member clusters for a cluster group

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 
  **clusterGroupId** | **int32**| Cluster Group ID | 
  **apiUpdateRequest** | [**ApiUpdateRequest**](ApiUpdateRequest.md)| Update Cluster Group Request | 

### Return type

[**ApiUpdateResponse**](api.UpdateResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiV1OrgsOrgidClustergroupsGet**
> []ApiClusterGroup ApiV1OrgsOrgidClustergroupsGet(ctx, orgid)
List Cluster Groups of an Organization

retrieve list of cluster groups of an organization

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 

### Return type

[**[]ApiClusterGroup**](api.ClusterGroup.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiV1OrgsOrgidClustergroupsPost**
> ApiCreateResponse ApiV1OrgsOrgidClustergroupsPost(ctx, orgid, apiCreateRequest)
Create Cluster Group

create a group of clusters, joining clusters together with a name

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 
  **apiCreateRequest** | [**ApiCreateRequest**](ApiCreateRequest.md)| Create Cluster Group Request | 

### Return type

[**ApiCreateResponse**](api.CreateResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

