# TriggerParamSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Parameter name as it appears in policy document | [optional] 
**Description** | **string** |  | [optional] 
**Example** | Pointer to **string** | An example value for the parameter (encoded as a string if the parameter is an object or list type) | [optional] 
**State** | **string** | State of the trigger parameter | [optional] 
**SupercededBy** | Pointer to **string** | The name of another trigger that supercedes this on functionally if this is deprecated | [optional] 
**Required** | **bool** | Is this a required parameter or optional | [optional] 
**Validator** | [**map[string]interface{}**](.md) | If present, a definition for validation of input. Typically a jsonschema object that can be used to validate an input against. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


