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

package global

import (
	"path/filepath"

	"github.com/banzaicloud/pipeline/internal/platform/log"
)

// Config is a global config instance.
// nolint: gochecknoglobals
var Config Configuration

// Configuration exposes various config options used globally.
type Configuration struct {
	Log log.Config

	Telemetry struct {
		Debug bool
	}

	Cluster struct {
		Logging struct {
			Charts struct {
				Operator struct {
					Chart   string
					Version string
					Values  struct {
						Image struct {
							Repository string
							Tag        string
						}
					}
				}
			}
		}

		DNS struct {
			Enabled        bool
			BaseDomain     string
			ProviderSecret string

			Charts struct {
				ExternalDNS struct {
					Chart   string
					Version string
					Values  struct {
						Image struct {
							Repository string
							Tag        string
						}
					}
				}
			}
		}
	}

	Helm struct {
		Tiller struct {
			Version string
		}

		Home string

		Repositories map[string]string
	}

	Cloud struct {
		Amazon struct {
			DefaultRegion string
		}

		Alibaba struct {
			DefaultRegion string
		}
	}

	Hooks struct {
		DomainHookDisabled bool
	}
}

// GetHelmPath returns local helm path
func GetHelmPath(organizationName string) string {
	return filepath.Join(Config.Helm.Home, organizationName)
}
