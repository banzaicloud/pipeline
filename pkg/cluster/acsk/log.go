package acsk

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/sirupsen/logrus"
)

var log logrus.FieldLogger

func init() {
	log = config.Logger()
}
