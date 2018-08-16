package model

import (
	"fmt"
	"time"

	"github.com/banzaicloud/pipeline/model"
	"github.com/satori/go.uuid"
)

const unknownLocation = "unknown"

// TableName constants
const (
	clustersTableName = "clusters"
)

// ClusterModel describes the common cluster model.
type ClusterModel struct {
	ID  uint   `gorm:"primary_key"`
	UID string `gorm:"unique_index:idx_uid"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"unique_index:idx_unique_id" sql:"index"`
	CreatedBy uint

	Name           string `gorm:"unique_index:idx_unique_id"`
	Location       string
	Cloud          string
	Distribution   string
	OrganizationId uint `gorm:"unique_index:idx_unique_id"`
	SecretId       string
	ConfigSecretId string
	SshSecretId    string
	Status         string
	RbacEnabled    bool
	Monitoring     bool
	Logging        bool
	StatusMessage  string              `sql:"type:text;"`
	Applications   []model.Application `gorm:"foreignkey:ClusterID"`
}

// TableName changes the default table name.
func (ClusterModel) TableName() string {
	return clustersTableName
}

func (m *ClusterModel) BeforeCreate() (err error) {
	if m.UID == "" {
		m.UID = uuid.NewV4().String()
	}

	return
}

// AfterFind converts Location field(s) to unknown if they are empty.
func (m *ClusterModel) AfterFind() error {
	if len(m.Location) == 0 {
		m.Location = unknownLocation
	}

	return nil
}

// String method prints formatted cluster fields.
func (m ClusterModel) String() string {
	return fmt.Sprintf("Id: %d, Creation date: %s, Cloud: %s, Distribution: %s", m.ID, m.CreatedAt, m.Cloud, m.Distribution)
}
