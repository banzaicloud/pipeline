# ImageAnalysisRequest

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Dockerfile** | **string** | Base64 encoded content of the dockerfile for the image, if available. Deprecated in favor of the &#39;source&#39; field. | [optional] 
**Digest** | **string** | A digest string for an image, maybe a pull string or just a digest. e.g. nginx@sha256:123 or sha256:abc123. If a pull string, it must have same regisry/repo as the tag field. Deprecated in favor of the &#39;source&#39; field | [optional] 
**Tag** | **string** | Full pullable tag reference for image. e.g. docker.io/nginx:latest. Deprecated in favor of the &#39;source&#39; field | [optional] 
**CreatedAt** | [**time.Time**](time.Time.md) | Optional override of the image creation time, only honored when both tag and digest are also supplied  e.g. 2018-10-17T18:14:00Z. Deprecated in favor of the &#39;source&#39; field | [optional] 
**ImageType** | **string** | Optional. The type of image this is adding, defaults to \&quot;docker\&quot;. This can be ommitted until multiple image types are supported. | [optional] 
**Annotations** | [**map[string]interface{}**](.md) | Annotations to be associated with the added image in key/value form | [optional] 
**Source** | [**ImageSource**](ImageSource.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


