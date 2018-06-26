# \OrganizationsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateOrg**](OrganizationsApi.md#CreateOrg) | **Post** /api/v1/orgs | Create organization
[**GetOrg**](OrganizationsApi.md#GetOrg) | **Get** /api/v1/orgs/{orgId} | Get organization
[**ListOrgs**](OrganizationsApi.md#ListOrgs) | **Get** /api/v1/orgs | List organizations


# **CreateOrg**
> OrganizationCreateResponse CreateOrg(ctx, body)
Create organization

Creating organization

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**Body**](Body.md)|  | 

### Return type

[**OrganizationCreateResponse**](OrganizationCreateResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetOrg**
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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListOrgs**
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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

