package monitor

import (
	"os"

	"fmt"

	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/database"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	"github.com/banzaicloud/pipeline/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type prometheusTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

//PrometheusCfg
type PrometheusCfg struct {
	Endpoint string
	Name     string
}

//UpdatePrometheusConfig updates the Prometheus configuration
func UpdatePrometheusConfig() error {

	//TODO configsets
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) <= 0 {
		banzaiUtils.LogWarn(banzaiConstants.TagPrometheus, "Non k8s Env -> UpdatePrometheusConfig skip! ")
		return nil
	}
	prometheusConfigMapName := "prometheus-server"

	releaseName := os.Getenv("KUBERNETES_RELEASE_NAME")
	if len(releaseName) > 0 {
		banzaiUtils.LogDebug(banzaiConstants.TagPrometheus, "K8s Release Name:", releaseName)
		prometheusConfigMapName = releaseName + "-" + prometheusConfigMapName
	}

	var clusters []banzaiSimpleTypes.ClusterSimple
	database.Find(&clusters)
	var prometheusConfig []PrometheusCfg
	//Gathering information about clusters
	for _, cluster := range clusters {
		banzaiUtils.LogDebug(banzaiConstants.TagPrometheus, "Cluster: ", cluster.Name)
		cloudType := cluster.Cloud
		banzaiUtils.LogDebug(banzaiConstants.TagPrometheus, "Cluster type is: ", cloudType)
		var kubeEndpoint string
		switch cloudType {
		case banzaiConstants.Amazon:
			cloudCluster, err := cloud.ReadCluster(cluster)
			if err != nil {
				banzaiUtils.LogWarn(banzaiConstants.TagPrometheus, "Cluster Parser Error: ", err.Error())
				continue
			}
			kubeEndpoint = cloudCluster.KubernetesAPI.Endpoint
		case banzaiConstants.Azure:
			clusterRep := cloud.ReadClusterAzure(&cluster)
			kubeEndpoint = clusterRep.Value.Properties.Fqdn
		}
		banzaiUtils.LogDebug(banzaiConstants.TagPrometheus, "Cluster Endpoint IP: ", kubeEndpoint)

		prometheusConfig = append(
			prometheusConfig,
			PrometheusCfg{
				Endpoint: kubeEndpoint,
				Name:     cluster.Name,
			})

	}
	prometheusConfigRaw := GenerateConfig(prometheusConfig)

	var kubeconfig = os.Getenv("KUBECONFIG")
	banzaiUtils.LogDebug(banzaiConstants.TagPrometheus, "KUBECONFIG:", kubeconfig)

	var (
		config *rest.Config
		err    error
	)
	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagPrometheus, "Use K8S InCluster Config.")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return fmt.Errorf("K8S Connection Failed: %v", err)
	}

	client := kubernetes.NewForConfigOrDie(config)
	banzaiUtils.LogDebug(banzaiConstants.TagPrometheus, "K8S Connection Successful!")

	//TODO configurable namespace and service
	configmap, err := client.CoreV1().ConfigMaps("default").Get(prometheusConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("K8S get Configmap Failed: %v", err)
	}

	banzaiUtils.LogDebug(banzaiConstants.TagPrometheus, "Actual k8sclusters.json content: ", configmap.Data["prometheus.yml"])
	banzaiUtils.LogDebug(banzaiConstants.TagPrometheus, "K8S Update prometheus-server.k8sclusters.json Configmap.")
	configmap.Data["prometheus.yml"] = string(prometheusConfigRaw)
	client.CoreV1().ConfigMaps("default").Update(configmap)
	banzaiUtils.LogInfo(banzaiConstants.TagPrometheus, "K8S prometheus-server.k8sclusters.json Configmap Updated.")

	NewConfigmap, _ := (client.CoreV1().ConfigMaps("default").Get(prometheusConfigMapName, metav1.GetOptions{}))
	banzaiUtils.LogDebug(banzaiConstants.TagPrometheus, "K8S Updated Configmap:", NewConfigmap.Data["k8sclusters.json"])
	return nil
}
