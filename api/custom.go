package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

type EndpointResponse struct {
	Endpoints []*EndpointItem `json:"endpoints"`
}

type EndpointItem struct {
	Name         string          `json:"name"`
	Host         string          `json:"host"`
	EndPointURLs []*EndPointURLs `json:"urls"`
}

type EndPointURLs struct {
	ServiceName string `json:"servicename"`
	URL         string `json:"url"`
}

// List service public endpoints
func ListEndpoints(c *gin.Context) {
	const traefik = "traefik"
	var endpointList []*EndpointItem
	var endpointURLs []*EndPointURLs

	// --- [ Get cluster ] ---- //
	banzaiUtils.LogInfo(banzaiConstants.TagListDeployments, "Get cluster")
	cloudCluster, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	}
	banzaiUtils.LogInfo(banzaiConstants.TagListDeployments, "Getting cluster succeeded")

	cloudType := cloudCluster.Cloud

	// --- [ Get K8S Config ] --- //
	kubeConfig, err := cloud.GetK8SConfig(cloudCluster, c)
	if err != nil {
		return
	}
	apiconfig, _ := clientcmd.Load(kubeConfig)
	clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
	config, err := clientConfig.ClientConfig()
	if err != nil {
		banzaiUtils.LogErrorf(banzaiConstants.TagKubernetes, "Could not create kubernetes client from config. %+v", config)
		banzaiUtils.LogErrorf(banzaiConstants.TagKubernetes, "Error message: %+v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("create kubernetes client failed: %v", err),
		})
		return
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagKubernetes, "Could not create kubernetes client from config.")
		banzaiUtils.LogErrorf(banzaiConstants.TagKubernetes, "Error message: %+v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("create kubernetes client failed: %v", err),
		})
		return
	}
	serviceList, err := client.CoreV1().Services("").List(meta_v1.ListOptions{})
	if err != nil {
		banzaiUtils.LogErrorf(banzaiConstants.TagKubernetes, "Could not list kubernetes services, %+v", config)
		banzaiUtils.LogErrorf(banzaiConstants.TagKubernetes, "Error message: %+v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("List kubernetes services failed: %+v", err),
		})
		return
	}
	for _, service := range serviceList.Items {
		banzaiUtils.LogDebugf(banzaiConstants.TagKubernetes, "Service: %#v", service.Status)
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			var publicIp string
			switch cloudType {
			case banzaiConstants.Amazon:
				publicIp = service.Status.LoadBalancer.Ingress[0].Hostname
			case banzaiConstants.Azure:
				publicIp = service.Status.LoadBalancer.Ingress[0].IP
			}
			if strings.Contains(service.Spec.Selector["app"], traefik) {
				ingressList, err := client.ExtensionsV1beta1().Ingresses("").List(meta_v1.ListOptions{})
				if err != nil {
					banzaiUtils.LogErrorf(banzaiConstants.TagKubernetes, "Could not list kubernetes ingresses, %+v", config)
					banzaiUtils.LogErrorf(banzaiConstants.TagKubernetes, "Error message: %+v", err)
					c.JSON(http.StatusInternalServerError, ErrorResponse{
						Code:    http.StatusInternalServerError,
						Message: fmt.Sprintf("List kubernetes ingresses failed: %+v", err),
					})
					return
				}
				for _, ingress := range ingressList.Items {
					banzaiUtils.LogDebugf(banzaiConstants.TagKubernetes, "Inspecting ingress: %s", ingress.Name)
					if ingress.Annotations["kubernetes.io/ingress.class"] == traefik {
						path := ingress.Spec.Rules[0].HTTP.Paths[0].Path
						endpointURLs = append(endpointURLs, &EndPointURLs{
							ServiceName: strings.TrimPrefix(path, "/"),
							URL:         fmt.Sprint("http://", publicIp, path),
						})
					}
				}
			}
			endpointList = append(endpointList, &EndpointItem{
				Name:         service.Name,
				Host:         publicIp,
				EndPointURLs: endpointURLs,
			})
		}
	}

	c.JSON(http.StatusOK, EndpointResponse{
		Endpoints: endpointList,
	})
}
