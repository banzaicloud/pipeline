package api

import (
	"fmt"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
)

// List service public endpoints
func ListEndpoints(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "ListEndpoints"})
	const traefik = "traefik"
	var endpointList []*htype.EndpointItem
	var endpointURLs []*htype.EndPointURLs

	kubeConfig, ok := GetK8sConfig(c)
	if ok != true {
		return
	}

	client, err := helm.GetK8sConnection(kubeConfig)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	serviceList, err := client.CoreV1().Services("").List(meta_v1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, htype.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}
	for _, service := range serviceList.Items {
		log.Debugf("Service: %#v", service.Status)
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			//TODO we should avoid differences on kubernetes level
			var publicIp string
			if service.Status.LoadBalancer.Ingress[0].Hostname != "" {
				publicIp = service.Status.LoadBalancer.Ingress[0].Hostname
			} else {
				publicIp = service.Status.LoadBalancer.Ingress[0].IP
			}
			if strings.Contains(service.Spec.Selector["app"], traefik) {
				ingressList, err := client.ExtensionsV1beta1().Ingresses("").List(meta_v1.ListOptions{})
				if err != nil {
					log.Errorf("Error listing ingresses: %s", err)
					c.JSON(http.StatusInternalServerError, htype.ErrorResponse{
						Code:    http.StatusInternalServerError,
						Message: fmt.Sprintf("List kubernetes ingresses failed: %+v", err),
					})
					return
				}
				for _, ingress := range ingressList.Items {
					log.Debugf("Inspecting ingress: %s", ingress.Name)
					if ingress.Annotations["kubernetes.io/ingress.class"] == traefik {
						path := ingress.Spec.Rules[0].HTTP.Paths[0].Path
						endpointURLs = append(endpointURLs, &htype.EndPointURLs{
							ServiceName: strings.TrimPrefix(path, "/"),
							URL:         fmt.Sprint("http://", publicIp, path),
						})
					}
				}
			}
			endpointList = append(endpointList, &htype.EndpointItem{
				Name:         service.Name,
				Host:         publicIp,
				EndPointURLs: endpointURLs,
			})
		}
	}

	c.JSON(http.StatusOK, htype.EndpointResponse{
		Endpoints: endpointList,
	})
}
