# \ImportApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ImportImageArchive**](ImportApi.md#ImportImageArchive) | **Post** /import/images | Import an anchore image tar.gz archive file.



## ImportImageArchive

> []AnchoreImage ImportImageArchive(ctx, archiveFile)
Import an anchore image tar.gz archive file.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**archiveFile** | ***os.File*****os.File**| anchore image tar archive. | 

### Return type

[**[]AnchoreImage**](AnchoreImage.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: multipart/form-data
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

