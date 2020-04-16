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

package main

import (
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification/notificationadapter"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/providers/alibaba/alibabaadapter"
	"github.com/banzaicloud/pipeline/internal/providers/azure/azureadapter"
	"github.com/banzaicloud/pipeline/internal/providers/kubernetes/kubernetesadapter"
	"github.com/banzaicloud/pipeline/src/model"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/api/middleware/audit"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/process/processadapter"
	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/clustergroup"
	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/banzaicloud/pipeline/src/auth"
	route53model "github.com/banzaicloud/pipeline/src/dns/route53/model"
	"github.com/banzaicloud/pipeline/src/spotguide"
)

// Migrate runs migrations for the application.
func Migrate(db *gorm.DB, logger logrus.FieldLogger, commonLogger common.Logger) error {
	if err := clustermodel.Migrate(db, logger); err != nil {
		return err
	}

	if err := alibabaadapter.Migrate(db, logger); err != nil {
		return err
	}

	if err := eksmodel.Migrate(db, logger); err != nil {
		return err
	}

	if err := azureadapter.Migrate(db, logger); err != nil {
		return err
	}

	if err := kubernetesadapter.Migrate(db, logger); err != nil {
		return err
	}

	if err := model.Migrate(db, logger); err != nil {
		return err
	}

	if err := auth.Migrate(db, logger); err != nil {
		return err
	}

	if err := route53model.Migrate(db, logger); err != nil {
		return err
	}

	if err := spotguide.Migrate(db, logger); err != nil {
		return err
	}

	if err := audit.Migrate(db, logger); err != nil {
		return err
	}

	if err := clustergroup.Migrate(db, logger); err != nil {
		return err
	}

	if err := deployment.Migrate(db, logger); err != nil {
		return err
	}

	if err := providers.Migrate(db, logger); err != nil {
		return err
	}

	if err := ark.Migrate(db, logger); err != nil {
		return err
	}

	if err := notificationadapter.Migrate(db, commonLogger); err != nil {
		return err
	}

	if err := integratedserviceadapter.Migrate(db, logger); err != nil {
		return err
	}

	if err := helmadapter.Migrate(db, commonLogger); err != nil {
		return err
	}

	if err := processadapter.Migrate(db, commonLogger); err != nil {
		return err
	}

	return nil
}
