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
	"time"

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
		Namespace string

		Ingress struct {
			Cert struct {
				Source string
				Path   string
			}
		}

		Labels struct {
			Namespace string

			Domain           string
			ForbiddenDomains []string

			Charts struct {
				NodepoolLabelOperator struct {
					Chart   string
					Version string
				}
			}
		}

		Vault struct {
			Namespace string

			Charts struct {
				Webhook struct {
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
			Namespace      string
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

		Autoscale struct {
			Namespace string

			Charts struct {
				ClusterAutoscaler struct {
					Chart   string
					Version string
				}

				HPAOperator struct {
					Chart   string
					Version string
				}
			}
		}

		DisasterRecovery struct {
			Namespace string

			Ark struct {
				SyncEnabled         bool
				BucketSyncInterval  time.Duration
				RestoreSyncInterval time.Duration
				BackupSyncInterval  time.Duration
				RestoreWaitTimeout  time.Duration
			}

			Charts struct {
				Ark struct {
					Chart   string
					Version string
					Values  struct {
						Image struct {
							Repository string
							Tag        string
							PullPolicy string
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

	Cloudinfo struct {
		Endpoint string
	}

	Hooks struct {
		DomainHookDisabled bool
	}

	Spotguide struct {
		AllowPrereleases                bool
		AllowPrivateRepos               bool
		SyncInterval                    time.Duration
		SharedLibraryGitHubOrganization string
	}
}

func (c *Configuration) Process() error {
	if c.Cluster.Labels.Namespace == "" {
		c.Cluster.Labels.Namespace = c.Cluster.Namespace
	}

	if c.Cluster.Vault.Namespace == "" {
		c.Cluster.Vault.Namespace = c.Cluster.Namespace
	}

	if c.Cluster.DNS.Namespace == "" {
		c.Cluster.DNS.Namespace = c.Cluster.Namespace
	}

	if c.Cluster.Autoscale.Namespace == "" {
		c.Cluster.Autoscale.Namespace = c.Cluster.Namespace
	}

	if c.Cluster.DisasterRecovery.Namespace == "" {
		c.Cluster.DisasterRecovery.Namespace = c.Cluster.Namespace
	}

	return nil
}

// GetHelmPath returns local helm path
func GetHelmPath(organizationName string) string {
	return filepath.Join(Config.Helm.Home, organizationName)
}
