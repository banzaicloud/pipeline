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

package monitor

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	pipCluster "github.com/banzaicloud/pipeline/cluster"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	pipSecret "github.com/banzaicloud/pipeline/secret"
	promconfig "github.com/banzaicloud/prometheus-config"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	promCommon "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type clusterSubscriber struct {
	client  kubernetes.Interface
	manager *cluster.Manager
	db      *gorm.DB

	dnsBaseDomain          string
	controlPlaneNamespace  string
	pipelineNamespace      string
	configMap              string
	configMapPrometheusKey string
	certSecret             string
	certMountPath          string

	// TODO: find a better way to avoid config race condition (eg. occasional flush)
	mu           sync.Mutex
	errorHandler emperror.Handler
}

func NewClusterSubscriber(
	client kubernetes.Interface,
	manager *cluster.Manager,
	db *gorm.DB,
	dnsBaseDomain string,
	controlPlaneNamespace string,
	pipelineNamespace string,
	configMap string,
	configMapPrometheusKey string,
	certSecret string,
	certMountPath string,
	errorHandler emperror.Handler,
) *clusterSubscriber {
	return &clusterSubscriber{
		client:                 client,
		manager:                manager,
		db:                     db,
		dnsBaseDomain:          dnsBaseDomain,
		controlPlaneNamespace:  controlPlaneNamespace,
		pipelineNamespace:      pipelineNamespace,
		configMap:              configMap,
		configMapPrometheusKey: configMapPrometheusKey,
		certSecret:             certSecret,
		certMountPath:          certMountPath,

		errorHandler: errorHandler,
	}
}

func (s *clusterSubscriber) Init() {
	s.mu.Lock()
	defer s.mu.Unlock()

	prometheusConfig, prometheusSecret, err := s.getPrometheusConfigAndSecret()
	if err != nil {
		s.errorHandler.Handle(err)

		return
	}
	clusters, err := s.manager.GetAllClusters(context.Background())
	if err != nil {
		s.errorHandler.Handle(err)

		return
	}
	prometheusConfig.ScrapeConfigs = []*promconfig.ScrapeConfig{}
	for _, c := range clusters {
		clusterStatus, err := c.GetStatus()
		if err != nil {
			s.errorHandler.Handle(err)

			return
		}
		if clusterStatus.Monitoring {
			org, err := s.getOrganization(c.GetOrganizationId())
			if err != nil {
				s.errorHandler.Handle(err)

				return
			}
			basicAuthSecret, err := pipSecret.Store.GetByName(org.ID, fmt.Sprintf("cluster-%d-prometheus", c.GetID()))
			if err != nil {
				s.errorHandler.Handle(emperror.Wrap(err, "failed to get prometheus secret"))

				return
			}
			tlsSecret, err := pipSecret.Store.GetByName(org.ID, pipCluster.DefaultCertSecretName)
			if err != nil {
				s.errorHandler.Handle(emperror.Wrap(err, "failed to get ingress secret"))

				return
			}

			params := scrapeConfigParameters{
				orgName:     org.Name,
				clusterName: c.GetName(),
				endpoint:    fmt.Sprintf("%s.%s.%s", c.GetName(), org.Name, s.dnsBaseDomain),
				basicAuthConfig: &basicAuthConfig{
					username:     string(basicAuthSecret.Values[pkgSecret.Username]),
					password:     string(basicAuthSecret.Values[pkgSecret.Password]),
					passwordFile: fmt.Sprintf("%s_%s_basic_auth.conf", org.Name, c.GetName()),
				},
				tlsConfig: &scrapeTLSConfig{
					caCertFileName: fmt.Sprintf("%s_%s_certificate-authority-data.pem", org.Name, c.GetName()),
				},
			}

			prometheusSecret.StringData[params.basicAuthConfig.passwordFile] = string(basicAuthSecret.Values[pkgSecret.Password])

			prometheusConfig.ScrapeConfigs = append(prometheusConfig.ScrapeConfigs, s.getScrapeConfigForCluster(params))
			prometheusSecret.StringData[params.tlsConfig.caCertFileName] = string(tlsSecret.Values[pkgSecret.CACert])
		}

	}
	err = s.save(prometheusConfig, prometheusSecret)
	if err != nil {
		s.errorHandler.Handle(err)

		return
	}
}

func (s *clusterSubscriber) Register(events clusterEvents) {
	events.NotifyClusterCreated(s.AddClusterToPrometheusConfig)
	events.NotifyClusterDeleted(s.RemoveClusterFromPrometheusConfig)
}

type scrapeConfigParameters struct {
	orgName         string
	clusterName     string
	endpoint        string
	tlsConfig       *scrapeTLSConfig
	basicAuthConfig *basicAuthConfig
}

type scrapeTLSConfig struct {
	caCertFileName string
	certFileName   string
	keyFileName    string
}

type basicAuthConfig struct {
	username     string
	password     string
	passwordFile string
}

