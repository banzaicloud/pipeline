# \ArchivesApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ArchiveImageAnalysis**](ArchivesApi.md#ArchiveImageAnalysis) | **Post** /archives/images | 
[**CreateAnalysisArchiveRule**](ArchivesApi.md#CreateAnalysisArchiveRule) | **Post** /archives/rules | 
[**DeleteAnalysisArchiveRule**](ArchivesApi.md#DeleteAnalysisArchiveRule) | **Delete** /archives/rules/{ruleId} | 
[**DeleteArchivedAnalysis**](ArchivesApi.md#DeleteArchivedAnalysis) | **Delete** /archives/images/{imageDigest} | 
[**GetAnalysisArchiveRule**](ArchivesApi.md#GetAnalysisArchiveRule) | **Get** /archives/rules/{ruleId} | 
[**GetArchivedAnalysis**](ArchivesApi.md#GetArchivedAnalysis) | **Get** /archives/images/{imageDigest} | 
[**ListAnalysisArchive**](ArchivesApi.md#ListAnalysisArchive) | **Get** /archives/images | 
[**ListAnalysisArchiveRules**](ArchivesApi.md#ListAnalysisArchiveRules) | **Get** /archives/rules | 
[**ListArchives**](ArchivesApi.md#ListArchives) | **Get** /archives | 



## ArchiveImageAnalysis

> []AnalysisArchiveAddResult ArchiveImageAnalysis(ctx, imageReferences)


### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**imageReferences** | [**[]string**](string.md)|  | 

### Return type

[**[]AnalysisArchiveAddResult**](AnalysisArchiveAddResult.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateAnalysisArchiveRule

> AnalysisArchiveTransitionRule CreateAnalysisArchiveRule(ctx, rule)


### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**rule** | [**AnalysisArchiveTransitionRule**](AnalysisArchiveTransitionRule.md)|  | 

### Return type

[**AnalysisArchiveTransitionRule**](AnalysisArchiveTransitionRule.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteAnalysisArchiveRule

> AnalysisArchiveTransitionRule DeleteAnalysisArchiveRule(ctx, ruleId)


### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**ruleId** | **string**|  | 

### Return type

[**AnalysisArchiveTransitionRule**](AnalysisArchiveTransitionRule.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteArchivedAnalysis

> ArchivedAnalysis DeleteArchivedAnalysis(ctx, imageDigest, optional)


Performs a synchronous archive deletion

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**imageDigest** | **string**|  | 
 **optional** | ***DeleteArchivedAnalysisOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a DeleteArchivedAnalysisOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **force** | **optional.Bool**|  | 

### Return type

[**ArchivedAnalysis**](ArchivedAnalysis.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetAnalysisArchiveRule

> AnalysisArchiveTransitionRule GetAnalysisArchiveRule(ctx, ruleId)


### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**ruleId** | **string**|  | 

### Return type

[**AnalysisArchiveTransitionRule**](AnalysisArchiveTransitionRule.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetArchivedAnalysis

> ArchivedAnalysis GetArchivedAnalysis(ctx, imageDigest)


Returns the archive metadata record identifying the image and tags for the analysis in the archive.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**imageDigest** | **string**| The image digest to identify the image analysis | 

### Return type

[**ArchivedAnalysis**](ArchivedAnalysis.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListAnalysisArchive

> []ArchivedAnalysis ListAnalysisArchive(ctx, )


### Required Parameters

This endpoint does not need any parameter.

### Return type

[**[]ArchivedAnalysis**](ArchivedAnalysis.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListAnalysisArchiveRules

> []AnalysisArchiveTransitionRule ListAnalysisArchiveRules(ctx, optional)


### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ListAnalysisArchiveRulesOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a ListAnalysisArchiveRulesOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **systemGlobal** | **optional.Bool**| If true include system global rules (owned by admin) even for non-admin users. Defaults to true if not set. Can be set to false to exclude globals | 

### Return type

[**[]AnalysisArchiveTransitionRule**](AnalysisArchiveTransitionRule.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListArchives

> ArchiveSummary ListArchives(ctx, )


### Required Parameters

This endpoint does not need any parameter.

### Return type

[**ArchiveSummary**](ArchiveSummary.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

