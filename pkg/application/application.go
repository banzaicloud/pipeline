package application

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/catalog"
)

//CreateRequest describes an application creation request
type CreateRequest struct {
	Name        string                           `json:"name" binding:"required"`
	CatalogName string                           `json:"catalogName" binding:"required"`
	Cluster     *components.CreateClusterRequest `json:"cluster"`
	ClusterId   uint                             `json:"clusterId"`
	Options     []catalog.ApplicationOptions     `json:"options"`
}

// CreateResponse API response for CreateApplication
type CreateResponse struct {
	Name      string `json:"name" binding:"required"`
	Id        uint   `json:"id" binding:"required"`
	ClusterId uint   `json:"clusterId" binding:"required"`
}

// DetailsResponse for API
type DetailsResponse struct {
	Name        string   `json:"name"`
	ClusterName string   `json:"clusterName"`
	ClusterId   int      `json:"clusterId"`
	Status      string   `json:"status"`
	Icon        string   `json:"icon"`
	Deployments []string `json:"deployments"`
	Error       string   `json:"error"`
	//Spotguide
}

// ListResponse for API
type ListResponse struct {
	Id            uint   `json:"id"`
	Name          string `json:"name"`
	ClusterName   string `json:"clusterName"`
	ClusterId     uint   `json:"clusterId"`
	Status        string `json:"status"`
	CatalogName   string `json:"catalogName"`
	Icon          string `json:"icon"`
	StatusMessage string `json:"statusMessage"`
}
