package oracle

import (
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/objectstore"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

const Provider = "oracle"

// NewObjectStore returns a new object store instance
func NewObjectStore(
	org *auth.Organization,
	secret *secret.SecretItemResponse,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *objectstore.ObjectStore {
	return objectstore.NewObjectStore(org, secret, db, logger)
}
