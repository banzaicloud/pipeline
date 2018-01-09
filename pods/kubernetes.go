package pods

import (
	"flag"
	"path/filepath"
	"github.com/banzaicloud/pipeline/utils"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/banzaicloud/pipeline/cloud"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"fmt"
)

//Get Kubernetes config - running inside or outside K8S
func getConfig() *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Warnf("Cannot use service account from /var/run/secrets/kubernetes.io/serviceaccount/" +
			corev1.ServiceAccountTokenKey + ") fallback to config file")
	}

	if config == nil {
		var kubeconfig *string
		if home := utils.GetHomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "path to kubeconfig file")
		}
		log.Info("Use kubernetes config: %s", *kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}
	return config
}

//get the Pod client
func getPodClient(namespace string, config *rest.Config) v1.PodInterface {
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	podClient := clientSet.CoreV1().Pods(namespace)
	return podClient
}

//Find one cluster node, return reference
func findNode(name string, nodes []corev1.Node) *corev1.Node {
	for _, node := range nodes {
		if node.Name == name {
			return &node
		}
	}
	return nil
}

//List pods on a given node
func listPodsOnNode(ListPodsOnNode func(opts metav1.ListOptions) (*corev1.PodList, error), node corev1.Node) []corev1.Pod {
	log.Info("List the Pods on node: %s", node.Name)
	podsOnNode, err := ListPodsOnNode(metav1.ListOptions{FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Name}).String()})
	if err != nil {
		log.Errorf("Failed to list Pods on node: %s", node.Name)
		return nil
	}
	return podsOnNode.Items
}

//List the pods info as name, status, ip and node
func logPods(podGroups map[string][]corev1.Pod) {
	for _, pods := range podGroups {
		for _, pod := range pods {
			log.Info("%s\t%s\t%s\t%s", pod.Name, pod.Status.Phase, pod.Status.PodIP, pod.Spec.NodeName)
		}
	}
}

// Group the Pods that belong to the same Deployment/StatefulSet
func groupPods(pods []corev1.Pod) (podGroup map[string][]corev1.Pod) {
	podGroup = make(map[string][]corev1.Pod)
	for _, pod := range pods {
		groupName := getPodGroupName(&pod)
		if groupName != nil {
			podGroup[*groupName] = append(podGroup[*groupName], pod)
			log.Info("Pod map: %s", podGroup)
		}
	}
	return podGroup
}

//Get the generated Pod name (unless given when created)
func getPodGroupName(pod *corev1.Pod) *string {
	generatedName := pod.GenerateName
	if len(generatedName) > 0 {
		generatedName = generatedName[0: len(generatedName)-1]
		return &generatedName
	}
	return nil
}

type PodStatusResponse struct {
	Name   string          `json:"name"`
	Status corev1.PodPhase `json:"status"`
}

func GroupPodsByNode(pods []corev1.Pod) (podGroup map[string][]PodStatusResponse) {
	podGroup = make(map[string][]PodStatusResponse)
	for _, pod := range pods {
		groupName := &pod.Spec.NodeName

		banzaiUtils.LogInfo(utils.TagStatus, "Pod name: %s status: %s running on node: %s", pod.Name, pod.Status.Phase, groupName)
		podStatus := PodStatusResponse{
			Name:   pod.Name,
			Status: pod.Status.Phase,
		}
		podGroup[*groupName] = append(podGroup[*groupName], podStatus)

	}
	return podGroup
}

type ClusterStatusResponse struct {
	Name       string                         `json:"name"`
	SystemPods map[string][]PodStatusResponse `json:"systemPods"`
	Pods       map[string][]PodStatusResponse `json:"userPods"`
}

func CreatePodMap(client *kubernetes.Clientset, namespace string) (map[string][]PodStatusResponse, error) {

	podList, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil && podList != nil {
		return make(map[string][]PodStatusResponse), fmt.Errorf("PoD list failed: %v", err)
	}

	var podGroupMap map[string][]PodStatusResponse
	podGroupMap = GroupPodsByNode(podList.Items)
	return podGroupMap, nil
}

func ListPodsForCluster(cluster *cluster.Cluster) (ClusterStatusResponse, error) {
	var (
		config *rest.Config
		err    error
	)
	kubeConfig, err := cloud.GetConfig(cluster, "")
	if err != nil {
		return ClusterStatusResponse{}, err
	}
	if kubeConfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	}
	if err != nil {
		return ClusterStatusResponse{}, fmt.Errorf("K8S Connection Failed: %v", err)
	}

	client := kubernetes.NewForConfigOrDie(config)
	systemPodGroupMap, err := CreatePodMap(client, metav1.NamespaceSystem)
	if err != nil {
		return ClusterStatusResponse{}, err
	}
	podGroupMap, err := CreatePodMap(client, metav1.NamespaceDefault)
	if err != nil {
		return ClusterStatusResponse{}, err
	}
	return ClusterStatusResponse{
		Name:       cluster.Name,
		SystemPods: systemPodGroupMap,
		Pods:       podGroupMap,
	}, nil
}
