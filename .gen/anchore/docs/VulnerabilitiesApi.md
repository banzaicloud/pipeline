# \VulnerabilitiesApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**QueryImagesByVulnerability**](VulnerabilitiesApi.md#QueryImagesByVulnerability) | **Get** /query/images/by_vulnerability | List images vulnerable to the specific vulnerability ID.
[**QueryVulnerabilities**](VulnerabilitiesApi.md#QueryVulnerabilities) | **Get** /query/vulnerabilities | Listing information about given vulnerability


# **QueryImagesByVulnerability**
> PaginatedVulnerableImageList QueryImagesByVulnerability(ctx, vulnerabilityId, optional)
List images vulnerable to the specific vulnerability ID.

Returns a listing of images and their respective packages vulnerable to the given vulnerability ID

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **vulnerabilityId** | **string**| The ID of the vulnerability to search for within all images stored in anchore-engine (e.g. CVE-1999-0001) | 
 **optional** | ***QueryImagesByVulnerabilityOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a QueryImagesByVulnerabilityOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **namespace** | **optional.String**| Filter results to images within the given vulnerability namespace (e.g. debian:8, ubuntu:14.04) | 
 **affectedPackage** | **optional.String**| Filter results to images with vulnable packages with the given package name (e.g. libssl) | 
 **severity** | **optional.String**| Filter results to vulnerable package/vulnerability with the given severity | 
 **vendorOnly** | **optional.Bool**| Filter results to include only vulnerabilities that are not marked as invalid by upstream OS vendor data | [default to true]
 **page** | **optional.Int32**| The page of results to fetch. Pages start at 1 | 
 **limit** | **optional.Int32**| Limit the number of records for the requested page. If omitted or set to 0, return all results in a single page | 
 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**PaginatedVulnerableImageList**](PaginatedVulnerableImageList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **QueryVulnerabilities**
> PaginatedVulnerabilityList QueryVulnerabilities(ctx, id, optional)
Listing information about given vulnerability

List (w/filters) vulnerability records known by the system, with affected packages information if present

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **string**| The ID of the vulnerability (e.g. CVE-1999-0001) | 
 **optional** | ***QueryVulnerabilitiesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a QueryVulnerabilitiesOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **affectedPackage** | **optional.String**| Filter results by specified package name (e.g. sed) | 
 **affectedPackageVersion** | **optional.String**| Filter results by specified package version (e.g. 4.4-1) | 
 **page** | **optional.String**| The page of results to fetch. Pages start at 1 | [default to 1]
 **limit** | **optional.Int32**| Limit the number of records for the requested page. If omitted or set to 0, return all results in a single page | 

### Return type

[**PaginatedVulnerabilityList**](PaginatedVulnerabilityList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

