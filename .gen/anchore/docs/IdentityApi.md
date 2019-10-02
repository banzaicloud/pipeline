# \IdentityApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddCredential**](IdentityApi.md#AddCredential) | **Post** /user/credentials | add/replace credential
[**GetCredentials**](IdentityApi.md#GetCredentials) | **Get** /user/credentials | Get current credential summary
[**GetUser**](IdentityApi.md#GetUser) | **Get** /user | List authenticated user info
[**GetUsersAccount**](IdentityApi.md#GetUsersAccount) | **Get** /account | List the account for the authenticated user


# **AddCredential**
> User AddCredential(ctx, credential)
add/replace credential

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **credential** | [**AccessCredential**](AccessCredential.md)|  | 

### Return type

[**User**](User.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetCredentials**
> []AccessCredential GetCredentials(ctx, )
Get current credential summary

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**[]AccessCredential**](AccessCredential.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetUser**
> User GetUser(ctx, )
List authenticated user info

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**User**](User.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetUsersAccount**
> Account GetUsersAccount(ctx, )
List the account for the authenticated user

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**Account**](Account.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

