# \PoliciesApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddPolicy**](PoliciesApi.md#AddPolicy) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/policies | Add a new policy
[**DeletePolicy**](PoliciesApi.md#DeletePolicy) | **Delete** /api/v1/orgs/{orgId}/clusters/{id}/policies/{policyId} | Delete policy
[**GetPolicy**](PoliciesApi.md#GetPolicy) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/policies/{policyId} | Get specific policy
[**ListPolicies**](PoliciesApi.md#ListPolicies) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/policies | List policies
[**UpdatePolicy**](PoliciesApi.md#UpdatePolicy) | **Put** /api/v1/orgs/{orgId}/clusters/{id}/policies/{policyId} | Update policy


# **AddPolicy**
> PolicyBundleRecord AddPolicy(ctx, id, orgId, policyBundle)
Add a new policy

Adds a new policy bundle to the system

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **int32**| Selected cluster identification (number) | 
  **orgId** | **int32**| Organization identification | 
  **policyBundle** | [**PolicyBundle**](PolicyBundle.md)|  | 

### Return type

[**PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeletePolicy**
> DeletePolicy(ctx, id, orgId, policyId)
Delete policy

Delete the specified policy

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **int32**| Selected cluster identification (number) | 
  **orgId** | **int32**| Organization identification | 
  **policyId** | **string**|  | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPolicy**
> []PolicyBundleRecord GetPolicy(ctx, id, orgId, policyId, optional)
Get specific policy

Get the policy bundle content

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **int32**| Selected cluster identification (number) | 
  **orgId** | **int32**| Organization identification | 
  **policyId** | **string**|  | 
 **optional** | ***GetPolicyOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetPolicyOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **detail** | **optional.Bool**| Include policy bundle detail in the form of the full bundle content for each entry | 

### Return type

[**[]PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListPolicies**
> []PolicyBundleRecord ListPolicies(ctx, id, orgId, optional)
List policies

List all saved policy bundles

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **int32**| Selected cluster identification (number) | 
  **orgId** | **int32**| Organization identification | 
 **optional** | ***ListPoliciesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ListPoliciesOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **detail** | **optional.Bool**| Include policy bundle detail in the form of the full bundle content for each entry | 

### Return type

[**[]PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdatePolicy**
> []PolicyBundleRecord UpdatePolicy(ctx, id, orgId, policyId, policyBundleRecord, optional)
Update policy

Update/replace and existing policy

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **int32**| Selected cluster identification (number) | 
  **orgId** | **int32**| Organization identification | 
  **policyId** | **string**|  | 
  **policyBundleRecord** | [**PolicyBundleRecord**](PolicyBundleRecord.md)|  | 
 **optional** | ***UpdatePolicyOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UpdatePolicyOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------




 **active** | **optional.Bool**| Mark policy as active | 

### Return type

[**[]PolicyBundleRecord**](PolicyBundleRecord.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

