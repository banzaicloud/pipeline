# \ClustergroupFeaturesApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNameDelete**](ClustergroupFeaturesApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNameDelete) | **Delete** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/features/{featureName} | Disable Feature of Cluster Group
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNameGet**](ClustergroupFeaturesApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNameGet) | **Get** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/features/{featureName} | Get Feature of Cluster Group
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNamePost**](ClustergroupFeaturesApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNamePost) | **Post** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/features/{featureName} | Enable Feature of Cluster Group
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNamePut**](ClustergroupFeaturesApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNamePut) | **Put** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/features/{featureName} | Update Feature of Cluster Group
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesGet**](ClustergroupFeaturesApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesGet) | **Get** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/features | Get All Features of Cluster Group


# **ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNameDelete**
> string ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNameDelete(ctx, orgid, clusterGroupId, featureName)
Disable Feature of Cluster Group

disable feature on all members of a cluster group

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 
  **clusterGroupId** | **int32**| Cluster Group ID | 
  **featureName** | **string**| Name of the feature | 

### Return type

**string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNameGet**
> ApiFeatureResponse ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNameGet(ctx, orgid, clusterGroupId, featureName)
Get Feature of Cluster Group

retrieve info about a cluster group feature and it's status on each member cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 
  **clusterGroupId** | **int32**| Cluster Group ID | 
  **featureName** | **string**| Name of the feature | 

### Return type

[**ApiFeatureResponse**](api.FeatureResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNamePost**
> string ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNamePost(ctx, orgid, clusterGroupId, featureName, body)
Enable Feature of Cluster Group

enable feature on all members of a cluster group

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 
  **clusterGroupId** | **int32**| Cluster Group ID | 
  **featureName** | **string**| Name of the feature | 
  **body** | **map[string]interface{}**| Feature properties | 

### Return type

**string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNamePut**
> string ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesFeatureNamePut(ctx, orgid, clusterGroupId, featureName, body)
Update Feature of Cluster Group

update properties of a feature

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 
  **clusterGroupId** | **int32**| Cluster Group ID | 
  **featureName** | **string**| Name of the feature | 
  **body** | **map[string]interface{}**| Feature properties | 

### Return type

**string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesGet**
> []ApiFeatureResponse ApiV1OrgsOrgidClustergroupsClusterGroupIdFeaturesGet(ctx, orgid, clusterGroupId, featureName)
Get All Features of Cluster Group

retrieve info about a cluster group feature and it's status on each member cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgid** | **int32**| Organization ID | 
  **clusterGroupId** | **int32**| Cluster Group ID | 
  **featureName** | **string**| Name of the future | 

### Return type

[**[]ApiFeatureResponse**](api.FeatureResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

