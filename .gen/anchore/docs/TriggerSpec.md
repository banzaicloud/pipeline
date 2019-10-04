# TriggerSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name of the trigger as it would appear in a policy document | [optional] 
**Description** | **string** | Trigger description for what it tests and when it will fire during evaluation | [optional] 
**State** | **string** | State of the trigger | [optional] 
**SupercededBy** | Pointer to **string** | The name of another trigger that supercedes this on functionally if this is deprecated | [optional] 
**Parameters** | [**[]TriggerParamSpec**](TriggerParamSpec.md) | The list of parameters that are valid for this trigger | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


