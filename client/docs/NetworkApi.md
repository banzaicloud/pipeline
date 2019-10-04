# \NetworkApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ListRouteTables**](NetworkApi.md#ListRouteTables) | **Get** /api/v1/orgs/{orgId}/networks/{networkId}/routeTables | List VPC route tables
[**ListVPCNetworks**](NetworkApi.md#ListVPCNetworks) | **Get** /api/v1/orgs/{orgId}/networks | List VPC networks
[**ListVPCSubnets**](NetworkApi.md#ListVPCSubnets) | **Get** /api/v1/orgs/{orgId}/networks/{networkId}/subnets | List VPC subnetworks



## ListRouteTables

> []RouteTableInfo ListRouteTables(ctx, orgId, networkId, secretId, cloudType, optional)
List VPC route tables

List route tables of the given VPC network

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**networkId** | **string**| VPC network identification | 
**secretId** | **string**| Secret identification | 
**cloudType** | **string**| Identifies the cloud provider | 
 **optional** | ***ListRouteTablesOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a ListRouteTablesOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------




 **region** | **optional.String**| Identifies the region of the VPC network (required when cloudType !&#x3D; azure) | 
 **resourceGroup** | **optional.String**| Identifies the resource group of the Azure virtual network (required when cloudType &#x3D;&#x3D; azure) | 

### Return type

[**[]RouteTableInfo**](RouteTableInfo.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListVPCNetworks

> []VpcNetworkInfo ListVPCNetworks(ctx, orgId, secretId, cloudType, optional)
List VPC networks

List VPC networks accessible by the organization.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**secretId** | **string**| Secret identification | 
**cloudType** | **string**| Identifies the cloud provider | 
 **optional** | ***ListVPCNetworksOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a ListVPCNetworksOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **region** | **optional.String**| Identifies the region of the VPC network (required when cloudType !&#x3D; azure) | 
 **resourceGroup** | **optional.String**| Identifies the resource group of the Azure virtual network (required when cloudType &#x3D;&#x3D; azure) | 

### Return type

[**[]VpcNetworkInfo**](VPCNetworkInfo.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListVPCSubnets

> []SubnetInfo ListVPCSubnets(ctx, orgId, networkId, secretId, cloudType, optional)
List VPC subnetworks

List subnetworks of the given VPC network

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**networkId** | **string**| VPC network identification | 
**secretId** | **string**| Secret identification | 
**cloudType** | **string**| Identifies the cloud provider | 
 **optional** | ***ListVPCSubnetsOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a ListVPCSubnetsOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------




 **region** | **optional.String**| Identifies the region of the VPC network (required when cloudType !&#x3D; azure) | 
 **resourceGroup** | **optional.String**| Identifies the resource group of the Azure virtual network (required when cloudType &#x3D;&#x3D; azure) | 

### Return type

[**[]SubnetInfo**](SubnetInfo.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

