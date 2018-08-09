package defaults

type AmazonNodePoolProfileBaseFields struct {
	ID           uint   `gorm:"primary_key"`
	InstanceType string `gorm:"default:'m4.xlarge'"`
	Name         string `gorm:"unique_index:idx_model_name"`
	NodeName     string `gorm:"unique_index:idx_model_name"`
	SpotPrice    string `gorm:"default:'0.2'"`
	Autoscaling  bool   `gorm:"default:false"`
	MinCount     int    `gorm:"default:1"`
	MaxCount     int    `gorm:"default:2"`
	Count        int    `gorm:"default:1"`
}
