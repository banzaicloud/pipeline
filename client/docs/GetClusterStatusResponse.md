# GetClusterStatusResponse

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Status** | **string** |  | [optional] 
**StatusMessage** | **string** |  | [optional] 
**Name** | **string** |  | [optional] 
**Cloud** | **string** |  | [optional] 
**Distribution** | **string** |  | [optional] 
**Version** | **string** |  | [optional] 
**Spot** | **bool** |  | [optional] 
**Location** | **string** |  | [optional] 
**Id** | **int32** |  | [optional] 
**Logging** | **bool** |  | [optional] 
**Monitoring** | **bool** |  | [optional] 
**Servicemesh** | **bool** |  | [optional] 
**Securityscan** | **bool** |  | [optional] 
**CreatedAt** | **string** |  | [optional] 
**StartedAt** | **string** |  | [optional] 
**CreatorName** | **string** |  | [optional] 
**CreatorId** | **int32** |  | [optional] 
**Region** | **string** |  | [optional] 
**TtlMinutes** | **int32** | The lifespan of the cluster expressed in minutes after which it is automatically deleted. Zero value means the cluster is never automatically deleted. | [optional] 
**NodePools** | [**map[string]NodePoolStatus**](NodePoolStatus.md) |  | [optional] 
**TotalSummary** | [**ResourceSummary**](ResourceSummary.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


