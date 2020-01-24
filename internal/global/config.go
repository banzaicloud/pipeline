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
	"errors"
	"path/filepath"
	"time"

	"github.com/banzaicloud/pipeline/internal/anchore"
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

	Pipeline struct {
		UUID     string
		External struct {
			URL      string
			Insecure bool
		}
	}

	Auth struct {
		OIDC struct {
			Issuer       string
			Insecure     bool
			ClientID     string
			ClientSecret string
		}

		Cookie struct {
			Secure    bool
			SetDomain bool
		}

		Token struct {
			Issuer   string
			Audience string
		}
	}

	Dex struct {
		APIAddr string
		APICa   string
	}

	Kubernetes struct {
		Client struct {
			ForceGlobal bool
		}
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

		Monitoring struct {
			Grafana struct {
				AdminUser string
			}
		}

		DNS struct {
			Enabled        bool
			Namespace      string
			BaseDomain     string
			ProviderSecret string

			Charts struct {
				ExternalDNS struct {
					Chart string
				}
			}
		}

		SecurityScan struct {
			Anchore struct {
				Enabled bool

				anchore.Config `mapstructure:",squash"`
			}
		}

		Autoscale struct {
			Namespace string

			HPA struct {
				Prometheus struct {
					ServiceName    string
					ServiceContext string
					LocalPort      int
				}
			}

			Charts struct {
				ClusterAutoscaler struct {
					Chart                   string
					Version                 string
					ImageVersionConstraints []struct {
						K8sVersion string
						Tag        string
						Repository string
					}
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

		Backyards struct {
			Istio struct {
				GrafanaDashboardLocation string
				PilotImage               string
				MixerImage               string
			}

			Charts struct {
				IstioOperator struct {
					Chart   string
					Version string
					Values  struct {
						Operator struct {
							Image struct {
								Repository string
								Tag        string
							}
						}
					}
				}

				Backyards struct {
					Chart   string
					Version string
					Values  struct {
						Application struct {
							Image struct {
								Repository string
								Tag        string
							}
						}

						Web struct {
							Image struct {
								Repository string
								Tag        string
							}
						}
					}
				}

				CanaryOperator struct {
					Chart   string
					Version string
					Values  struct {
						Operator struct {
							Image struct {
								Repository string
								Tag        string
							}
						}
					}
				}
			}
		}

		Federation struct {
			Charts struct {
				Kubefed struct {
					Chart   string
					Version string
					Values  struct {
						ControllerManager struct {
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

	Distribution struct {
		EKS struct {
			TemplateLocation      string
			ExposeAdminKubeconfig bool
		}
	}

	Hollowtrees struct {
		Endpoint        string
		TokenSigningKey string
	}

	CICD struct {
		Enabled  bool
		URL      string
		Insecure bool
		SCM      string
	}

	Github struct {
		Token string
	}

	Gitlab struct {
		URL   string
		Token string
	}

	Spotguide struct {
		AllowPrereleases                bool
		AllowPrivateRepos               bool
		SyncInterval                    time.Duration
		SharedLibraryGitHubOrganization string
	}

	Secret struct {
		TLS struct {
			DefaultValidity time.Duration
		}
	}
}

func (c Configuration) Validate() error {
	if c.Auth.OIDC.Issuer == "" {
		return errors.New("auth oidc issuer is required")
	}

	if c.Auth.OIDC.ClientID == "" {
		return errors.New("auth oidc client ID is required")
	}

	if c.Auth.OIDC.ClientSecret == "" {
		return errors.New("auth oidc client secret is required")
	}

	if c.CICD.Enabled {
		if c.CICD.URL == "" {
			return errors.New("cicd url is required")
		}

		switch c.CICD.SCM {
		case "github":
			if c.Github.Token == "" {
				return errors.New("github token is required")
			}

		case "gitlab":
			if c.Gitlab.URL == "" {
				return errors.New("gitlab url is required")
			}

			if c.Gitlab.Token == "" {
				return errors.New("gitlab token is required")
			}

		default:
			return errors.New("cicd scm is required")
		}
	}

	return nil
}

func (c *Configuration) Process() error {
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

	if c.Cluster.Labels.Namespace == "" {
		c.Cluster.Labels.Namespace = c.Cluster.Namespace
	}

	return nil
}

// GetHelmPath returns local helm path
func GetHelmPath(organizationName string) string {
	return filepath.Join(Config.Helm.Home, organizationName)
}
