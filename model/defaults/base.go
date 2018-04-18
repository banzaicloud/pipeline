package defaults

import (
	"fmt"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

// TODO se who will win
var logger *logrus.Logger
var log *logrus.Entry

// cluster profile table names
const (
	DefaultAmazonProfileTablaName         = "amazon_default_profile"
	DefaultAzureProfileTablaName          = "azure_default_profile"
	DefaultAzureNodePoolProfileTablaName  = "azure_nodepool_default_profile"
	DefaultGoogleProfileTablaName         = "google_default_profile"
	DefaultGoogleNodePoolProfileTablaName = "google_nodepool_default_profile"
)

const (
	DefaultNodeName = "pool1"
)

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"action": constants.TagGetClusterProfile})
}

// SetDefaultValues saves the default cluster profile into the database if not exists yet
func SetDefaultValues() error {

	log.Info("Save default cluster profiles")

	defaults := GetDefaultProfiles()
	for _, d := range defaults {
		if !d.IsDefinedBefore() {
			// the table not contains the default profile
			log.Infof("%s default table NOT contains the default values. Fill it...", d.GetType())
			if err := d.SaveInstance(); err != nil {
				// save failed
				return fmt.Errorf("Could not save default values[%s]: %s", d.GetType(), err.Error())
			}
		} else {
			// it's already exists
			log.Infof("%s default table already contains the default values", d.GetType())
		}
	}

	return nil
}

// ClusterProfile describes a cluster profile
type ClusterProfile interface {
	IsDefinedBefore() bool
	SaveInstance() error
	GetType() string
	GetProfile() *components.ClusterProfileResponse
	UpdateProfile(*components.ClusterProfileRequest, bool) error
	DeleteProfile() error
}

// DefaultModel describes the common variables all types of clouds
type DefaultModel struct {
	Name      string `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// save saves the given data into database
func save(i interface{}) error {
	database := model.GetDB()
	return database.Save(i).Error
}

// loadFirst find first record that match given conditions, order by primary key
func loadFirst(output interface{}) error {
	return model.GetDB().First(output).Error
}

// GetDefaultProfiles create all types of clouds with default profile name
func GetDefaultProfiles() []ClusterProfile {
	var defaults []ClusterProfile
	defaults = append(defaults,
		&AWSProfile{DefaultModel: DefaultModel{Name: GetDefaultProfileName()}},
		&AKSProfile{
			DefaultModel: DefaultModel{Name: GetDefaultProfileName()},
			NodePools: []*AKSNodePoolProfile{{
				NodeName: DefaultNodeName,
			}},
		},
		&GKEProfile{
			DefaultModel: DefaultModel{Name: GetDefaultProfileName()},
			NodePools: []*GKENodePoolProfile{{
				NodeName: DefaultNodeName,
			}},
		})
	return defaults
}

// GetAllProfiles loads all saved cluster profile from database by given cloud type
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
		return nil, constants.ErrorNotSupportedCloudType
	}

	return defaults, nil

}

// GetProfile finds cluster profile from database by given name and cloud type
func GetProfile(cloudType string, name string) (ClusterProfile, error) {
	db := model.GetDB()

	switch cloudType {
	case constants.Amazon:
		var awsProfile AWSProfile
		if err := db.Where(GKEProfile{DefaultModel: DefaultModel{Name: name}}).First(&awsProfile).Error; err != nil {
			return nil, err
		}
		return &awsProfile, nil

	case constants.Azure:
		var aksProfile AKSProfile
		if err := db.Where(GKEProfile{DefaultModel: DefaultModel{Name: name}}).First(&aksProfile).Error; err != nil {
			return nil, err
		}
		return &aksProfile, nil

	case constants.Google:
		var gkeProfile GKEProfile
		if err := db.Where(GKEProfile{DefaultModel: DefaultModel{Name: name}}).First(&gkeProfile).Error; err != nil {
			return nil, err
		}
		return &gkeProfile, nil

	default:
		return nil, constants.ErrorNotSupportedCloudType
	}

}

// GetDefaultProfileName reads the default profile name env var
func GetDefaultProfileName() string {
	return viper.GetString("cloud.defaultProfileName")
}
