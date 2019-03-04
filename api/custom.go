// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/banzaicloud/pipeline/helm"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// ListEndpoints lists service public endpoints
func ListEndpoints(c *gin.Context) {

	releaseName := c.Query("releaseName")
	log.Infof("Filtering for helm release name: %s", releaseName)
	log.Info("if empty(\"\") all the endpoints will be returned")

	kubeConfig, ok := GetK8sConfig(c)
	if ok != true {
		return
	}
	if releaseName != "" {
		status, err := helm.GetDeploymentStatus(releaseName, kubeConfig)
		if err != nil {
			c.JSON(int(status), pkgCommon.ErrorResponse{
				Code:    int(status),
				Message: err.Error(),
				Error:   err.Error(),
			})
			return
		}
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error getting k8s connection: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting k8s connection",
			Error:   err.Error(),
		})
		return
	}

	serviceList, err := client.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error listing services: %s", err.Error())
		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Error during listing services",
			Error:   err.Error(),
		})
		return
	}

	ingressList, err := client.ExtensionsV1beta1().Ingresses(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error listing ingresses: %s", err)
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("List kubernetes ingresses failed: %+v", err),
		})
		return
	}
	ownLoadBalancer := deploymentHasOwnLoadBalancer(serviceList, releaseName)

	if !ownLoadBalancer && releaseName != "" {
		ingressList = filterIngressList(ingressList, releaseName)

		if ingressList.Items == nil {
			message := fmt.Sprintf("Releasename: %s does not have public endpoint exposed via ingress", releaseName)
			c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
				Code:    http.StatusNotFound,
				Message: message,
				Error:   message,
			})
			return
		}
	}

	if releaseName != "" {
		if pendingLoadBalancer(serviceList) {
			c.JSON(http.StatusAccepted, pkgHelm.StatusResponse{
				Status:  http.StatusAccepted,
				Message: "There is at least one LoadBalancer type service with Pending state",
			})
			return
		}
	}

	if ownLoadBalancer && releaseName != "" {
		serviceList = filterServiceList(serviceList, releaseName)
	}

	endpointList := getLoadBalancersWithIngressPaths(serviceList, ingressList)

	c.JSON(http.StatusOK, pkgHelm.EndpointResponse{
		Endpoints: endpointList,
	})
}

func deploymentHasOwnLoadBalancer(serviceList *v1.ServiceList, releaseName string) bool {
	if releaseName == "" {
		return false
	}
	for _, service := range serviceList.Items {
		if strings.Contains(service.Name, releaseName) && string(service.Spec.Type) == string(core.ServiceTypeLoadBalancer) {
			return true
		}
	}
	return false
}

func filterServiceList(serviceList *v1.ServiceList, releaseName string) *v1.ServiceList {
	var filteredService v1.ServiceList
	for _, service := range serviceList.Items {
		if strings.Contains(service.Name, releaseName) {
			filteredService.Items = append(filteredService.Items, service)
		}
	}
	return &filteredService
}

func filterIngressList(ingressList *v1beta1.IngressList, releaseName string) *v1beta1.IngressList {
	var filteredIngresses v1beta1.IngressList
	for _, ingress := range ingressList.Items {
		if strings.Contains(ingress.Name, releaseName) {
			filteredIngresses.Items = append(filteredIngresses.Items, ingress)
		}
	}
	return &filteredIngresses

}

func pendingLoadBalancer(serviceList *v1.ServiceList) bool {
	log.Info("Checking loadbalancer status..")

	plb := map[string]struct{}{}

	for _, service := range serviceList.Items {
		if string(service.Spec.Type) == string(core.ServiceTypeLoadBalancer) {
			if len(service.Status.LoadBalancer.Ingress) > 0 {
				plb["false"] = struct{}{}
			} else {
				plb["true"] = struct{}{}
			}
		}
	}
	_, contains := plb["true"]
	return contains
}

