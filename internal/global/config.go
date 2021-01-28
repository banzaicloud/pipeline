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
	"time"
)

// Config is a global config instance.
// Deprecated: Use only if you must! Try not to extend with new values!
// nolint: gochecknoglobals
var Config struct {
	Auth struct {
		Cookie struct {
			Secure    bool
			SetDomain bool
		}
		OIDC struct {
			Issuer string
		}
		Token struct {
			Audience string
			Issuer   string
		}
	}
	Cloud struct {
		Amazon struct {
			DefaultRegion string
		}
	}
	Cluster struct {
		Namespace string
		Autoscale struct {
			Namespace string
			Charts    struct {
				ClusterAutoscaler struct {
					Chart                   string
					Version                 string
					ImageVersionConstraints []struct {
						K8sVersion string
						Repository string
						Tag        string
					}
				}
				HPAOperator struct {
					Chart   string
					Version string
					Values  map[string]interface{}
				}
			}
			HPA struct {
				Prometheus struct {
					LocalPort      int
					ServiceContext string
					ServiceName    string
				}
			}
		}
		DisasterRecovery struct {
			Namespace string
			Ark       struct {
				RestoreWaitTimeout time.Duration
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
						AwsPluginImage struct {
							Repository string
							Tag        string
							PullPolicy string
						}
						AzurePluginImage struct {
							Repository string
							Tag        string
							PullPolicy string
						}
						GcpPluginImage struct {
							Repository string
							Tag        string
							PullPolicy string
						}
					}
				}
			}
		}
		DNS struct {
			Enabled        bool
			BaseDomain     string
			ProviderSecret string
		}
		Labels struct {
			Namespace string
		}
		PostHook struct {
			Autoscaler struct {
				Enabled bool
			}
			Dashboard struct {
				Enabled bool
				Chart   string
				Version string
			}
			HPA struct {
				Enabled bool
			}
			Ingress struct {
				Enabled bool
				Chart   string
				Version string
				Values  map[string]interface{}
			}
			ITH struct {
				Enabled bool
				Chart   string
				Version string
			}
			Spotconfig struct {
				Enabled bool
				Charts  struct {
					Scheduler struct {
						Chart   string
						Version string
					}
					Webhook struct {
						Chart   string
						Version string
					}
				}
			}
		}
	}
	Dex struct {
		APIAddr string
		APICa   string
	}
	Distribution struct {
		EKS struct {
			ExposeAdminKubeconfig bool
			TemplateLocation      string
			SSH                   struct {
				Generate bool
			}
		}
		PKE struct {
			Amazon struct {
				DefaultNetworkProvider string
			}
		}
	}
	Hollowtrees struct {
		Endpoint        string
		TokenSigningKey string
	}
	Kubernetes struct {
		Client struct {
			ForceGlobal bool
		}
	}
	Pipeline struct {
		External struct {
			URL string
		}
		UUID       string
		Enterprise bool
	}
	Telemetry struct {
		Debug bool
	}
}
