package cluster

import (
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/pkg/security"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const GroupName = "security.banzaicloud.com"
const GroupVersion = "v1alpha1"

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&security.WhiteListItem{},
		&security.WhiteList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

func GetWhitelists(cc CommonCluster) (*security.WhiteList, error) {
	result := &security.WhiteList{}
	// Get kubernetes configuration
	kubeConfig, err := cc.GetK8sConfig()
	if err != nil {
		return nil, err
	}
	// Get customresource Client
	clientCfg, err := helm.GetK8sClientConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	clientCfg.ContentConfig.GroupVersion = &schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	clientCfg.APIPath = "/apis"
	clientCfg.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	clientCfg.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(clientCfg)
	if err != nil {
		return nil, err
	}
	opts := &metav1.ListOptions{}
	err = client.Get().Namespace("").Resource("whitelists").VersionedParams(opts, scheme.ParameterCodec).Do().Into(result)
	return result, err
}
