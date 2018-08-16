package objectstore

// Option sets configuration in the object store.
type Option interface {
	apply(*objectStore)
}

// WaitForCompletion makes the object store wait for the operations to actually complete.
// Eg. bucket is created and available or bucket is removed permanently.
type WaitForCompletion bool

func (o WaitForCompletion) apply(s *objectStore) {
	s.waitForCompletion = bool(o)
}
