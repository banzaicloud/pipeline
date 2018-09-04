package objectstore

// ObjectStore is the interface that cloud specific object store implementation must implement.
type ObjectStore interface {
	// CreateBucket creates a new bucket in the object store.
	CreateBucket(string) error

	// ListBuckets lists the current buckets in the object store.
	ListBuckets() ([]string, error)

	// CheckBucket checks the status of the given bucket.
	CheckBucket(string) error

	// DeleteBucket removes a bucket from the object store.
	DeleteBucket(string) error

	// GetRegion returns the region for a given bucket
	GetRegion(string) (string, error)
}
