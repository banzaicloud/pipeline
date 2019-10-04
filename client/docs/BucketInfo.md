# BucketInfo

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | the name of the object storage / bucket | 
**Managed** | **bool** | true if the bucket has been created via pipeline | 
**Cloud** | **string** | cloud provider of the bucket | 
**Location** | **string** | location of the bucket | 
**Notes** | **string** | notes for the bucket | [optional] 
**Secret** | [**BucketInfoSecret**](BucketInfo_secret.md) |  | [optional] 
**Aks** | [**AzureBlobStorageProps**](AzureBlobStorageProps.md) |  | [optional] 
**Oracle** | [**OracleStorageProps**](OracleStorageProps.md) |  | [optional] 
**Status** | **string** | the status of the bucket | 
**StatusMessage** | **string** | the reason for the error status | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


