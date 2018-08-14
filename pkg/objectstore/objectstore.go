package objectstore

// ObjectStore is the interface that cloud specific object store implementation
// must implement
type ObjectStore interface {
	CreateBucket(string)
	ListBuckets() ([]*BucketInfo, error)
	DeleteBucket(string) error
	CheckBucket(string) error
}

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
