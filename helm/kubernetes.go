package helm

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"

	"fmt"
	"github.com/banzaicloud/pipeline/cloud"
	"github.com/banzaicloud/pipeline/conf"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var log = conf.Logger()
var tillerTunnel *kube.Tunnel

func getHelmClient(kubeConfigPath string) (*helm.Client, error) {
	var config *rest.Config
	var err error
	if kubeConfigPath != "" {
		log.Infoln("Create Kubernetes config from file: ", kubeConfigPath)
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	} else {
		log.Infoln("Use K8S InCluster Config.")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("create kubernetes config failed: %v", err)
	}
	log.Debugln("Create kubernetes Client.")
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("Could not create kubernetes client from config.")
		return nil, fmt.Errorf("create kubernetes client failed: %v", err)
	}
	log.Debugln("Create kubernetes Tunnel.")
	tillerTunnel, err := portforwarder.New("kube-system", client, config)
	if err != nil {
		return nil, fmt.Errorf("create tunnel failed: %v", err)
	}
	log.Debugf("Created kubernetes tunnel on address: localhost:%d .", tillerTunnel.Local)
	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	hclient := helm.NewClient(helm.Host(tillerTunnelAddress))
	return hclient, nil
}

func CheckDeploymentState(cluster *cluster.Cluster, releaseName string) (string, error) {
	var (
		config *rest.Config
		err    error
	)
	kubeConfig, err := cloud.GetConfig(cluster, "")
	if err != nil {
		return "", err
	}
	if kubeConfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		log.Infoln("Use K8S InCluster Config.")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return "", fmt.Errorf("K8S Connection Failed: %v", err)
	}
	client := kubernetes.NewForConfigOrDie(config)
	filter := fmt.Sprintf("release=%s", releaseName)

	state := v1.PodRunning
	podList, err := client.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: filter})
	if err != nil && podList != nil {
		return "", fmt.Errorf("PoD list failed: %v", err)
	}
	for _, pod := range podList.Items {
		log.Debugf("PodStatus: %s", pod.Status.Phase)
		if pod.Status.Phase == v1.PodRunning {
			continue
		} else {
			state = pod.Status.Phase
			break
		}
	}

	return string(state), nil
}

func tearDown() {
	log.Debug("There is no Tunnel to close.")
	if tillerTunnel != nil {
		log.Debug("Closing Tunnel.")
		tillerTunnel.Close()
	}
}
