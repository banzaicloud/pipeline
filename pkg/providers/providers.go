package providers

import (
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers/alibaba"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/pkg/providers/google"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle"
)

const (
	Alibaba = alibaba.Provider
	Amazon  = amazon.Provider
	Azure   = azure.Provider
	Google  = google.Provider
	Oracle  = oracle.Provider
)

// ValidateProvider validates if the passed cloud provider is supported.
// Unsupported cloud providers trigger an pkgErrors.ErrorNotSupportedCloudType error.
func ValidateProvider(provider string) error {
	switch provider {
	case Alibaba:
	case Amazon:
	case Google:
	case Azure:
	case Oracle:
	default:
		// TODO: create an error value in this package instead
		return pkgErrors.ErrorNotSupportedCloudType
	}

	return nil
}
