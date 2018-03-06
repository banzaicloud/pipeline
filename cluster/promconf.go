package cluster

import (
	"github.com/prometheus/common/model"
	promcfg "github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	"net/url"
)

//GenerateConfig generates prometheus config
func GenerateConfig(prometheusCfg []PrometheusCfg) []byte {
	//Set Global Config
	config := promcfg.Config{}
	config.AlertingConfig = promcfg.AlertingConfig{
		AlertmanagerConfigs: []*promcfg.AlertmanagerConfig{
			{
				ServiceDiscoveryConfig: promcfg.ServiceDiscoveryConfig{
					KubernetesSDConfigs: []*promcfg.KubernetesSDConfig{
						{
							Role: promcfg.KubernetesRole("pod"),
							TLSConfig: promcfg.TLSConfig{
								CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
							},
							BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
						},
					},
				},
				RelabelConfigs: []*promcfg.RelabelConfig{
					{
						SourceLabels: model.LabelNames{
							model.LabelName("__meta_kubernetes_namespace"),
						},
						Action: "keep",
						Regex:  promcfg.MustNewRegexp("default"),
					},
					{
						SourceLabels: model.LabelNames{
							model.LabelName("___meta_kubernetes_pod_label_app"),
						},
						Action: "keep",
						Regex:  promcfg.MustNewRegexp("prometheus"),
					},
					{
						SourceLabels: model.LabelNames{
							model.LabelName("___meta_kubernetes_pod_label_component"),
						},
						Action: "keep",
						Regex:  promcfg.MustNewRegexp("alertmanager"),
					},
					{
						SourceLabels: model.LabelNames{
							model.LabelName("__meta_kubernetes_pod_container_port_number"),
						},
						Action: "drop",
						Regex:  promcfg.MustNewRegexp(""),
					},
				},
			},
		},
	}
	config.GlobalConfig = promcfg.GlobalConfig{}
	duration, _ := model.ParseDuration("15s")
	config.GlobalConfig.EvaluationInterval = duration
	duration, _ = model.ParseDuration("15s")
	config.GlobalConfig.ScrapeInterval = duration
	duration, _ = model.ParseDuration("7s")
	config.GlobalConfig.ScrapeTimeout = duration
	config.RuleFiles = []string{
		"/etc/config/*.rules",
	}
	//Set Scrape Config
	var ScrapeConfigs []*promcfg.ScrapeConfig
	for _, cluster := range prometheusCfg {

		scrapeConfig := promcfg.ScrapeConfig{}
		scrapeConfig.JobName = cluster.Name
		scrapeConfig.HonorLabels = true
		scrapeConfig.MetricsPath = "/api/v1/namespaces/default/services/monitor-prometheus-server:80/proxy/prometheus/federate"
		scrapeConfig.Scheme = "https"
		scrapeConfig.Params = url.Values{
			"match[]": {
				`{job="kubernetes-nodes"}`,
				`{job="kubernetes-pods"}`,
				`{job="kubernetes-apiservers"}`,
				`{job="kubernetes-service-endpoints"}`,
				`{job="kubernetes-cadvisor"}`,
				`{job="banzaicloud-pushgateway"}`,
				`{job="node_exporter"}`,
			},
		}
		scrapeConfig.RelabelConfigs = []*promcfg.RelabelConfig{
			{
				SourceLabels: model.LabelNames{
					model.LabelName("__address__"),
				},
				Action:      "replace",
				Regex:       promcfg.MustNewRegexp(`(.+):(?:\d+)`),
				Replacement: "${1}",
				TargetLabel: "cluster",
			},
		}
		scrapeConfig.HTTPClientConfig = promcfg.HTTPClientConfig{
			TLSConfig: promcfg.TLSConfig{
				CAFile:             cluster.CaFilePath,
				CertFile:           cluster.CertFilePath,
				KeyFile:            cluster.KeyFile,
				InsecureSkipVerify: true,
			},
		}
		scrapeConfig.ServiceDiscoveryConfig = promcfg.ServiceDiscoveryConfig{
			StaticConfigs: []*promcfg.TargetGroup{
				{
					Targets: []model.LabelSet{
						{
							model.AddressLabel: model.LabelValue(cluster.Endpoint),
						},
					},
					Labels: model.LabelSet{"cluster_name": model.LabelValue(cluster.Name)},
				},
			},
		}
		ScrapeConfigs = append(ScrapeConfigs, &scrapeConfig)
	}

	config.ScrapeConfigs = ScrapeConfigs

	// Reload configuration?
	out, err := yaml.Marshal(config)
	if err != nil {
		log.Errorf("%v", err)
	}
	return out

}