func getLoadBalancersWithIngressPaths(serviceList *v1.ServiceList, ingressList *v1beta1.IngressList) []*pkgHelm.EndpointItem {
	var endpointList []*pkgHelm.EndpointItem
	const traefik = "traefik"

	for _, service := range serviceList.Items {
		var endpointURLs []*pkgHelm.EndPointURLs
		log := log.WithFields(logrus.Fields{"serviceName": service.Name, "serviceNamespace": service.Namespace})
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			//TODO we should avoid differences on kubernetes level
			var publicEndpoint string
			if service.Status.LoadBalancer.Ingress[0].Hostname != "" {
				publicEndpoint = service.Status.LoadBalancer.Ingress[0].Hostname
			} else {
				publicEndpoint = service.Status.LoadBalancer.Ingress[0].IP
			}
			log.Debugf("publicEndpoint: %s", publicEndpoint)
			ports := make(map[string]int32)
			if len(service.Spec.Ports) > 0 {
				for _, port := range service.Spec.Ports {
					ports[port.Name] = port.Port
				}
			}
			if strings.Contains(service.Spec.Selector["app"], traefik) {
				for _, ingress := range ingressList.Items {
					log.Debugf("Inspecting ingress: %s", ingress.Name)
					if ingress.Annotations["kubernetes.io/ingress.class"] == traefik {
						if len(ingress.Spec.Rules) > 0 {
							for _, rule := range ingress.Spec.Rules {
								if rule.Host != "" {
									publicEndpoint = rule.Host
									log.Debugf("new publicEndpoint: %s", publicEndpoint)
									break
								}
							}
						}
						endpoints := getIngressEndpoints(publicEndpoint, &ingress, serviceList)
						for i := 0; i < len(endpoints); i++ {
							endpointURLs = append(endpointURLs, &(endpoints[i]))
						}
					}
				}
			}
			endpointList = append(endpointList, &pkgHelm.EndpointItem{
				Name:         service.Name,
				Host:         publicEndpoint,
				Ports:        ports,
				EndPointURLs: endpointURLs,
			})
		}
	}
	return endpointList
}

// getIngressEndpoints iterates through all the rules->paths defined in the given Ingress object
// and returns a collection of EndPointURLs form it.
// The EndPointURLs struct is constructed as:
//             EndPointURLs {
//                     ServiceName: {path from ingress rule}
//                     URL: http://{loadBalancerPublicHost}/{path from ingress rule}
//                     HelmReleaseName: {helm generated release name}
//             }
func getIngressEndpoints(loadBalancerPublicHost string, ingress *v1beta1.Ingress, serviceList *v1.ServiceList) []pkgHelm.EndPointURLs {
	var endpointUrls []pkgHelm.EndPointURLs

	for _, ingressRule := range ingress.Spec.Rules {
		for _, ingressPath := range ingressRule.HTTP.Paths {
			path := ingressPath.Path

			if !strings.HasSuffix(path, "/") {
				path += "/"
			}
			endpointUrls = append(endpointUrls,
				pkgHelm.EndPointURLs{
					Path:        fmt.Sprintf("/%s", strings.Trim(path, "/")),
					URL:         fmt.Sprint("http://", loadBalancerPublicHost, path),
					ReleaseName: getIngressReleaseName(ingressPath.Backend, serviceList),
				})
		}
	}

	return endpointUrls
}

// getIngressReleaseName returns the release name generated by the helm for the specific ingress
func getIngressReleaseName(backend v1beta1.IngressBackend, serviceList *v1.ServiceList) string {
	serviceName := backend.ServiceName
	for _, service := range serviceList.Items {
		if service.Name == serviceName {
			return service.Labels["release"]
		}
	}
	return "No release name for this ingress."
}

//GetClusterNodes Get node information
func GetClusterNodes(c *gin.Context) {

	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error getting k8s connection: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting k8s connection",
			Error:   err.Error(),
		})

		return
	}

	response, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error listing nodes: %s", err.Error())
		c.JSON(http.StatusNotFound, pkgCommon.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Error during listing nodes",
			Error:   err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, response)

}
