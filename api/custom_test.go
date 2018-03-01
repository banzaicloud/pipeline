package api

import (
	"fmt"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"testing"
)

func TestIngressEndpointUrls(t *testing.T) {
	// given
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
	}

	// when
	actualEndpoints := getIngressEndpoints(loadBalancerPublicHost, ingress)

	// then
	if !reflect.DeepEqual(expectedEndpoints, actualEndpoints) {
		t.Errorf("Expected: %v, got: %v", expectedEndpoints, actualEndpoints)
	}
}
