package providers

import (
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/database"
	_objectstore "github.com/banzaicloud/pipeline/objectstore"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/pkg/providers/google"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/sirupsen/logrus"
)

// ObjectStoreContext describes all parameters necessary to create a cloud provider agnostic object store instance.
type ObjectStoreContext struct {
	Provider     string
	Secret       *secret.SecretItemResponse
	Organization *auth.Organization

	// Location (or region) is used by some cloud providers to determine where the bucket should be created.
	Location string

	// Azure specific parameters
	ResourceGroup  string
	StorageAccount string
}

// NewObjectStore creates an object store client for the given cloud provider.
// The created object is initialized with the passed in secret and organization.
func NewObjectStore(ctx *ObjectStoreContext, logger logrus.FieldLogger) (objectstore.ObjectStore, error) {
	db := database.GetDB()

	switch ctx.Provider {
	case Alibaba:
		return _objectstore.NewAlibabaObjectStore(ctx.Location, ctx.Secret, ctx.Organization), nil

	case Amazon:
		return amazon.NewObjectStore(ctx.Location, ctx.Secret, ctx.Organization, db, logger), nil

	case Azure:
		return azure.NewObjectStore(ctx.Location, ctx.ResourceGroup, ctx.StorageAccount, ctx.Secret, ctx.Organization, db, logger), nil

	case Google:
		return google.NewObjectStore(ctx.Organization, verify.CreateServiceAccount(ctx.Secret.Values), ctx.Location, db, logger), nil

	case Oracle:
		return oracle.NewObjectStore(ctx.Location, ctx.Secret, ctx.Organization, db, logger), nil

	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}
