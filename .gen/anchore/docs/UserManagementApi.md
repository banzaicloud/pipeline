# \UserManagementApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateAccount**](UserManagementApi.md#CreateAccount) | **Post** /accounts | Create a new user. Only avaialble to admin user.
[**CreateUser**](UserManagementApi.md#CreateUser) | **Post** /accounts/{accountname}/users | Create a new user
[**CreateUserCredential**](UserManagementApi.md#CreateUserCredential) | **Post** /accounts/{accountname}/users/{username}/credentials | add/replace credential
[**DeleteAccount**](UserManagementApi.md#DeleteAccount) | **Delete** /accounts/{accountname} | Delete the specified account, only allowed if the account is in the disabled state. All users will be deleted along with the account and all resources will be garbage collected
[**DeleteUser**](UserManagementApi.md#DeleteUser) | **Delete** /accounts/{accountname}/users/{username} | Delete a specific user credential by username of the credential. Cannot be the credential used to authenticate the request.
[**DeleteUserCredential**](UserManagementApi.md#DeleteUserCredential) | **Delete** /accounts/{accountname}/users/{username}/credentials | Delete a credential by type
[**GetAccount**](UserManagementApi.md#GetAccount) | **Get** /accounts/{accountname} | Get info about an user. Only available to admin user. Uses the main user Id, not a username.
[**GetAccountUser**](UserManagementApi.md#GetAccountUser) | **Get** /accounts/{accountname}/users/{username} | Get a specific user in the specified account
[**ListAccounts**](UserManagementApi.md#ListAccounts) | **Get** /accounts | List user summaries. Only available to the system admin user.
[**ListUserCredentials**](UserManagementApi.md#ListUserCredentials) | **Get** /accounts/{accountname}/users/{username}/credentials | Get current credential summary
[**ListUsers**](UserManagementApi.md#ListUsers) | **Get** /accounts/{accountname}/users | List accounts for the user
[**UpdateAccountState**](UserManagementApi.md#UpdateAccountState) | **Put** /accounts/{accountname}/state | Update the state of an account to either enabled or disabled. For deletion use the DELETE route



## CreateAccount

> Account CreateAccount(ctx, account)
Create a new user. Only avaialble to admin user.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**account** | [**AccountCreationRequest**](AccountCreationRequest.md)|  | 

### Return type

[**Account**](Account.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateUser

> User CreateUser(ctx, accountname, user)
Create a new user

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 
**user** | [**UserCreationRequest**](UserCreationRequest.md)|  | 

### Return type

[**User**](User.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateUserCredential

> User CreateUserCredential(ctx, accountname, username, credential)
add/replace credential

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 
**username** | **string**|  | 
**credential** | [**AccessCredential**](AccessCredential.md)|  | 

### Return type

[**User**](User.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteAccount

> DeleteAccount(ctx, accountname)
Delete the specified account, only allowed if the account is in the disabled state. All users will be deleted along with the account and all resources will be garbage collected

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteUser

> DeleteUser(ctx, accountname, username)
Delete a specific user credential by username of the credential. Cannot be the credential used to authenticate the request.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 
**username** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteUserCredential

> DeleteUserCredential(ctx, accountname, username, credentialType)
Delete a credential by type

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 
**username** | **string**|  | 
**credentialType** | **string**|  | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetAccount

> Account GetAccount(ctx, accountname)
Get info about an user. Only available to admin user. Uses the main user Id, not a username.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 

### Return type

[**Account**](Account.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetAccountUser

> User GetAccountUser(ctx, accountname, username)
Get a specific user in the specified account

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 
**username** | **string**|  | 

### Return type

[**User**](User.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListAccounts

> []Account ListAccounts(ctx, optional)
List user summaries. Only available to the system admin user.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ListAccountsOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a ListAccountsOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **state** | **optional.String**| Filter accounts by state | 

### Return type

[**[]Account**](Account.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListUserCredentials

> []AccessCredential ListUserCredentials(ctx, accountname, username)
Get current credential summary

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 
**username** | **string**|  | 

### Return type

[**[]AccessCredential**](AccessCredential.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListUsers

> []User ListUsers(ctx, accountname)
List accounts for the user

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 

### Return type

[**[]User**](User.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateAccountState

> AccountStatus UpdateAccountState(ctx, accountname, desiredState)
Update the state of an account to either enabled or disabled. For deletion use the DELETE route

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountname** | **string**|  | 
**desiredState** | [**AccountStatus**](AccountStatus.md)|  | 

### Return type

[**AccountStatus**](AccountStatus.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

