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

package providers

import (
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	logrusadapter "logur.dev/adapter/logrus"

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/providers/alibaba"
	"github.com/banzaicloud/pipeline/internal/providers/amazon"
	"github.com/banzaicloud/pipeline/internal/providers/azure"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	"github.com/banzaicloud/pipeline/internal/providers/google"
	"github.com/banzaicloud/pipeline/internal/providers/oracle"
	"github.com/banzaicloud/pipeline/internal/providers/pke"
)

// Migrate runs migrations for cloud provider services.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	if err := alibaba.Migrate(db, logger); err != nil {
		return err
	}

	if err := amazon.Migrate(db, logger); err != nil {
		return err
	}

	if err := azure.Migrate(db, logger); err != nil {
		return err
	}

	if err := google.Migrate(db, logger); err != nil {
		return err
	}

	if err := oracle.Migrate(db, logger); err != nil {
		return err
	}

	if err := pke.Migrate(db, logger); err != nil {
		return err
	}

	var logurLogger *logrusadapter.Logger
	switch l := logger.(type) {
	case *logrus.Logger:
		logurLogger = logrusadapter.New(l)
	case *logrus.Entry:
		logurLogger = logrusadapter.NewFromEntry(l)
	}

	if err := adapter.Migrate(db, commonadapter.NewLogger(logurLogger)); err != nil {
		return err
	}

	return nil
}
