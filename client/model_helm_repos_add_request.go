/*
 * Pipeline API
 *
 * Pipeline v0.3.0 swagger
 *
 * API version: latest
 * Contact: info@banzaicloud.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

type HelmReposAddRequest struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
	CaFile   string `json:"caFile,omitempty"`
}
