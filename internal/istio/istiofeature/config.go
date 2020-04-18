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

package istiofeature

type StaticConfig struct {
	Istio struct {
		GrafanaDashboardLocation string
		PilotImage               string
		MixerImage               string
		ProxyImage               string
		SidecarInjectorImage     string
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

		NodeExporter struct {
			Chart   string
			Version string
		}
	}
}
