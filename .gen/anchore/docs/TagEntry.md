# TagEntry

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Pullstring** | **string** | The pullable string for the tag. E.g. \&quot;docker.io/library/node:latest\&quot; | [optional] 
**Registry** | **string** | The registry hostname:port section of the pull string | [optional] 
**Repository** | **string** | The repository section of the pull string | [optional] 
**Tag** | **string** | The tag-only section of the pull string | [optional] 
**DetectedAt** | [**time.Time**](time.Time.md) | The timestamp at which the Anchore Engine detected this tag was mapped to the image digest. Does not necessarily indicate when the tag was actually pushed to the registry. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


