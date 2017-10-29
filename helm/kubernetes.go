package helm

import (
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/rest"

    "k8s.io/helm/pkg/kube"
    "k8s.io/helm/pkg/helm"
    "k8s.io/helm/pkg/helm/portforwarder"

    "github.com/banzaicloud/pipeline/conf"

    "fmt"
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

func tearDown() {
    log.Debug("There is no Tunnel to close.")
    if tillerTunnel != nil {
        log.Debug("Closing Tunnel.")
        tillerTunnel.Close()
    }
}
