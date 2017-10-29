package monitor

import (
	"encoding/json"
	"os"

	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/banzaicloud/pipeline/cloud"
	"github.com/banzaicloud/pipeline/conf"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type prometheusTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

func UpdatePrometheusConfig(db *gorm.DB) error {
	log := conf.Logger()

	//TODO configsets
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) <= 0 {
		log.Warningln("Non k8s Env -> UpdatePrometheusConfig skip! ")
		return nil
	}

	var clusters []cloud.ClusterType
	db.Find(&clusters)
	var Targets []prometheusTarget

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
		item := prometheusTarget{
			Targets: []string{fmt.Sprintf("%s:30080", ip)},
			Labels: map[string]string{
				"cluster":      cloudCluster.Name,
				"cluster_name": cloudCluster.Name,
			},
		}
		Targets = append(Targets, item)

	}
	//Generate Prometheus Target configuration
	resJSON, errJSON := json.Marshal(Targets)
	if errJSON != nil {
		log.Warningln(errJSON)
	}
	log.Infoln("Prometheus Target Json: ", string(resJSON))

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
	configmap, err := (client.CoreV1().ConfigMaps("default").Get("prometheus-server", metav1.GetOptions{}))
	if err != nil {
		return fmt.Errorf("K8S get Configmap Failed: %v", err)
	}

	log.Debugln("Actual k8sclusters.json content: ", configmap.Data["k8sclusters.json"])
	log.Debugln("K8S Update prometheus-server.k8sclusters.json Configmap.")
	configmap.Data["k8sclusters.json"] = string(resJSON)
	client.CoreV1().ConfigMaps("default").Update(configmap)
	log.Infoln("K8S prometheus-server.k8sclusters.json Configmap Updated.")

	NewConfigmap, _ := (client.CoreV1().ConfigMaps("default").Get("prometheus-server", metav1.GetOptions{}))
	log.Debugln("K8S Updated Configmap:", NewConfigmap.Data["k8sclusters.json"])
	return nil
}
