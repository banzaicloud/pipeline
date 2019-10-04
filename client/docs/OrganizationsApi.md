# \OrganizationsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetOrg**](OrganizationsApi.md#GetOrg) | **Get** /api/v1/orgs/{orgId} | Get organization
[**ListOrgs**](OrganizationsApi.md#ListOrgs) | **Get** /api/v1/orgs | List organizations
[**SyncOrgs**](OrganizationsApi.md#SyncOrgs) | **Put** /api/v1/orgs | Synchronize Github organizations



## GetOrg

> OrganizationListItemResponse GetOrg(ctx, orgId)
Get organization

Getting organization

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 

### Return type

[**OrganizationListItemResponse**](OrganizationListItemResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListOrgs

> []OrganizationListItemResponse ListOrgs(ctx, )
List organizations

Listing organizations

### Required Parameters

This endpoint does not need any parameter.

### Return type

[**[]OrganizationListItemResponse**](OrganizationListItemResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SyncOrgs

> SyncOrgs(ctx, )
Synchronize Github organizations

### Required Parameters

This endpoint does not need any parameter.

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

