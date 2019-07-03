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

package cluster

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/goph/logur"
	"github.com/goph/logur/adapters/logrusadapter"
	"github.com/sirupsen/logrus"
)

// nolint: gochecknoglobals
var log logrus.FieldLogger

func init() {
	log = config.Logger()
}

func NewLogurLogger(fl logrus.FieldLogger) logur.Logger {
	if l, ok := fl.(*logrus.Logger); ok {
		return logrusadapter.New(l)
	}

	entry, ok := fl.(*logrus.Entry)
	if !ok {
		entry = fl.WithFields(logrus.Fields{})
	}

	logger := logrusadapter.New(entry.Logger)

	return logur.WithFields(logger, entry.Data)
}
