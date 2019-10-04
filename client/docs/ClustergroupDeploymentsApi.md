# \ClustergroupDeploymentsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameDelete**](ClustergroupDeploymentsApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameDelete) | **Delete** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments/{deploymentName} | Delete Cluster Group Deployment
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameGet**](ClustergroupDeploymentsApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameGet) | **Get** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments/{deploymentName} | Get Cluster Group Deployment
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNamePut**](ClustergroupDeploymentsApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNamePut) | **Put** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments/{deploymentName} | Update Cluster Group Deployment
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameSyncPut**](ClustergroupDeploymentsApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameSyncPut) | **Put** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments/{deploymentName}/sync | Synchronize Cluster Group Deployment
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsGet**](ClustergroupDeploymentsApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsGet) | **Get** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments | Get All Deployments of a Cluster Group
[**ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsPost**](ClustergroupDeploymentsApi.md#ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsPost) | **Post** /api/v1/orgs/{orgid}/clustergroups/{clusterGroupId}/deployments | Create Cluster Group Deployment



## ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameDelete

> DeploymentTargetClusterStatus ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameDelete(ctx, orgid, clusterGroupId, deploymentName, optional)
Delete Cluster Group Deployment

deletes a cluster group deployment, also deleting deployments from member clusters

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgid** | **int32**| Organization ID | 
**clusterGroupId** | **int32**| Cluster Group ID | 
**deploymentName** | **string**| release name of a cluster group deployment | 
 **optional** | ***ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameDeleteOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameDeleteOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **force** | **optional.Bool**| if true cluster group deployment gets deleted even if some deployments can not be deleted from each target cluster | 

### Return type

[**DeploymentTargetClusterStatus**](deployment.TargetClusterStatus.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameGet

> DeploymentDeploymentInfo ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameGet(ctx, orgid, clusterGroupId, deploymentName)
Get Cluster Group Deployment

retrieve info about a cluster group deployment and it's status on each member cluster

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgid** | **int32**| Organization ID | 
**clusterGroupId** | **int32**| Cluster Group ID | 
**deploymentName** | **string**| release name of a cluster group deployment | 

### Return type

[**DeploymentDeploymentInfo**](deployment.DeploymentInfo.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNamePut

> DeploymentCreateUpdateDeploymentResponse ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNamePut(ctx, orgid, clusterGroupId, deploymentName, deploymentClusterGroupDeployment)
Update Cluster Group Deployment

updates a cluster group deployment, installs or upgrades deployment on each member cluster accordingly

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgid** | **int32**| Organization ID | 
**clusterGroupId** | **int32**| Cluster Group ID | 
**deploymentName** | **string**| release name of a cluster group deployment | 
**deploymentClusterGroupDeployment** | [**DeploymentClusterGroupDeployment**](DeploymentClusterGroupDeployment.md)| Deployment Update Request | 

### Return type

[**DeploymentCreateUpdateDeploymentResponse**](deployment.CreateUpdateDeploymentResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameSyncPut

> DeploymentTargetClusterStatus ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsDeploymentNameSyncPut(ctx, orgid, clusterGroupId, deploymentName)
Synchronize Cluster Group Deployment

install / upgrade deployment to target clusters where deployment is not found or has wrong

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgid** | **int32**| Organization ID | 
**clusterGroupId** | **int32**| Cluster Group ID | 
**deploymentName** | **string**| release name of a cluster group deployment | 

### Return type

[**DeploymentTargetClusterStatus**](deployment.TargetClusterStatus.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsGet

> []DeploymentListDeploymentResponse ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsGet(ctx, orgid, clusterGroupId)
Get All Deployments of a Cluster Group

retrieve all deployments from a cluster group

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgid** | **int32**| Organization ID | 
**clusterGroupId** | **int32**| Cluster Group ID | 

### Return type

[**[]DeploymentListDeploymentResponse**](deployment.ListDeploymentResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsPost

> DeploymentCreateUpdateDeploymentResponse ApiV1OrgsOrgidClustergroupsClusterGroupIdDeploymentsPost(ctx, orgid, clusterGroupId, deploymentClusterGroupDeployment)
Create Cluster Group Deployment

creates a new cluster group deployment, installs or upgrades deployment on each member cluster accordingly

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgid** | **int32**| Organization ID | 
**clusterGroupId** | **int32**| Cluster Group ID | 
**deploymentClusterGroupDeployment** | [**DeploymentClusterGroupDeployment**](DeploymentClusterGroupDeployment.md)| Deployment Create Request | 

### Return type

[**DeploymentCreateUpdateDeploymentResponse**](deployment.CreateUpdateDeploymentResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

