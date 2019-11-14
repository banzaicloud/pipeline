// Copyright © 2019 Banzai Cloud
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

package logging

type loggingOperatorValues struct {
	Image imageValues `json:"image" mapstructure:"image"`
}

type imageValues struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Tag        string `json:"tag" mapstructure:"tag"`
}

type loggingOperatorLoggingValues struct {
	Tls       tlsValues    `json:"tls" mapstructure:"tls"`
	Fluentbit fluentValues `json:"fluentbit" mapstructure:"fluentbit"`
	Fluentd   fluentValues `json:"fluentd" mapstructure:"fluentd"`
}

type fluentValues struct {
	Enabled bool        `json:"enabled" mapstructure:"enabled"`
	Image   imageValues `json:"image" mapstructure:"image"`
}

type tlsValues struct {
	Enabled             bool   `json:"enabled" mapstructure:"enabled"`
	FluentdSecretName   string `json:"fluentdSecretName" mapstructure:"fluentdSecretName"`
	FluentbitSecretName string `json:"fluentbitSecretName" mapstructure:"fluentbitSecretName"`
}

type lokiValues struct {
	Ingress     ingressValues          `json:"ingress" mapstructure:"ingress"`
	Annotations map[string]interface{} `json:"annotations,omitempty" mapstructure:"annotations"`
	Image       imageValues            `json:"image"`
}

type ingressValues struct {
	Enabled bool     `json:"enabled" mapstructure:"enabled"`
	Hosts   []string `json:"hosts" mapstructure:"hosts"`
	Path    string   `json:"path,omitempty" mapstructure:"path"`
}
