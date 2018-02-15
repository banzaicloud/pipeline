package defaults

import (
	"github.com/banzaicloud/banzai-types/constants"
	"time"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/spf13/viper"
)

// TODO se who will win
var logger *logrus.Logger
var log *logrus.Entry

const (
	DefaultAmazonProfileTablaName = "amazon_default_profile"
	DefaultAzureProfileTablaName  = "azure_default_profile"
	DefaultGoogleProfileTablaName = "google_default_profile"
)

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"action": constants.TagGetClusterProfile})
}

func SetDefaultValues() {

	defaults := GetDefaultProfiles()
	for _, d := range defaults {
		if !d.IsDefinedBefore() {
			log.Infof("%s default table NOT contains the default values. Fill it...", d.GetType())
			if err := d.SaveInstance(); err != nil {
				log.Errorf("Could not save default values[%s]: %s", d.GetType(), err.Error())
			}
		} else {
			log.Infof("%s default table already contains the default values", d.GetType())
		}
	}
}

type ClusterProfile interface {
	IsDefinedBefore() bool
	SaveInstance() error
	GetType() string
	GetProfile() *components.ClusterProfileResponse
	UpdateProfile(*components.ClusterProfileRequest, bool) error
	DeleteProfile() error
}

type DefaultModel struct {
	Name      string `gorm:"primary_key"`
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

func GetDefaultProfiles() []ClusterProfile {
	var defaults []ClusterProfile
	defaults = append(defaults,
		&AWSProfile{DefaultModel: DefaultModel{Name: GetDefaultProfileName()},},
		&AKSProfile{DefaultModel: DefaultModel{Name: GetDefaultProfileName()},},
		&GKEProfile{DefaultModel: DefaultModel{Name: GetDefaultProfileName()},})
	return defaults
}

func GetAllProfiles(cloudType string) ([]ClusterProfile, error) {

	var defaults []ClusterProfile
	db := model.GetDB()

	switch cloudType {

	case constants.Amazon:
		var awsProfiles []AWSProfile
		db.Find(&awsProfiles)
		for i := range awsProfiles {
			defaults = append(defaults, &awsProfiles[i])
		}

	case constants.Azure:
		var aksProfiles []AKSProfile
		db.Find(&aksProfiles)
		for i := range aksProfiles {
			defaults = append(defaults, &aksProfiles[i])
		}

	case constants.Google:
		var gkeProfiles []GKEProfile
		db.Find(&gkeProfiles)
		for i := range gkeProfiles {
			defaults = append(defaults, &gkeProfiles[i])
		}

	default:
		return nil, constants.NotSupportedCloudType
	}

	return defaults, nil

}

func GetProfile(cloudType string, name string) (ClusterProfile, error) {
	db := model.GetDB()

	switch cloudType {
	case constants.Amazon:
		var awsProfile AWSProfile
		if err := db.Where(GKEProfile{DefaultModel: DefaultModel{Name: name}}).First(&awsProfile).Error; err != nil {
			return nil, err
		} else {
			return &awsProfile, nil
		}

	case constants.Azure:
		var aksProfile AKSProfile
		if err := db.Where(GKEProfile{DefaultModel: DefaultModel{Name: name}}).First(&aksProfile).Error; err != nil {
			return nil, err
		} else {
			return &aksProfile, nil
		}

	case constants.Google:
		var gkeProfile GKEProfile
		if err := db.Where(GKEProfile{DefaultModel: DefaultModel{Name: name}}).First(&gkeProfile).Error; err != nil {
			return nil, err
		} else {
			return &gkeProfile, nil
		}
	default:
		return nil, constants.NotSupportedCloudType
	}

}

func GetDefaultProfileName() string {
	return viper.GetString("cloud.defaultProfileName")
}
