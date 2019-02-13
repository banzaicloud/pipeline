# CreateEksPropertiesEks

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Version** | **string** |  | [optional] 
**NodePools** | [**map[string]NodePoolsAmazon**](NodePoolsAmazon.md) |  | 
**Vpc** | [**EksVpc**](.md) |  | [optional] 
**RouteTableId** | **string** | Id of the RouteTable of the VPC to be used by subnets. This is used only when subnets are created into existing VPC. | [optional] 
**Subnets** | [**[]EksSubnet**](EKSSubnet.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


