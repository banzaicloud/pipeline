// Copyright © 2020 Banzai Cloud
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

package ingress

import (
	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
)

type Spec struct {
	Controller   ControllerSpec `json:"controller" mapstructure:"controller"`
	IngressClass string         `json:"ingressClass" mapstructure:"ingressClass"`
	Service      ServiceSpec    `json:"service" mapstructure:"service"`
}

func (s Spec) Validate(config Config) error {
	return errors.Combine(s.Controller.Validate(config), s.Service.Validate())
}

type ControllerSpec struct {
	Type          string                 `json:"type" mapstructure:"type"`
	RawConfig     map[string]interface{} `json:"config" mapstructure:"config"`
	traefikConfig *TraefikConfigSpec
}

func (s ControllerSpec) Validate(config Config) error {
	var errs error

	if !contains(config.Controllers, s.Type) {
		errs = errors.Append(errs, unavailableControllerError{
			Controller: s.Type,
		})
	}

	switch s.Type {
	case ControllerTraefik:
		cfg, err := s.TraefikConfig()
		if err != nil {
			errs = errors.Append(errs, err)
		}

		errs = errors.Append(errs, cfg.Validate())
	}

	return errs
}

func (s *ControllerSpec) TraefikConfig() (TraefikConfigSpec, error) {
	if s.traefikConfig == nil {
		s.traefikConfig = new(TraefikConfigSpec)
		if err := mapstructure.Decode(s.RawConfig, s.traefikConfig); err != nil {
			return TraefikConfigSpec{}, errors.WrapIf(err, "failed to decode config values as traefik config")
		}
	}
	return *s.traefikConfig, nil
}

type TraefikConfigSpec struct {
	SSL TraefikSSLSpec `json:"ssl" mapstructure:"ssl"`
}

func (s TraefikConfigSpec) Validate() error {
	return nil
}

type TraefikSSLSpec struct {
	DefaultCN      string   `json:"defaultCN" mapstructure:"defaultCN"`
	DefaultIPList  []string `json:"defaultIPList" mapstructure:"defaultIPList"`
	DefaultSANList []string `json:"defaultSANList" mapstructure:"defaultSANList"`
}

type ServiceSpec struct {
	Type        string            `json:"type" mapstructure:"type"`
	Annotations map[string]string `json:"annotations" mapstructure:"annotations"`
}

func (s ServiceSpec) Validate() error {
	switch s.Type {
	case "", ServiceTypeClusterIP, ServiceTypeLoadBalancer, ServiceTypeNodePort:
		return nil
	default:
		return unsupportedServiceTypeError{
			ServiceType: s.Type,
		}
	}
}

func contains(slice []string, str string) bool {
	for _, e := range slice {
		if e == str {
			return true
		}
	}
	return false
}
