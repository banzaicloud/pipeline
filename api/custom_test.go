package api

import (
	"fmt"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"testing"
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
										ServiceName: "serviceForIngress",
										ServicePort: intstr.FromInt(1000),
									},
								},
								{
									Path: "/svc1_path2",
									Backend: v1beta1.IngressBackend{
										ServiceName: "serviceForIngress",
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
										ServiceName: "serviceForIngress",
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
										ServiceName: "serviceForIngress",
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
			Path:        "/svc1_path1",
			URL:         fmt.Sprint("http://", loadBalancerPublicHost, "/svc1_path1/"),
			ReleaseName: dummyReleaseName,
		},
		{
			Path:        "/svc1_path2",
			URL:         fmt.Sprint("http://", loadBalancerPublicHost, "/svc1_path2/"),
			ReleaseName: dummyReleaseName,
		},
		{
			Path:        "/svc1_ui",
			URL:         fmt.Sprint("http://", loadBalancerPublicHost, "/svc1_ui/"),
			ReleaseName: dummyReleaseName,
		},
		{
			Path:        "/",
			URL:         fmt.Sprint("http://", loadBalancerPublicHost, "/"),
			ReleaseName: dummyReleaseName,
		},
	}

	// when
	actualEndpoints := getIngressEndpoints(loadBalancerPublicHost, ingress, serviceForIngress)

	// then
	if !reflect.DeepEqual(expectedEndpoints, actualEndpoints) {
		t.Errorf("Expected: %v, got: %v", expectedEndpoints, actualEndpoints)
	}
}

const (
	dummyLoadBalancer  = "dummy.loadbalancer"
	dummyLoadBalancer2 = "dummy.loadbalancer2"
	dummyIP            = "192.168.0.1"
	traefik            = "traefik"
	dummyReleaseName   = "vetoed-ibis"
)

var (
	serviceForIngress = &v1.ServiceList{
		Items: []v1.Service{{
			ObjectMeta: v12.ObjectMeta{
				Name:   "serviceForIngress",
				Labels: map[string]string{"release": dummyReleaseName},
			},
		},
		},
	}
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
				Selector: map[string]string{"app": traefik},
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
			{
				ObjectMeta: v12.ObjectMeta{
					Name:   "serviceForIngress",
					Labels: map[string]string{"release": dummyReleaseName},
				},
			},
		},
	}
	serviceListWithPort = &v1.ServiceList{
		Items: []v1.Service{{
			ObjectMeta: v12.ObjectMeta{
				Name: "loadBalancerWithPort",
			},
			Spec: v1.ServiceSpec{
				Ports: []v1.ServicePort{{
					Name: "UI",
					Port: 80,
				}, {
					Name: "API",
					Port: 3000,
				},
				},
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
)
var (
	ingressListWithMultipleLoadBalancer = &v1beta1.IngressList{
		Items: []v1beta1.Ingress{{
			ObjectMeta: v12.ObjectMeta{
				Name:        "test-ingress1",
				Annotations: map[string]string{"kubernetes.io/ingress.class": traefik},
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
											ServiceName: "serviceForIngress",
											ServicePort: intstr.FromInt(1000),
										},
									},
									{
										Path: "/svc1_path2",
										Backend: v1beta1.IngressBackend{
											ServiceName: "serviceForIngress",
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
		Ports:        make(map[string]int32),
		EndPointURLs: nil,
	}}
	expectedEndpointListWithIP = []*htype.EndpointItem{{
		Name:         "serviceListWithIP",
		Host:         dummyIP,
		Ports:        make(map[string]int32),
		EndPointURLs: nil,
	}}
	expectedEndpointWithMultipleLoadBalancer = []*htype.EndpointItem{{
		Name:  "loadBalancerWithIngress",
		Host:  "dummy.loadbalancer",
		Ports: make(map[string]int32),
		EndPointURLs: []*htype.EndPointURLs{
			{
				Path:        "/svc1_path1",
				URL:         fmt.Sprint("http://", dummyLoadBalancer, "/svc1_path1/"),
				ReleaseName: dummyReleaseName,
			},
			{
				Path:        "/svc1_path2",
				URL:         fmt.Sprint("http://", dummyLoadBalancer, "/svc1_path2/"),
				ReleaseName: dummyReleaseName,
			},
		},
	}, {
		Name:         "loadBalancerWithoutIngress",
		Host:         dummyLoadBalancer2,
		Ports:        make(map[string]int32),
		EndPointURLs: nil,
	}}
	expectedEndpointListWithPort = []*htype.EndpointItem{{
		Name: "loadBalancerWithPort",
		Host: "dummy.loadbalancer",
		Ports: map[string]int32{
			"UI":  80,
			"API": 3000,
		},
		EndPointURLs: nil,
	},
	}
)

func TestLoadBalancersWithIngressPaths(t *testing.T) {
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
		{testName: "serviceWithPorts", inputServiceList: serviceListWithPort, inputIngressList: nil, expectedEndPointList: expectedEndpointListWithPort},
	}
	for _, tc := range cases {
		t.Run(tc.testName, func(t *testing.T) {
			endpointList := getLoadBalancersWithIngressPaths(tc.inputServiceList, tc.inputIngressList)

			if !reflect.DeepEqual(tc.expectedEndPointList, endpointList) {
				t.Errorf("Expected: %#v, got: %#v", tc.expectedEndPointList, endpointList)
			}
		})
	}
}

var (
	serviceListWithPendingLoadBalancer = &v1.ServiceList{
		Items: []v1.Service{{
			ObjectMeta: v12.ObjectMeta{
				Name: "serviceListWithPendingLoadBalancer",
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{},
				},
			},
		},
		}

	serviceListReadyLoadBalancer = &v1.ServiceList{
		Items: []v1.Service{{
			ObjectMeta: v12.ObjectMeta{
				Name: "serviceListWithReadyLoadBalancer",
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

	serviceListWithPendingReadyLoadBalancer = &v1.ServiceList{
		Items: []v1.Service{{
			ObjectMeta: v12.ObjectMeta{
				Name: "serviceWithPendingLoadBalancer",
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{},
			},
		},
			{
				ObjectMeta: v12.ObjectMeta{
					Name: "serviceWithReadyLoadBalancer",
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
)

func TestPendingLoadBalancer(t *testing.T) {
	cases := []struct {
		testName					string
		inputServiceList	*v1.ServiceList
		expectedResult		bool
	}{
		{testName: "PendingLoadBalancer", inputServiceList: serviceListWithPendingLoadBalancer, expectedResult: true},
		{testName: "ReadyLoadBalancer", inputServiceList: serviceListReadyLoadBalancer, expectedResult: false},
		{testName: "MultipleLoadBalancer", inputServiceList: serviceListWithPendingReadyLoadBalancer, expectedResult: true},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(t *testing.T) {
			loadBalancerState := pendingLoadBalancer(tc.inputServiceList)

			if loadBalancerState != tc.expectedResult {
				t.Errorf("Expected: %#v, got: %#v", tc.expectedResult, loadBalancerState)
			}
		})
	}
}
