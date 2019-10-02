# ArchivedAnalysis

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ImageDigest** | **string** | The image digest (digest of the manifest describing the image, per docker spec) | [optional] 
**ParentDigest** | **string** | The digest of a parent manifest (for manifest-list images) | [optional] 
**Annotations** | [**map[string]interface{}**](.md) | User provided annotations as key-value pairs | [optional] 
**Status** | **string** | The archival status | [optional] 
**ImageDetail** | [**[]TagEntry**](TagEntry.md) | List of tags associated with the image digest | [optional] 
**CreatedAt** | [**time.Time**](time.Time.md) |  | [optional] 
**LastUpdated** | [**time.Time**](time.Time.md) |  | [optional] 
**AnalyzedAt** | [**time.Time**](time.Time.md) |  | [optional] 
**ArchiveSizeBytes** | **int32** | The size, in bytes, of the analysis archive file | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


