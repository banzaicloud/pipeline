# \NetworkApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ListRouteTables**](NetworkApi.md#ListRouteTables) | **Get** /api/v1/orgs/{orgId}/networks/{networkId}/routeTables | List VPC route tables
[**ListVPCNetworks**](NetworkApi.md#ListVPCNetworks) | **Get** /api/v1/orgs/{orgId}/networks | List VPC networks
[**ListVPCSubnets**](NetworkApi.md#ListVPCSubnets) | **Get** /api/v1/orgs/{orgId}/networks/{networkId}/subnets | List VPC subnetworks


# **ListRouteTables**
> ListRouteTablesResponse ListRouteTables(ctx, orgId, networkId, secretId, cloudType, region)
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
  **region** | **string**| Identifies the region of the VPC network | 

### Return type

[**ListRouteTablesResponse**](ListRouteTablesResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListVPCNetworks**
> ListVpcNetworksResponse ListVPCNetworks(ctx, orgId, secretId, cloudType, region)
List VPC networks

List VPC networks accessible by the organization.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identification | 
  **cloudType** | **string**| Identifies the cloud provider | 
  **region** | **string**| Identifies the region of the VPC network | 

### Return type

[**ListVpcNetworksResponse**](ListVPCNetworksResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListVPCSubnets**
> ListVpcSubnetsResponse ListVPCSubnets(ctx, orgId, networkId, secretId, cloudType, region)
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
  **region** | **string**| Identifies the region of the VPC network | 

### Return type

[**ListVpcSubnetsResponse**](ListVPCSubnetsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

