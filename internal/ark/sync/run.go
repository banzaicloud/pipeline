// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sync

import (
	"context"
	"time"

	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/cluster"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
)

// RunSyncServices runs ARK sync services
func RunSyncServices(
	context context.Context,
	db *gorm.DB,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
) {

	clusterManager := cluster.NewManager(
		intCluster.NewClusters(db),
		providers.NewSecretValidator(secret.Store),
		logger, errorHandler,
	)

	svc := NewSyncService(
		clusterManager,
		time.Duration(10)*time.Minute,
		time.Duration(20)*time.Second,
		time.Duration(20)*time.Second,
	)

	svc.Run(context, db, logger)
}
