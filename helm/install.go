package helm

import (
	"k8s.io/helm/cmd/helm/installer"
	"fmt"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/kube"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/banzaicloud/banzai-types/utils"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/components"
	"net/http"
	"github.com/banzaicloud/banzai-types/components/helm"
)

// Install uses Kubernetes client to install Tiller.
func Install(helmInstall *helm.Install) *components.BanzaiResponse {
	opts := installer.Options{
		Namespace:      helmInstall.Namespace,
		ServiceAccount: helmInstall.ServiceAccount,
	}
	_, kubeClient, err := getKubeClient(helmInstall.KubeContext)
	if err != nil {
		utils.LogErrorf(constants.TagHelmInstall, "could not get kubernetes client: %s", err)
		return &components.BanzaiResponse{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("could not get kubernetes client: %s", err),
		}
	}
	if err := installer.Install(kubeClient, &opts); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			utils.LogErrorf(constants.TagHelmInstall, "error installing: %s", err)
			return &components.BanzaiResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("error installing: %s", err),
			}
		}
		if helmInstall.Upgrade {
			if err := installer.Upgrade(kubeClient, &opts); err != nil {
				utils.LogErrorf(constants.TagHelmInstall, "error when upgrading: %s", err)
				return &components.BanzaiResponse{
					StatusCode: http.StatusInternalServerError,
					Message:    fmt.Sprintf("error when upgrading: %s", err),
				}
			}
			utils.LogInfo(constants.TagHelmInstall, "Tiller (the Helm server-side component) has been upgraded to the current version.")
		} else {
			utils.LogInfo(constants.TagHelmInstall, "Warning: Tiller is already installed in the cluster.")
		}
	} else {
		utils.LogInfo(constants.TagHelmInstall, "Tiller (the Helm server-side component) has been installed into your Kubernetes Cluster.")
	}
	utils.LogInfo(constants.TagHelmInstall, "Helm install finished")
	return &components.BanzaiResponse{
		StatusCode: http.StatusOK,
	}
}

// getKubeClient creates a Kubernetes config and client for a given kubeconfig context.
func getKubeClient(context string) (*rest.Config, kubernetes.Interface, error) {
	config, err := configForContext(context)
	if err != nil {
		return nil, nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	return config, client, nil
}

// configForContext creates a Kubernetes REST client configuration for a given kubeconfig context.
func configForContext(context string) (*rest.Config, error) {
	config, err := kube.GetConfig(context).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes config for context %q: %s", context, err)
	}
	return config, nil
}
