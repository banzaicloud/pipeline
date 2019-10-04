# \ArkSchedulesApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateARKSchedule**](ArkSchedulesApi.md#CreateARKSchedule) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/schedules | Create ARK schedule
[**DeleteARKSchedule**](ArkSchedulesApi.md#DeleteARKSchedule) | **Delete** /api/v1/orgs/{orgId}/clusters/{id}/schedules/{scheduleName} | Delete ARK schedule
[**GetARKSchedule**](ArkSchedulesApi.md#GetARKSchedule) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/schedules/{scheduleName} | Get ARK schedule
[**ListARKSchedules**](ArkSchedulesApi.md#ListARKSchedules) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/schedules | List ARK schedules



## CreateARKSchedule

> CreateScheduleResponse CreateARKSchedule(ctx, orgId, id, createScheduleRequest)
Create ARK schedule

Create ARK schedule

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**createScheduleRequest** | [**CreateScheduleRequest**](CreateScheduleRequest.md)|  | 

### Return type

[**CreateScheduleResponse**](CreateScheduleResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteARKSchedule

> DeleteScheduleResponse DeleteARKSchedule(ctx, orgId, id, scheduleName)
Delete ARK schedule

Delete ARK schedule

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**scheduleName** | **string**| Name of the schedule | 

### Return type

[**DeleteScheduleResponse**](DeleteScheduleResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetARKSchedule

> ScheduleResponse GetARKSchedule(ctx, orgId, id, scheduleName)
Get ARK schedule

Get ARK schedule

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**scheduleName** | **string**| Name of the schedule | 

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListARKSchedules

> []ScheduleResponse ListARKSchedules(ctx, orgId, id)
List ARK schedules

List ARK schedules

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 

### Return type

[**[]ScheduleResponse**](ScheduleResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

