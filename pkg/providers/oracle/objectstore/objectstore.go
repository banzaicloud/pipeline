package objectstore

// CreateObjectStoreBucketProperties describes Oracle Object Store Bucket creation request
type CreateObjectStoreBucketProperties struct {
	Location string `json:"location" binding:"required"`
}
