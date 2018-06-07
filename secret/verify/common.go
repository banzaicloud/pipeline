package verify

import "github.com/banzaicloud/banzai-types/constants"

// Verifier validates cloud credentials
type Verifier interface {
	VerifySecret() error
}

// NewVerifier create new instance which implements `Verifier` interface
func NewVerifier(cloudType string, values map[string]string) Verifier {
	switch cloudType {

	case constants.Amazon:
		return CreateAWSSecret(values)
	case constants.Azure:
		return CreateAKSSecret(values)
	case constants.Google:
		return CreateGKESecret(values)
	default:
		return nil
	}
}
