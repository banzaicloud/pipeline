package secret

import "github.com/banzaicloud/pipeline/secret"

type store interface {
	// Get returns the requested secret of the organization.
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
}
