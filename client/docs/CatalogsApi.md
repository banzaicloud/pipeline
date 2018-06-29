# \CatalogsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetCatalogDetail**](CatalogsApi.md#GetCatalogDetail) | **Get** /api/v1/orgs/{orgId}/catalogs/{name} | Get catalog details
[**ListCatalogs**](CatalogsApi.md#ListCatalogs) | **Get** /api/v1/orgs/{orgId}/catalogs | List catalogs
[**UpdateCatalogs**](CatalogsApi.md#UpdateCatalogs) | **Put** /api/v1/orgs/{orgId}/catalogs/update | Update repository for catalog


# **GetCatalogDetail**
> CatalogDetailsResponse GetCatalogDetail(ctx, orgId, name)
Get catalog details

Get details about specific catalog

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **name** | **string**| Catalog name | 

### Return type

[**CatalogDetailsResponse**](CatalogDetailsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListCatalogs**
> ListCatalogResponse ListCatalogs(ctx, orgId)
List catalogs

List all available catalogs

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 

### Return type

[**ListCatalogResponse**](ListCatalogResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateCatalogs**
> UpdateCatalogs(ctx, orgId)
Update repository for catalog

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

