package cluster

import (
	"context"
	"fmt"

	pipConfig "github.com/banzaicloud/pipeline/config"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//PrometheusCfg describes Prometheus config
type PrometheusCfg struct {
	Endpoint     string
	Name         string
	CaFilePath   string
	CertFilePath string
	KeyFile      string
}

//UpdatePrometheusConfig updates the Prometheus configuration
func UpdatePrometheusConfig() error {
	//TODO configsets
	if !viper.GetBool("monitor.enabled") {
		log.Warn("Update monitoring configuration is disabled")
		return nil
	}

	//TODO move to configuration or sg like this
	prometheusConfigMap := "prometheus-server"
	releaseName := viper.GetString("monitor.release")
	log.Debugf("Prometheus relelase name: %s", releaseName)
	log.Debugf("Prometheus Config map  name: %s", prometheusConfigMap)
	prometheusConfigMapName := releaseName + "-" + prometheusConfigMap
	log.Debugf("Prometheus Config map full name: %s", prometheusConfigMapName)

	prefix := pipConfig.GetStateStorePath("")
	configMapPath := viper.GetString("monitor.mountpath")

	// TODO: move these to a struct and create them only once upon application init
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterManager := NewManager(intCluster.NewClusters(pipConfig.DB()), secretValidator, log, errorHandler)

	clusters, err := clusterManager.GetAllClusters(context.Background())
	if err != nil {
		return emperror.Wrap(err, "could not get all clusters")
	}

	var prometheusConfig []PrometheusCfg
	//Gathering information about clusters
	for _, cluster := range clusters {
		kubeEndpoint, err := cluster.GetAPIEndpoint()
		if err != nil {
			log.Errorf("Cluster endpoint is not available for cluster: %s, err: %s", cluster.GetName(), err)
			continue
		}

		log.Debugf("Cluster Endpoint IP: %s", kubeEndpoint)
		basePath := prefix + "/" + cluster.GetName()

		cfgElement := PrometheusCfg{
			Endpoint: kubeEndpoint,
			Name:     cluster.GetName(),
		}
		if configMapPath == "" {
			cfgElement.CaFilePath = basePath + "/certificate-authority-data.pem"
			cfgElement.CertFilePath = basePath + "/client-certificate-data.pem"
			cfgElement.KeyFile = basePath + "/client-key-data.pem"
		} else {
			cfgElement.CaFilePath = configMapPath + "/" + cluster.GetName() + "_certificate-authority-data.pem"
			cfgElement.CertFilePath = configMapPath + "/" + cluster.GetName() + "_client-certificate-data.pem"
			cfgElement.KeyFile = configMapPath + "/" + cluster.GetName() + "_client-key-data.pem"
		}

		prometheusConfig = append(prometheusConfig, cfgElement)

	}
	prometheusConfigRaw := GenerateConfig(prometheusConfig)

	log.Info("Kubernetes in-cluster configuration.")
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "can't use kubernetes in-cluster config")
	}
	client := kubernetes.NewForConfigOrDie(config)

	//TODO configurable namespace and service
	configmap, err := client.CoreV1().ConfigMaps("default").Get(prometheusConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting kubernetes confgimap failed: %s", err)
	}
	log.Info("Updating configmap")
	configmap.Data["prometheus.yml"] = string(prometheusConfigRaw)
	client.CoreV1().ConfigMaps("default").Update(configmap)
	log.Info("Update configmap finished")

	return nil
}
