package defaults

import (
	"fmt"
	"time"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log *logrus.Logger

// cluster profile table names
const (
	DefaultAmazonProfileTablaName         = "amazon_default_profile"
	DefaultAmazonNodePoolProfileTablaName = "amazon_nodepool_default_profile"
	DefaultAmazonEksProfileTablaName      = "amazon_eks_default_profile"
	DefaultAzureProfileTablaName          = "azure_default_profile"
	DefaultAzureNodePoolProfileTablaName  = "azure_nodepool_default_profile"
	DefaultGoogleProfileTablaName         = "google_default_profile"
	DefaultGoogleNodePoolProfileTablaName = "google_nodepool_default_profile"
)

// default node name for all provider
const (
	DefaultNodeName = "pool1"
)

// Simple init for logging
func init() {
	log = config.Logger()
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
	GetProfile() *pkgCluster.ClusterProfileResponse
	UpdateProfile(*pkgCluster.ClusterProfileRequest, bool) error
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
	database := database.GetDB()
	return database.Save(i).Error
}

// loadFirst find first record that match given conditions, order by primary key
func loadFirst(output interface{}) error {
	return database.GetDB().First(output).Error
}

// GetDefaultProfiles create all types of clouds with default profile name
func GetDefaultProfiles() []ClusterProfile {
	var defaults []ClusterProfile
	defaults = append(defaults,
		&AWSProfile{DefaultModel: DefaultModel{Name: GetDefaultProfileName()},
			NodePools: []*AWSNodePoolProfile{{
				NodeName: DefaultNodeName,
			}},
		},
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
		},
		&oracle.Profile{
			Name: GetDefaultProfileName(),
			NodePools: []*oracle.ProfileNodePool{{
				Name: DefaultNodeName,
			}},
		})
	return defaults
}

// GetAllProfiles loads all saved cluster profile from database by given cloud type
func GetAllProfiles(cloudType string) ([]ClusterProfile, error) {

	var defaults []ClusterProfile
	db := database.GetDB()

	switch cloudType {

	case pkgCluster.Amazon:
		var awsProfiles []AWSProfile
		db.Find(&awsProfiles)
		for i := range awsProfiles {
			defaults = append(defaults, &awsProfiles[i])
		}

	case pkgCluster.Azure:
		var aksProfiles []AKSProfile
		db.Find(&aksProfiles)
		for i := range aksProfiles {
			defaults = append(defaults, &aksProfiles[i])
		}

	case pkgCluster.Google:
		var gkeProfiles []GKEProfile
		db.Find(&gkeProfiles)
		for i := range gkeProfiles {
			defaults = append(defaults, &gkeProfiles[i])
		}

	case pkgCluster.Oracle:
		okeProfiles := oracle.GetProfiles()
		for i := range okeProfiles {
			defaults = append(defaults, &okeProfiles[i])
		}

	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}

	return defaults, nil

}

// GetProfile finds cluster profile from database by given name and cloud type
func GetProfile(cloudType string, name string) (ClusterProfile, error) {
	db := database.GetDB()

	switch cloudType {
	case pkgCluster.Amazon:
		var awsProfile AWSProfile
		if err := db.Where(GKEProfile{DefaultModel: DefaultModel{Name: name}}).First(&awsProfile).Error; err != nil {
			return nil, err
		}
		return &awsProfile, nil

	case pkgCluster.Azure:
		var aksProfile AKSProfile
		if err := db.Where(GKEProfile{DefaultModel: DefaultModel{Name: name}}).First(&aksProfile).Error; err != nil {
			return nil, err
		}
		return &aksProfile, nil

	case pkgCluster.Google:
		var gkeProfile GKEProfile
		if err := db.Where(GKEProfile{DefaultModel: DefaultModel{Name: name}}).First(&gkeProfile).Error; err != nil {
			return nil, err
		}
		return &gkeProfile, nil

	case pkgCluster.Oracle:
		var okeProfile oracle.Profile
		okeProfile, err := oracle.GetProfileByName(name)
		return &okeProfile, err

	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}

}

// GetDefaultProfileName reads the default profile name env var
func GetDefaultProfileName() string {
	return viper.GetString("cloud.defaultProfileName")
}
