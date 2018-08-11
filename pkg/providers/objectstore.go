package providers

import (
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/database"
	_objectstore "github.com/banzaicloud/pipeline/objectstore"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers/alibaba"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/pkg/providers/google"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/sirupsen/logrus"
)

// NewObjectStore creates an object store client for the given cloud provider.
// The created object is initialized with the passed in secret and organization.
func NewObjectStore(provider string, s *secret.SecretItemResponse, organization *auth.Organization, logger logrus.FieldLogger) (objectstore.ObjectStore, error) {
	switch provider {
	case alibaba.Provider:
		return _objectstore.NewAlibabaObjectStore(s, organization), nil

	case amazon.Provider:
		return amazon.NewObjectStore(organization, s, database.GetDB(), logger), nil

	case google.Provider:
		return google.NewObjectStore(organization, verify.CreateServiceAccount(s.Values), database.GetDB(), logger), nil

	case azure.Provider:
		return azure.NewObjectStore(organization, s, database.GetDB(), logger), nil

	case oracle.Provider:
		return _objectstore.NewOracleObjectStore(s, organization), nil

	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}
