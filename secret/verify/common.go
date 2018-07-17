package verify

import (
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
)

// Verifier validates cloud credentials
type Verifier interface {
	VerifySecret() error
}

// NewVerifier create new instance which implements `Verifier` interface
func NewVerifier(cloudType string, values map[string]string) Verifier {
	switch cloudType {

	case pkgCluster.Alibaba:
		return CreateAlibabaSecret(values)
	case pkgCluster.Amazon:
		return CreateAWSSecret(values)
	case pkgCluster.Azure:
		return CreateAKSSecret(values)
	case pkgCluster.Google:
		return CreateGKESecret(values)
	case pkgCluster.Oracle:
		return oracle.CreateOCISecret(values)
	default:
		return nil
	}
}
