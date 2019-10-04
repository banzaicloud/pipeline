# AnalysisArchiveSummary

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**TotalImageCount** | **int32** | The number of unique images (digests) in the archive | [optional] 
**TotalTagCount** | **int32** | The number of tag records (registry/repo:tag pull strings) in the archive. This may include repeated tags but will always have a unique tag-&gt;digest mapping per record. | [optional] 
**TotalDataBytes** | **int32** | The total sum of all the bytes stored to the backing storage. Accounts for anchore-applied compression, but not compression by the underlying storage system. | [optional] 
**LastUpdated** | [**time.Time**](time.Time.md) | The timestamp of the most recent archived image | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


