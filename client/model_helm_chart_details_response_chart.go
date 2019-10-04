/*
 * Pipeline API
 *
 * Pipeline is a feature rich application platform, built for containers on top of Kubernetes to automate the DevOps experience, continuous application development and the lifecycle of deployments.
 *
 * API version: latest
 * Contact: info@banzaicloud.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

// HelmChartDetailsResponseChart struct for HelmChartDetailsResponseChart
type HelmChartDetailsResponseChart struct {
	Name        string                                     `json:"name,omitempty"`
	Home        string                                     `json:"home,omitempty"`
	Sources     []string                                   `json:"sources,omitempty"`
	Version     string                                     `json:"version,omitempty"`
	Description string                                     `json:"description,omitempty"`
	Keywords    []string                                   `json:"keywords,omitempty"`
	Maintainers []HelmChartDetailsResponseChartMaintainers `json:"maintainers,omitempty"`
	Engine      string                                     `json:"engine,omitempty"`
	Icon        string                                     `json:"icon,omitempty"`
	AppVersion  string                                     `json:"appVersion,omitempty"`
	ApiVersion  string                                     `json:"apiVersion,omitempty"`
	Deprecated  bool                                       `json:"deprecated,omitempty"`
	Urls        []string                                   `json:"urls,omitempty"`
	Created     string                                     `json:"created,omitempty"`
	Digest      string                                     `json:"digest,omitempty"`
}
