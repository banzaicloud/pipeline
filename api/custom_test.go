package api

import (
	"fmt"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"testing"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngressEndpointUrls(t *testing.T) {
	//given
	ingress := &v1beta1.Ingress{
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{},
			TLS:     []v1beta1.IngressTLS{},
			Rules: []v1beta1.IngressRule{
				{
					Host: "",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/svc1_path1",
									Backend: v1beta1.IngressBackend{
										ServiceName: "service1",
										ServicePort: intstr.FromInt(1000),
									},
								},
								{
									Path: "/svc1_path2",
									Backend: v1beta1.IngressBackend{
										ServiceName: "service1",
										ServicePort: intstr.FromInt(1000),
									},
								},
							},
						},
					},
				},
				{
					Host: "",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/svc1_ui",
									Backend: v1beta1.IngressBackend{
										ServiceName: "service1",
										ServicePort: intstr.FromInt(1000),
									},
								},
							},
						},
					},
				},
				{
					Host: "",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1beta1.IngressBackend{
										ServiceName: "service1",
										ServicePort: intstr.FromInt(1000),
									},
								},
							},
						},
					},
				},
			},
		},
		Status: v1beta1.IngressStatus{},
	}

	loadBalancerPublicHost := "lb.url.com"

	expectedEndpoints := []htype.EndPointURLs{
		{
			ServiceName: "svc1_path1",
			URL:         fmt.Sprint("http://", loadBalancerPublicHost, "/svc1_path1/"),
		},
		{
			ServiceName: "svc1_path2",
			URL:         fmt.Sprint("http://", loadBalancerPublicHost, "/svc1_path2/"),
		},
		{
			ServiceName: "svc1_ui",
			URL:         fmt.Sprint("http://", loadBalancerPublicHost, "/svc1_ui/"),
		},
		{
			ServiceName: "",
			URL:         fmt.Sprint("http://", loadBalancerPublicHost, "/"),
		},
	}

	// when
	actualEndpoints := getIngressEndpoints(loadBalancerPublicHost, ingress)

	// then
	if !reflect.DeepEqual(expectedEndpoints, actualEndpoints) {
		t.Errorf("Expected: %v, got: %v", expectedEndpoints, actualEndpoints)
	}
}
const(
	dummyLoadBalancer = "dummy.loadbalancer"
	dummyLoadBalancer2 = "dummy.loadbalancer2"
	dummyIP = "192.168.0.1"
	traefik = "traefik"
)

var (
	serviceListWithoutLoadBalancer = &v1.ServiceList{
		Items: []v1.Service{{
			ObjectMeta: v12.ObjectMeta{
				Name: "withoutLoadBalancer",
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: nil,
				},
			},
		},
		},
	}
	serviceListWithHostName = &v1.ServiceList{
		Items: []v1.Service{{
			ObjectMeta: v12.ObjectMeta{
				Name: "serviceListWithHostName",
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{{
						Hostname: dummyLoadBalancer,
					},
					},
				},
			},
		},
		},
	}
	serviceListWithIP = &v1.ServiceList{
		Items: []v1.Service{{
			ObjectMeta: v12.ObjectMeta{
				Name: "serviceListWithIP",
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{{
						IP: dummyIP,
					},
					},
				},
			},
		},
		},
	}
	serviceListWithMultipleLoadBalancer = &v1.ServiceList{
		Items: []v1.Service{{
			ObjectMeta: v12.ObjectMeta{
				Name: "loadBalancerWithIngress",
			},
			Spec: v1.ServiceSpec{
				Selector: map[string]string{"app": traefik,},
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{{
						Hostname: dummyLoadBalancer,
					},
					},
				},
			},
		}, {
			ObjectMeta: v12.ObjectMeta{
				Name: "loadBalancerWithoutIngress",
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{{
						Hostname: dummyLoadBalancer2,
					},
					},
				},
			},
		},
		},
	}
)
var (
	ingressListWithMultipleLoadBalancer = &v1beta1.IngressList{
		Items: []v1beta1.Ingress{{
			ObjectMeta: v12.ObjectMeta{
				Name:        "test-ingress1",
				Annotations: map[string]string{"kubernetes.io/ingress.class": traefik,},
			},
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/svc1_path1",
										Backend: v1beta1.IngressBackend{
											ServiceName: "service1",
											ServicePort: intstr.FromInt(1000),
										},
									},
									{
										Path: "/svc1_path2",
										Backend: v1beta1.IngressBackend{
											ServiceName: "service1",
											ServicePort: intstr.FromInt(1000),
										},
									},
								},
							},
						},
					},
				},
			},
		}},
	}
)
var (
	expectedEndpointListWithHostName = []*htype.EndpointItem{{
		Name:         "serviceListWithHostName",
		Host:         dummyLoadBalancer,
		EndPointURLs: nil,
	},}
	expectedEndpointListWithIP = []*htype.EndpointItem{{
		Name:         "serviceListWithIP",
		Host:         dummyIP,
		EndPointURLs: nil,
	},}
	expectedEndpointWithMultipleLoadBalancer = []*htype.EndpointItem{{
		Name: "loadBalancerWithIngress",
		Host: "dummy.loadbalancer",
		EndPointURLs: []*htype.EndPointURLs{
			{
				ServiceName: "svc1_path1",
				URL:         fmt.Sprint("http://", dummyLoadBalancer, "/svc1_path1/"),
			},
			{
				ServiceName: "svc1_path2",
				URL:         fmt.Sprint("http://", dummyLoadBalancer, "/svc1_path2/"),
			},
		},
	}, {
		Name:         "loadBalancerWithoutIngress",
		Host:         dummyLoadBalancer2,
		EndPointURLs: nil,
	},}
)

func TestProba(t *testing.T) {
	cases := []struct {
		testName             string
		inputServiceList     *v1.ServiceList
		inputIngressList     *v1beta1.IngressList
		expectedEndPointList []*htype.EndpointItem
	}{
		{testName: "withoutLoadBalancer", inputServiceList: serviceListWithoutLoadBalancer, inputIngressList: nil, expectedEndPointList: nil},
		{testName: "serviceWithHostName", inputServiceList: serviceListWithHostName, inputIngressList: nil, expectedEndPointList: expectedEndpointListWithHostName},
		{testName: "serviceWithIP", inputServiceList: serviceListWithIP, inputIngressList: nil, expectedEndPointList: expectedEndpointListWithIP},
		{testName: "serviceWithMultipleLoadBalancer", inputServiceList: serviceListWithMultipleLoadBalancer,
			inputIngressList: ingressListWithMultipleLoadBalancer, expectedEndPointList: expectedEndpointWithMultipleLoadBalancer},
	}
	for _, tc := range cases {
		t.Run(tc.testName, func(t *testing.T) {
			endpointList := getLoadBalancersWithIngressPaths(tc.inputServiceList, tc.inputIngressList)

			if !reflect.DeepEqual(tc.expectedEndPointList, endpointList) {
				t.Errorf("Expected: %v, got: %v", tc.expectedEndPointList, endpointList)
			}
		})
	}
}
