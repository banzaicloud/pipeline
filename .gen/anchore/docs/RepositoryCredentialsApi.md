# \RepositoryCredentialsApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddRepository**](RepositoryCredentialsApi.md#AddRepository) | **Post** /repositories | Add repository to watch



## AddRepository

> []Subscription AddRepository(ctx, repository, optional)
Add repository to watch

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**repository** | **string**| full repository to add e.g. docker.io/library/alpine | 
 **optional** | ***AddRepositoryOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a AddRepositoryOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **autosubscribe** | **optional.Bool**| flag to enable/disable auto tag_update activation when new images from a repo are added | 
 **lookuptag** | **optional.String**| use specified existing tag to perform repo scan (default is &#39;latest&#39;) | 
 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**[]Subscription**](Subscription.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

