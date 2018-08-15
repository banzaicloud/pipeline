package objectstore

type errBucketAlreadyExists struct{}

func (errBucketAlreadyExists) Error() string       { return "bucket already exists" }
func (errBucketAlreadyExists) AlreadyExists() bool { return true }

type errBucketNotFound struct{}

func (errBucketNotFound) Error() string  { return "bucket not found" }
func (errBucketNotFound) NotFound() bool { return true }
