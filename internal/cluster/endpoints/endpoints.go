// Copyright © 2019 Banzai Cloud
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

package endpoints

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"

	"github.com/banzaicloud/pipeline/internal/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

type EndpointManager struct {
	logger common.Logger
}

func NewEndpointManager(logger common.Logger) *EndpointManager {
	return &EndpointManager{
		logger: logger,
	}
}

func (m *EndpointManager) List(kubeConfig []byte, releaseName string) ([]*pkgHelm.EndpointItem, error) {
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create K8S client")
	}

	serviceList, err := client.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.WrapIf(err, "failed to list services")
	}

	ingressList, err := client.ExtensionsV1beta1().Ingresses(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.WrapIf(err, "failed to list ingress")
	}

	ownLoadBalancer := deploymentHasOwnLoadBalancer(serviceList, releaseName)

	if !ownLoadBalancer && releaseName != "" {
		ingressList = filterIngressList(ingressList, releaseName)

		if ingressList.Items == nil {
			return nil, &NotFoundError{releaseName: releaseName}
		}
	}

	if releaseName != "" {
		if pendingLoadBalancer(serviceList) {
			return nil, &PendingLoadBalancerError{}
		}

		if ownLoadBalancer {
			serviceList = filterServiceList(serviceList, releaseName)
		}
	}

	return m.getLoadBalancersWithIngressPaths(serviceList, ingressList), nil
}

func (m *EndpointManager) GetServiceUrl(kubeConfig []byte, serviceName string, namespace string) (string, error) {
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return "", errors.WrapIf(err, "failed to create K8S client")
	}

	service, err := client.CoreV1().Services(namespace).Get(serviceName, metav1.GetOptions{})
	if err != nil {
		return "", errors.WrapIf(err, "failed to list services")
	}

	return fmt.Sprintf("%s:%d", service.Spec.ClusterIP, service.Spec.Ports[0].Port), nil
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
	plb := false

	for _, service := range serviceList.Items {
		if string(service.Spec.Type) == string(core.ServiceTypeLoadBalancer) && len(service.Status.LoadBalancer.Ingress) == 0 {
			plb = true
		}
	}

	return plb
}

func (m EndpointManager) getLoadBalancersWithIngressPaths(serviceList *v1.ServiceList, ingressList *v1beta1.IngressList) []*pkgHelm.EndpointItem {
	var endpointList []*pkgHelm.EndpointItem
	const traefik = "traefik"

	for _, service := range serviceList.Items {
		var endpointURLs []*pkgHelm.EndPointURLs
		logger := m.logger.WithFields(map[string]interface{}{"serviceName": service.Name, "serviceNamespace": service.Namespace})
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			//TODO we should avoid differences on kubernetes level
			publicEndpoint := service.Status.LoadBalancer.Ingress[0].Hostname
			if publicEndpoint == "" {
				publicEndpoint = service.Status.LoadBalancer.Ingress[0].IP
			}
			logger.Debug(fmt.Sprintf("publicEndpoint: %s", publicEndpoint))
			ports := make(map[string]int32)
			for _, port := range service.Spec.Ports {
				ports[port.Name] = port.Port
			}
			if strings.Contains(service.Spec.Selector["app"], traefik) {
				for _, ingress := range ingressList.Items {
					logger.Debug(fmt.Sprintf("Inspecting ingress: %s", ingress.Name))
					for _, rule := range ingress.Spec.Rules {
						if rule.Host != "" {
							publicEndpoint = rule.Host
							logger.Debug(fmt.Sprintf("new publicEndpoint: %s", publicEndpoint))
							break
						}
					}
					endpoints := getIngressEndpoints(publicEndpoint, &ingress, serviceList, logger)
					for i := range endpoints {
						endpointURLs = append(endpointURLs, &endpoints[i])
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
func getIngressEndpoints(
	loadBalancerPublicHost string,
	ingress *v1beta1.Ingress,
	serviceList *v1.ServiceList,
	logger common.Logger,
) []pkgHelm.EndPointURLs {
	var endpointUrls []pkgHelm.EndPointURLs

	for _, ingressRule := range ingress.Spec.Rules {
		for _, ingressPath := range ingressRule.HTTP.Paths {
			path := ingressPath.Path

			if !strings.HasSuffix(path, "/") {
				path += "/"
			}

			releaseName, err := getIngressReleaseName(ingressPath.Backend.ServiceName, serviceList)
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to get release name for ingress: %s", err.Error()))
			}

			endpointUrls = append(endpointUrls,
				pkgHelm.EndPointURLs{
					Path:        fmt.Sprintf("/%s", strings.Trim(path, "/")),
					URL:         fmt.Sprint("http://", loadBalancerPublicHost, path),
					ReleaseName: releaseName,
				})
		}
	}

	return endpointUrls
}

// getIngressReleaseName returns the release name generated by the helm for the specific ingress
func getIngressReleaseName(serviceName string, serviceList *v1.ServiceList) (string, error) {
	for _, service := range serviceList.Items {
		if service.Name == serviceName {
			return pkgHelm.GetHelmReleaseName(service.Labels), nil
		}
	}
	return "", errors.NewWithDetails("no release name for this ingress", "serviceName", serviceName)
}
