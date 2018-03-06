package helm

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"

	"fmt"
	"github.com/banzaicloud/pipeline/config"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/banzai-types/constants"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var tillerTunnel *kube.Tunnel

func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"action": "Helm"})
}

//GetK8sConnection creates a new Kubernetes client
func GetK8sConnection(kubeConfig *[]byte) (*kubernetes.Clientset, error) {
	config, err := GetK8sClientConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes config failed: %v", err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes connection failed: %v", err)
	}
	return client, nil
}

func GetK8sInClusterConnection() (*kubernetes.Clientset, error) {
	log.Info("Kubernetes in-cluster configuration.")
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "can't use kubernetes in-cluster config")
	}
	client := kubernetes.NewForConfigOrDie(config)
	return client, nil
}

//GetK8sClientConfig creates a Kubernetes client config
func GetK8sClientConfig(kubeConfig *[]byte) (*rest.Config, error) {
	var config *rest.Config
	var err error
	if kubeConfig != nil {
		apiconfig, _ := clientcmd.Load(*kubeConfig)
		clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
		config, err = clientConfig.ClientConfig()
		log.Debug("Use K8S RemoteCluster Config: ", config.ServerName)
	} else {
		log.Info("Use K8S InCluster Config.")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("create kubernetes config failed: %v", err)
	}
	return config, nil
}

//GetHelmClient establishes Tunnel for Helm client TODO check client and config if both needed
func GetHelmClient(kubeConfig *[]byte) (*helm.Client, error) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagKubernetes})
	log.Debug("Create kubernetes Client.")
	config, err := GetK8sClientConfig(kubeConfig)
	client, err := GetK8sConnection(kubeConfig)
	if err != nil {
		log.Debug("Could not create kubernetes client from config.")
		return nil, fmt.Errorf("create kubernetes client failed: %v", err)
	}
	log.Debug("Create kubernetes Tunnel")
	tillerTunnel, err = portforwarder.New("kube-system", client, config)
	if err != nil {
		return nil, fmt.Errorf("create tunnel failed: %v", err)
	}
	log.Debug("Created kubernetes tunnel on address: localhost:", tillerTunnel.Local)
	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	hclient := helm.NewClient(helm.Host(tillerTunnelAddress))
	return hclient, nil
}

//CheckDeploymentState checks the state of Helm deployment
func CheckDeploymentState(kubeConfig *[]byte, releaseName string) (string, error) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagKubernetes})
	client, err := GetK8sConnection(kubeConfig)
	filter := fmt.Sprintf("release=%s", releaseName)

	state := v1.PodRunning
	podList, err := client.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: filter})
	if err != nil && podList != nil {
		return "", fmt.Errorf("PoD list failed: %v", err)
	}
	for _, pod := range podList.Items {
		log.Debug("PodStatus:", pod.Status.Phase)
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
	if tillerTunnel != nil {
		log.Debug("Closing Tunnel.")
		tillerTunnel.Close()
	}
}
