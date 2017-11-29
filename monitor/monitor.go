package monitor

import (
	"os"

	"fmt"

	"github.com/banzaicloud/pipeline/cloud"
	"github.com/banzaicloud/pipeline/conf"
	"github.com/jinzhu/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type prometheusTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

type PrometheusCfg struct {
	Endpoint string
	Name     string
}

//UpdatePrometheusConfig updates the Prometheus configuration
func UpdatePrometheusConfig(db *gorm.DB) error {
	log := conf.Logger()

	//TODO configsets
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) <= 0 {
		log.Warningln("Non k8s Env -> UpdatePrometheusConfig skip! ")
		return nil
	}
	prometheusConfigMapName := "prometheus-server"

	releaseName := os.Getenv("KUBERNETES_RELEASE_NAME")
	if len(releaseName) > 0 {
		log.Debugln("K8s Release Name:", releaseName)
		prometheusConfigMapName = releaseName + "-" + prometheusConfigMapName
	}

	var clusters []cloud.ClusterType
	db.Find(&clusters)
	var prometheusConfig []PrometheusCfg
	//Gathering information about clusters
	for _, cluster := range clusters {
		log.Debugln("Cluster: ", cluster.Name)
		cloudCluster, err := cloud.ReadCluster(cluster)
		if err != nil {
			log.Warningln("Cluster Parser Error: ", err.Error())
			continue
		}
		ip := cloudCluster.KubernetesAPI.Endpoint
		log.Debugln("Cluster Endpoint IP: ", ip)

		prometheusConfig = append(
			prometheusConfig,
			PrometheusCfg{
				Endpoint: cloudCluster.KubernetesAPI.Endpoint,
				Name:     cloudCluster.Name,
			})

	}
	prometheusConfigRaw := GenerateConfig(prometheusConfig)

	var kubeconfig = ""

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
		log.Debugln("KUBECONFIG:", kubeconfig)
	}
	var (
		config *rest.Config
		err    error
	)
	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		log.Infoln("Use K8S InCluster Config.")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return fmt.Errorf("K8S Connection Failed: %v", err)
	}

	client := kubernetes.NewForConfigOrDie(config)
	log.Debugln("K8S Connection Successful!")

	//TODO configurable namespace and service
	configmap, err := (client.CoreV1().ConfigMaps("default").Get(prometheusConfigMapName, metav1.GetOptions{}))
	if err != nil {
		return fmt.Errorf("K8S get Configmap Failed: %v", err)
	}

	log.Debugln("Actual k8sclusters.json content: ", configmap.Data["prometheus.yml"])
	log.Debugln("K8S Update prometheus-server.k8sclusters.json Configmap.")
	configmap.Data["prometheus.yml"] = string(prometheusConfigRaw)
	client.CoreV1().ConfigMaps("default").Update(configmap)
	log.Infoln("K8S prometheus-server.k8sclusters.json Configmap Updated.")

	NewConfigmap, _ := (client.CoreV1().ConfigMaps("default").Get(prometheusConfigMapName, metav1.GetOptions{}))
	log.Debugln("K8S Updated Configmap:", NewConfigmap.Data["k8sclusters.json"])
	return nil
}
