package verify

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/sirupsen/logrus"
)

// Note: this should be FieldLogger instead.
// Debug mode should be split to a separate config.
var log *logrus.Logger

func init() {
	log = config.Logger()
}
