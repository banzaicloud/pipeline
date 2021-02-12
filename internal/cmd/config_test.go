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

package cmd

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/ingress"
	"github.com/banzaicloud/pipeline/pkg/hook"
	"github.com/banzaicloud/pipeline/pkg/values"
)

func TestConfigure_DefaultValueBinding(t *testing.T) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("::"),
	)
	p := pflag.NewFlagSet("test", pflag.ContinueOnError)

	Configure(v, p)

	var config Config
	err := v.Unmarshal(&config, hook.DecodeHookWithDefaults())
	require.NoError(t, err)

	testCases := map[string]struct {
		Subtree  interface{}
		Expected interface{}
	}{
		"cluster DNS": {
			Subtree: config.Cluster.DNS,
			Expected: ClusterDNSConfig{
				Enabled: true,
				Config: dns.Config{
					ProviderSecret: "secret/data/banzaicloud/aws",
					Charts: dns.ChartsConfig{
						ExternalDNS: dns.ExternalDNSChartConfig{
							ChartConfigBase: dns.ChartConfigBase{
								Chart:   "bitnami/external-dns",
								Version: "4.5.0",
							},
							Values: dns.ExternalDNSChartValuesConfig{
								Image: dns.ExternalDNSChartValuesImageConfig{
									Registry:   "k8s.gcr.io",
									Repository: "external-dns/external-dns",
									Tag:        "v0.7.5",
								},
							},
						},
					},
				},
			},
		},
		"cluster ingress": {
			Subtree: config.Cluster.Ingress,
			Expected: ClusterIngressConfig{
				Enabled: false,
				Config: ingress.Config{
					ReleaseName: "ingress",
					Controllers: []string{
						"traefik",
					},
					Charts: ingress.ChartsConfig{
						Traefik: ingress.TraefikChartConfig{
							Chart:   "stable/traefik",
							Version: "1.86.2",
							Values: values.Config(map[string]interface{}{
								"rbac": map[string]interface{}{
									"enabled": true,
								},
								"ssl": map[string]interface{}{
									"enabled":     true,
									"generateTLS": true,
								},
							}),
						},
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			require.Equal(t, testCase.Expected, testCase.Subtree)
		})
	}
}
