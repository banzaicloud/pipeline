package defaults

import (
	"github.com/banzaicloud/banzai-types/constants"
	"time"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/banzai-types/components"
)

// TODO se who will win
var logger *logrus.Logger
var log *logrus.Entry

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"action": constants.TagGetDefaults})
}

func SetDefaultValues() {

	defaults := GetDefaults()
	for _, d := range defaults {
		if !d.IsDefinedBefore() {
			log.Infof("%s default table NOT contains the default values. Fill it...", d.GetType())
			if err := d.SaveDefaultInstance(); err != nil {
				log.Errorf("Could not save default values[%s]: %s", d.GetType(), err.Error())
			}
		} else {
			log.Infof("%s default table already contains the default values", d.GetType())
		}
	}
}

type Default interface {
	IsDefinedBefore() bool
	SaveDefaultInstance() error
	GetType() string
	GetDefaultCreateClusterRequest() *components.CreateClusterRequest
}

type DefaultModel struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func save(i interface{}) error {
	database := model.GetDB()
	if err := database.Save(i).Error; err != nil {
		return err
	}
	return nil
}

func loadFirst(output interface{}) {
	model.GetDB().First(output)
}

func GetDefaults() []Default {
	var defaults []Default
	defaults = append(defaults, &DefaultCreateRequestAWS{}, &DefaultCreateRequestAKS{}, &DefaultCreateRequestGKE{})
	return defaults
}
