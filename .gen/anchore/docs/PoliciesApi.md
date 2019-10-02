# \PoliciesApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddPolicy**](PoliciesApi.md#AddPolicy) | **Post** /policies | Add a new policy
[**DeletePolicy**](PoliciesApi.md#DeletePolicy) | **Delete** /policies/{policyId} | Delete policy
[**GetPolicy**](PoliciesApi.md#GetPolicy) | **Get** /policies/{policyId} | Get specific policy
[**ListPolicies**](PoliciesApi.md#ListPolicies) | **Get** /policies | List policies
[**UpdatePolicy**](PoliciesApi.md#UpdatePolicy) | **Put** /policies/{policyId} | Update policy


# **AddPolicy**
> PolicyBundleRecord AddPolicy(ctx, bundle, optional)
Add a new policy

Adds a new policy bundle to the system

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **bundle** | [**PolicyBundle**](PolicyBundle.md)|  | 
 **optional** | ***AddPolicyOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a AddPolicyOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeletePolicy**
> DeletePolicy(ctx, policyId, optional)
Delete policy

Delete the specified policy

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **policyId** | **string**|  | 
 **optional** | ***DeletePolicyOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DeletePolicyOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPolicy**
> []PolicyBundleRecord GetPolicy(ctx, policyId, optional)
Get specific policy

Get the policy bundle content

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **policyId** | **string**|  | 
 **optional** | ***GetPolicyOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetPolicyOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **detail** | **optional.Bool**| Include policy bundle detail in the form of the full bundle content for each entry | 
 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**[]PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListPolicies**
> []PolicyBundleRecord ListPolicies(ctx, optional)
List policies

List all saved policy bundles

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ListPoliciesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ListPoliciesOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **detail** | **optional.Bool**| Include policy bundle detail in the form of the full bundle content for each entry | 
 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**[]PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdatePolicy**
> []PolicyBundleRecord UpdatePolicy(ctx, policyId, bundle, optional)
Update policy

Update/replace and existing policy

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **policyId** | **string**|  | 
  **bundle** | [**PolicyBundleRecord**](PolicyBundleRecord.md)|  | 
 **optional** | ***UpdatePolicyOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UpdatePolicyOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **active** | **optional.Bool**| Mark policy as active | 
 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**[]PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

