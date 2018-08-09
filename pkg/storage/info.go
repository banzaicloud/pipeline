package storage

// BucketInfo desribes a storage bucket
type BucketInfo struct {
	Name     string                    `json:"name"  binding:"required"`
	Managed  bool                      `json:"managed" binding:"required"`
	Location string                    `json:"location,omitempty"`
	Azure    *BlobStoragePropsForAzure `json:"aks,omitempty"`
}

// BlobStoragePropsForAzure describes the Azure specific properties
type BlobStoragePropsForAzure struct {
	ResourceGroup  string `json:"resourceGroup" binding:"required"`
	StorageAccount string `json:"storageAccount" binding:"required"`
}
