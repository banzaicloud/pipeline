package api

// CreateResourceGroupRequest describes the resource group create request
type CreateResourceGroupRequest struct {
	Name     string `json:"name" binding:"required"`
	Location string `json:"location" binding:"required"`
	SecretId string `json:"secretId" binding:"required"`
}

// CreateResourceGroupResponse describes the resource group create response
type CreateResourceGroupResponse struct {
	Name string `json:"name" binding:"required"`
}
