package azure

type secretClient interface {
	// Get returns the requested secret of the organization.
	Get(organizationID uint, secretID string) (map[string]string, error)
}
