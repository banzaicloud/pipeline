# \DomainApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetDomain**](DomainApi.md#GetDomain) | **Get** /api/v1/orgs/{orgId}/domain | 


# **GetDomain**
> GetDomainResponse GetDomain(ctx, orgId, optional)


Return the base domain

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
 **optional** | ***GetDomainOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetDomainOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **clusterid** | **optional.String**| Cluster ID bounded domain name | [default to false]

### Return type

[**GetDomainResponse**](GetDomainResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

