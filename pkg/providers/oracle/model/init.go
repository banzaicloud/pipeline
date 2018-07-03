package model

import (
	"github.com/banzaicloud/pipeline/database"
	"github.com/sirupsen/logrus"
)

// Init initializes the models
func Init(logger *logrus.Entry) error {

	logger.Info("Create Oracle provider table(s):",
		Cluster.TableName(Cluster{}),
		NodePool.TableName(NodePool{}),
		NodePoolSubnet.TableName(NodePoolSubnet{}),
		NodePoolLabel.TableName(NodePoolLabel{}),
	)

	return database.GetDB().AutoMigrate(
		&Cluster{},
		&NodePool{},
		&NodePoolSubnet{},
		&NodePoolLabel{},
		&Profile{},
		&ProfileNodePool{},
		&ProfileNodePoolSubnet{},
		&ProfileNodePoolLabel{},
	).Error
}