func (s *clusterSubscriber) AddClusterToPrometheusConfig(clusterID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, org, prometheusConfig, prometheusSecret, err := s.init(clusterID)
	if err != nil {
		s.errorHandler.Handle(err)

		return
	}

	basicAuthSecret, err := pipSecret.Store.GetByName(org.ID, fmt.Sprintf("cluster-%d-prometheus", clusterID))
	if err != nil {
		s.errorHandler.Handle(emperror.Wrap(err, "failed to get prometheus secret"))

		return
	}

	params := scrapeConfigParameters{
		orgName:     org.Name,
		clusterName: c.GetName(),
		endpoint:    fmt.Sprintf("%s.%s.%s", c.GetName(), org.Name, s.dnsBaseDomain),
		basicAuthConfig: &basicAuthConfig{
			username: basicAuthSecret.Values[pkgSecret.Username],
			// password:     basicAuthSecret.Values[pkgSecret.Password],
			passwordFile: fmt.Sprintf("%s_%s_basic_auth.conf", org.Name, c.GetName()),
		},
		tlsConfig: &scrapeTLSConfig{
			caCertFileName: fmt.Sprintf("%s_%s_certificate-authority-data.pem", org.Name, c.GetName()),
		},
	}
	tlsSecret, err := pipSecret.Store.GetByName(org.ID, pipCluster.DefaultCertSecretName)
	if err != nil {
		s.errorHandler.Handle(emperror.Wrap(err, "failed to get ingress secret"))

		return
	}

	prometheusConfig.ScrapeConfigs = append(prometheusConfig.ScrapeConfigs, s.getScrapeConfigForCluster(params))
	prometheusSecret.StringData[params.tlsConfig.caCertFileName] = string(tlsSecret.Values[pkgSecret.CACert])

	prometheusSecret.StringData[params.basicAuthConfig.passwordFile] = string(basicAuthSecret.Values[pkgSecret.Password])

	err = s.save(prometheusConfig, prometheusSecret)
	if err != nil {
		s.errorHandler.Handle(err)
		return
	}
}

func (s *clusterSubscriber) RemoveClusterFromPrometheusConfig(orgID uint, clusterName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	org, err := s.getOrganization(orgID)
	if err != nil {
		s.errorHandler.Handle(err)

		return
	}

	prometheusConfig, _, err := s.getPrometheusConfigAndSecret()
	if err != nil {
		s.errorHandler.Handle(err)

		return
	}

	var scrapeConfigs []*promconfig.ScrapeConfig

	for _, scrapeConfig := range prometheusConfig.ScrapeConfigs {
		if scrapeConfig.JobName == fmt.Sprintf("%s-%s", org.Name, clusterName) {
			continue
		}

		scrapeConfigs = append(scrapeConfigs, scrapeConfig)
	}

	prometheusConfig.ScrapeConfigs = scrapeConfigs

	err = s.save(prometheusConfig, nil)
	if err != nil {
		s.errorHandler.Handle(err)

		return
	}
}

func (s *clusterSubscriber) init(clusterID uint) (cluster.CommonCluster, *auth.Organization, *promconfig.Config, *v1.Secret, error) {
	c, org, err := s.getClusterAndOrganization(clusterID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	prometheusConfig, secret, err := s.getPrometheusConfigAndSecret()

	return c, org, prometheusConfig, secret, err
}

func (s *clusterSubscriber) getClusterAndOrganization(clusterID uint) (cluster.CommonCluster, *auth.Organization, error) {
	c, err := s.manager.GetClusterByIDOnly(context.Background(), clusterID)
	if err != nil {
		return nil, nil, err
	}

	org, err := s.getOrganization(c.GetOrganizationId())

	return c, org, err
}

func (s *clusterSubscriber) getOrganization(orgID uint) (*auth.Organization, error) {
	org := auth.Organization{
		ID: orgID,
	}

	err := s.db.Where(org).First(&org).Error
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get organization")
	}

	return &org, nil
}

func (s *clusterSubscriber) getPrometheusConfigAndSecret() (*promconfig.Config, *v1.Secret, error) {
	prometheusConfig, err := s.getPrometheusConfig()
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to get prometheus config")
	}

	if prometheusConfig.ScrapeConfigs == nil {
		prometheusConfig.ScrapeConfigs = []*promconfig.ScrapeConfig{}
	}

	secret, err := s.client.CoreV1().Secrets(s.controlPlaneNamespace).Get(s.certSecret, metav1.GetOptions{})
	if err != nil {
		return nil, nil, emperror.With(
			emperror.Wrap(err, "failed to get cert secret"),
			"secret", s.certSecret,
			"namespace", s.controlPlaneNamespace,
		)
	}

	if secret.StringData == nil {
		secret.StringData = map[string]string{}
	}

	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	return prometheusConfig, secret, nil
}

