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

package autoscaling

import (
	"github.com/sirupsen/logrus"
)

// Option sets configuration on the Manager
type Option interface {
	apply(*Manager)
}

// MetricsEnabled turns on Prometheus metrics
type MetricsEnabled bool

func (o MetricsEnabled) apply(m *Manager) {
	m.metricsEnabled = bool(o)
}

// Logger sets an initialised outside logger
type Logger struct {
	logrus.FieldLogger
}

func (o Logger) apply(m *Manager) {
	m.logger = o.FieldLogger
}
