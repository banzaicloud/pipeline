# RegistryDigestSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Pullstring** | **string** | A digest-based pullstring (e.g. docker.io/nginx@sha256:123abc) | 
**Tag** | **string** | A docker pull string (e.g. docker.io/nginx:latest, or docker.io/nginx@sha256:abd) to retrieve the image | 
**CreationTimestampOverride** | [**time.Time**](time.Time.md) | Optional override of the image creation time to support proper tag history construction in cases of out-of-order analysis compared to registry history for the tag | 
**Dockerfile** | **string** | Base64 encoded content of the dockerfile used to build the image, if available. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