func (s *clusterSubscriber) save(prometheusConfig *promconfig.Config, secret *v1.Secret) error {
	if secret != nil {
		_, err := s.client.CoreV1().Secrets(s.controlPlaneNamespace).Update(secret)
		if err != nil {
			return emperror.With(
				emperror.Wrap(err, "failed to update secret"),
				"secret", s.certSecret,
				"namespace", s.controlPlaneNamespace,
			)
		}
	}
	err := s.savePrometheusConfig(prometheusConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to save prometheus config")
	}

	return nil
}

func (s *clusterSubscriber) getPrometheusConfig() (*promconfig.Config, error) {
	configMap, err := s.client.CoreV1().ConfigMaps(s.controlPlaneNamespace).Get(s.configMap, metav1.GetOptions{})
	if err != nil {
		return nil, emperror.With(
			emperror.Wrap(err, "failed to get configmap"),
			"configMap", s.configMap,
			"namespace", s.controlPlaneNamespace,
		)
	}

	rawPrometheusConfig, ok := configMap.Data[s.configMapPrometheusKey]
	if !ok {
		return nil, emperror.With(
			errors.New("could not find prometheus config"),
			"prometheusKey", s.configMapPrometheusKey,
			"configMap", s.configMap,
			"namespace", s.controlPlaneNamespace,
		)
	}

	config := &promconfig.Config{}

	err = yaml.Unmarshal([]byte(rawPrometheusConfig), config)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to parse prometheus config")
	}

	return config, nil
}

func (s *clusterSubscriber) savePrometheusConfig(config *promconfig.Config) error {
	configMap, err := s.client.CoreV1().ConfigMaps(s.controlPlaneNamespace).Get(s.configMap, metav1.GetOptions{})
	if err != nil {
		return emperror.With(
			emperror.Wrap(err, "failed to get configmap"),
			"configMap", s.configMap,
			"namespace", s.controlPlaneNamespace,
		)
	}

	rawPrometheusConfig, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "failed to marshal prometheus config")
	}

	configMap.Data[s.configMapPrometheusKey] = string(rawPrometheusConfig)

	_, err = s.client.CoreV1().ConfigMaps(s.controlPlaneNamespace).Update(configMap)
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func (s *clusterSubscriber) getScrapeConfigForCluster(params scrapeConfigParameters) *promconfig.ScrapeConfig {
	scrapeConfig := &promconfig.ScrapeConfig{
		JobName:     fmt.Sprintf("%s-%s", params.orgName, params.clusterName),
		HonorLabels: true,
		MetricsPath: "/prometheus/federate",
		Scheme:      "https",
		Params: url.Values{
			"match[]": {
				`{job="kubernetes-nodes"}`,
				`{job="kubernetes-pods"}`,
				`{job="kubernetes-apiservers"}`,
				`{job="kubernetes-service-endpoints"}`,
				`{job="kubernetes-cadvisor"}`,
				`{job="banzaicloud-pushgateway"}`,
				`{job="node_exporter"}`,
			},
		},
		RelabelConfigs: []*promconfig.RelabelConfig{
			{
				SourceLabels: model.LabelNames{
					model.LabelName("__address__"),
				},
				Action:      "replace",
				Regex:       promconfig.MustNewRegexp(`(.+):(?:\d+)`),
				Replacement: "${1}",
				TargetLabel: "cluster",
			},
		},
		HTTPClientConfig: promCommon.HTTPClientConfig{},
		ServiceDiscoveryConfig: promconfig.ServiceDiscoveryConfig{
			StaticConfigs: []*promconfig.TargetgroupGroup{
				{
					Targets: []model.LabelSet{
						{
							model.AddressLabel: model.LabelValue(params.endpoint),
						},
					},
					Labels: model.LabelSet{"cluster_name": model.LabelValue(params.clusterName)},
				},
			},
		},
	}
	if params.basicAuthConfig != nil {
		scrapeConfig.HTTPClientConfig.BasicAuth = &promCommon.BasicAuth{
			Username:     params.basicAuthConfig.username,
			PasswordFile: filepath.Join(s.certMountPath, params.basicAuthConfig.passwordFile),
		}
		if params.tlsConfig == nil || params.tlsConfig.certFileName == "" {
			scrapeConfig.HTTPClientConfig.TLSConfig.InsecureSkipVerify = true
		}
	}
	if params.tlsConfig != nil && params.tlsConfig.caCertFileName != "" {
		scrapeConfig.HTTPClientConfig.TLSConfig = promCommon.TLSConfig{
			CAFile: filepath.Join(s.certMountPath, params.tlsConfig.caCertFileName),
		}
		if params.tlsConfig.certFileName != "" && params.tlsConfig.keyFileName != "" {
			scrapeConfig.HTTPClientConfig.TLSConfig.CertFile = filepath.Join(s.certMountPath, params.tlsConfig.certFileName)
			scrapeConfig.HTTPClientConfig.TLSConfig.KeyFile = filepath.Join(s.certMountPath, params.tlsConfig.keyFileName)
		}
	}

	return scrapeConfig
}
