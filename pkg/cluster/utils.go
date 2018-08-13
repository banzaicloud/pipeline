package cluster

import pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"

// ValidateCloudType validates if the passed cloudType is supported.
// If a not supported cloud type is passed in than returns ErrorNotSupportedCloudType otherwise nil
func ValidateCloudType(cloudType string) error {
	switch cloudType {
	case Alibaba:
		return nil
	case Amazon:
	case Google:
	case Azure:
	case Oracle:
		return nil
	default:
		return pkgErrors.ErrorNotSupportedCloudType
	}
	return nil
}
