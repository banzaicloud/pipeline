# CreateUpdateDeploymentRequest

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** |  | 
**Version** | **string** | Version of the deployment. If not specified, the latest version is used. | [optional] 
**Package** | **string** | The chart content packaged by &#x60;helm package&#x60;. If specified chart version is ignored. | [optional] 
**Namespace** | **string** |  | [optional] 
**ReleaseName** | **string** |  | [optional] 
**DryRun** | **bool** |  | [optional] 
**Wait** | **bool** | if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment are in a ready state before marking the release as successful | [optional] 
**Timeout** | **int64** | time in seconds to wait for any individual Kubernetes operation (default 300) | [optional] 
**OdPcts** | [**map[string]interface{}**](.md) | Map of resources in the template where replicas should have a minimum on-demand percentage. Format: &lt;kind.resourceName: min-percentage&gt; | [optional] 
**ReuseValues** | **bool** |  | [optional] 
**Values** | [**map[string]interface{}**](.md) | current values of the deployment | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


