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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
)

var tillerTunnel *kube.Tunnel

func getHelmClient(kubeConfig string) (*helm.Client, error) {
	var config *rest.Config
	var err error

	//TODO Beatify this do not use string for kubeConfig
	if kubeConfig != "" {
		apiconfig, _ :=clientcmd.Load([]byte(kubeConfig))
		clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
		config, err = clientConfig.ClientConfig()
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagKubernetes, "Use K8S InCluster Config.")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("create kubernetes config failed: %v", err)
	}
	banzaiUtils.LogDebug(banzaiConstants.TagKubernetes, "Create kubernetes Client.")
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagKubernetes, "Could not create kubernetes client from config.")
		return nil, fmt.Errorf("create kubernetes client failed: %v", err)
	}
	banzaiUtils.LogDebug(banzaiConstants.TagKubernetes, "Create kubernetes Tunnel.")
	tillerTunnel, err := portforwarder.New("kube-system", client, config)
	if err != nil {
		return nil, fmt.Errorf("create tunnel failed: %v", err)
	}
	banzaiUtils.LogDebug(banzaiConstants.TagKubernetes, "Created kubernetes tunnel on address: localhost:", tillerTunnel.Local)
	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	hclient := helm.NewClient(helm.Host(tillerTunnelAddress))
	return hclient, nil
}

//CheckDeploymentState checks the state of Helm deployment
func CheckDeploymentState(cs *banzaiSimpleTypes.ClusterSimple, releaseName string) (string, error) {
	var (
		config *rest.Config
		err    error
	)

	kubeConfig, err := cloud.GetKubeConfigPath(fmt.Sprintf("./statestore/%s/", cs.Name))
	if err != nil {
		return "", err
	}
	if kubeConfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagKubernetes, "Use K8S InCluster Config.")
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
		banzaiUtils.LogDebug(banzaiConstants.TagKubernetes, "PodStatus:", pod.Status.Phase)
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
	banzaiUtils.LogDebug(banzaiConstants.TagKubernetes, "There is no Tunnel to close.")
	if tillerTunnel != nil {
		banzaiUtils.LogDebug(banzaiConstants.TagKubernetes, "Closing Tunnel.")
		tillerTunnel.Close()
	}
}
