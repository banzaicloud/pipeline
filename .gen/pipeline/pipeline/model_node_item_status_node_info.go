/*
 * Pipeline API
 *
 * Pipeline is a feature rich application platform, built for containers on top of Kubernetes to automate the DevOps experience, continuous application development and the lifecycle of deployments.
 *
 * API version: latest
 * Contact: info@banzaicloud.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package pipeline

type NodeItemStatusNodeInfo struct {
	MachineID string `json:"machineID,omitempty"`

	SystemUUID string `json:"systemUUID,omitempty"`

	BootID string `json:"bootID,omitempty"`

	KernelVersion string `json:"kernelVersion,omitempty"`

	OsImage string `json:"osImage,omitempty"`

	ContainerRuntimeVersion string `json:"containerRuntimeVersion,omitempty"`

	KubeletVersion string `json:"kubeletVersion,omitempty"`

	KubeProxyVersion string `json:"kubeProxyVersion,omitempty"`

	OperatingSystem string `json:"operatingSystem,omitempty"`

	Architecture string `json:"architecture,omitempty"`
}
