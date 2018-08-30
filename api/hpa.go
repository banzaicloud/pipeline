package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	pkgCommmon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/hpa"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const hpaAnnotationPrefix = "hpa.autoscaling.banzaicloud.io"

// PutHpaResource create/updates a Hpa resource annotations on scaleTarget - a K8s deployment/statefulset
func PutHpaResource(c *gin.Context) {

	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		log.Errorf("could not get the k8s config")
		return
	}

	var scalingRequest *hpa.DeploymentScalingRequest
	err := c.BindJSON(&scalingRequest)
	if err != nil {
		err := errors.Wrap(err, "Error parsing request:")
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}
	log.Info("Parse deployment succeeded")

	err = SetDeploymentAutoscalingInfo(kubeConfig, *scalingRequest)
	if err != nil {
		err := errors.Wrap(err, "Error during request processing:")
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during request processing!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, "")
}

// DeleteHpaResource deletes a Hpa resource annotations from scaleTarget - K8s deployment/statefulset
func DeleteHpaResource(c *gin.Context) {

	scaleTarget, ok := ginutils.RequiredQuery(c, "scaleTarget")
	if !ok {
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "missing required param: scaleTarget",
			Error:   "missing required param: scaleTarget",
		})
		return
	}
	log.Infof("getting hpa details for scaleTarget: [%s]", scaleTarget)

	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		log.Errorf("could not get the k8s config")
		return
	}

	err := DeleteDeploymentAutoscalingInfo(kubeConfig, scaleTarget)
	if err != nil {
		err := errors.Wrap(err, "Error during request processing:")
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during request processing!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.JSON(http.StatusOK, "")
}

// GetHpaResource returns a Hpa resource bound to a K8s deployment/statefulset
func GetHpaResource(c *gin.Context) {
	scaleTarget, ok := ginutils.RequiredQuery(c, "scaleTarget")
	if !ok {
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "missing required param: scaleTarget",
			Error:   "missing required param: scaleTarget",
		})
		return
	}
	log.Infof("getting hpa details for scaleTarget: [%s]", scaleTarget)

	kubeConfig, ok := GetK8sConfig(c)
	if !ok {
		log.Errorf("could not get the k8s config")
		return
	}

	deploymentResponse, err := GetHpaResources(scaleTarget, kubeConfig)
	if err != nil {
		log.Error("Error during getting deployment details: ", err.Error())

		httpStatusCode := http.StatusInternalServerError
		if _, ok := err.(*helm.DeploymentNotFoundError); ok {
			httpStatusCode = http.StatusBadRequest
		}

		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error getting deployment",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, deploymentResponse)

}

func GetHpaResources(scaleTragetRef string, kubeConfig []byte) ([]hpa.DeploymentScalingInfo, error) {
	client, err := helm.GetK8sConnection(kubeConfig)
	if err != nil {
		log.Errorf("Getting K8s client failed: %s", err.Error())
		return nil, err
	}
	responseDeployments := make([]hpa.DeploymentScalingInfo, 0)

	listOption := v12.ListOptions{
		TypeMeta: v12.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v1",
		},
	}
	hpaList, err := client.AutoscalingV2beta1().HorizontalPodAutoscalers(v12.NamespaceAll).List(listOption)
	if err != nil {
		log.Errorf("Getting hpa for %v failed: %s", scaleTragetRef, err.Error())
	} else {

		for _, hpaItem := range hpaList.Items {
			if !hpaBelongsToDeployment(hpaItem, scaleTragetRef) {
				continue
			}

			log.Infof("hpa found: %v for scaleTragetRef: %v", hpaItem.Name, scaleTragetRef)
			deploymentItem := hpa.DeploymentScalingInfo{
				ScaleTarget:          scaleTragetRef,
				Kind:          hpaItem.Spec.ScaleTargetRef.Kind,
				MinReplicas:   *hpaItem.Spec.MinReplicas,
				MaxReplicas:   hpaItem.Spec.MaxReplicas,
				CustomMetrics: map[string]hpa.CustomMetricStatus{},
			}

			for _, metric := range hpaItem.Spec.Metrics {
				switch metric.Type {
				case v2beta1.ResourceMetricSourceType:
					switch metric.Resource.Name {
					case v1.ResourceCPU:
						deploymentItem.Cpu = getResourceMetricStatus(hpaItem, metric)
					case v1.ResourceMemory:
						deploymentItem.Memory = getResourceMetricStatus(hpaItem, metric)
					}
				case v2beta1.PodsMetricSourceType:
					log.Warnf("custom metric %v found for hpa: %v", metric.Pods.MetricName, hpaItem.Name)
					deploymentItem.CustomMetrics[metric.Pods.MetricName] = getPodMetricStatus(hpaItem, metric)
				default:
					log.Warnf("metric found: %v for hpa: %v", metric.Type, hpaItem.Name)
				}
			}

			deploymentItem.Status.Message = generateStatusMessage(hpaItem.Status)
			responseDeployments = append(responseDeployments, deploymentItem)
		}

	}

	return responseDeployments, nil
}

func generateStatusMessage(status v2beta1.HorizontalPodAutoscalerStatus) string {
	for _, condition := range status.Conditions {
		if condition.Type == v2beta1.ScalingActive {
			return fmt.Sprintf("%v=%v : %v", v2beta1.ScalingActive, condition.Status, condition.Message)
		}
	}
	return "n/a"
}

func getResourceMetricStatus(hpaItem v2beta1.HorizontalPodAutoscaler, metric v2beta1.MetricSpec) hpa.ResourceMetricStatus {
	metricStatus := hpa.ResourceMetricStatus{}
	if metric.Resource.TargetAverageUtilization != nil {
		metricStatus.TargetAverageValue = fmt.Sprint(*metric.Resource.TargetAverageUtilization)
		metricStatus.TargetAverageValueType = hpa.PercentageValueType
	} else if metric.Resource.TargetAverageValue != nil {
		metricStatus.TargetAverageValue = metric.Resource.TargetAverageValue.String()
		metricStatus.TargetAverageValueType = hpa.QuantityValueType
	}
	for _, currentMetricStatus := range hpaItem.Status.CurrentMetrics {
		if currentMetricStatus.Resource != nil && currentMetricStatus.Resource.Name == metric.Resource.Name {
			if currentMetricStatus.Resource.CurrentAverageUtilization != nil {
				metricStatus.CurrentAverageValue = fmt.Sprint(*currentMetricStatus.Resource.CurrentAverageUtilization)
				metricStatus.TargetAverageValueType = hpa.PercentageValueType
			} else if !currentMetricStatus.Resource.CurrentAverageValue.IsZero() {
				metricStatus.CurrentAverageValue = currentMetricStatus.Resource.CurrentAverageValue.String()
				metricStatus.CurrentAverageValueType = hpa.QuantityValueType
			}
		}
	}

	return metricStatus
}

func getPodMetricStatus(hpaItem v2beta1.HorizontalPodAutoscaler, metric v2beta1.MetricSpec) hpa.CustomMetricStatus {
	metricStatus := hpa.CustomMetricStatus{}
	metricStatus.TargetAverageValue = metric.Pods.TargetAverageValue.String()
	metricStatus.Type = "pod"
	for _, currentMetricStatus := range hpaItem.Status.CurrentMetrics {
		if currentMetricStatus.Pods != nil && currentMetricStatus.Pods.MetricName == metric.Pods.MetricName {
			if !currentMetricStatus.Pods.CurrentAverageValue.IsZero() {
				metricStatus.CurrentAverageValue = currentMetricStatus.Pods.CurrentAverageValue.String()
			}
		}
	}

	return metricStatus
}

func hpaBelongsToDeployment(hpa v2beta1.HorizontalPodAutoscaler, scaleTragetRef string) bool {
	// TODO later may be check gvk as well
	if hpa.Spec.ScaleTargetRef.Name != scaleTragetRef {
		return false
	}
	return true
}

func DeleteDeploymentAutoscalingInfo(kubeConfig []byte, scaleTarget string) error {
	client, err := helm.GetK8sConnection(kubeConfig)
	if err != nil {
		log.Errorf("Getting K8s client failed: %s", err.Error())
		return err
	}

	// find deployment & update hpa annotations
	// get doesn't work with v12.NamespaceAll only if you specify the namespace exactly
	// deployment, err := client.AppsV1().Deployments(v12.NamespaceAll).Get(request.Name, v12.GetOptions{})
	scaleTargetFound := false

	deploymentList, err := client.AppsV1().Deployments(v12.NamespaceAll).List(v12.ListOptions{})
	for _, dep := range deploymentList.Items {
		if dep.Name == scaleTarget {
			scaleTargetFound = true
			log.Infof("remove annotations on deployment: %v", dep.Name)
			dep.Annotations = removeHpaAnnotations(dep.Annotations)
			_, err = client.AppsV1().Deployments(dep.Namespace).Update(&dep)
		}
	}

	// find statefulset & update hpa annotations
	statefulSetList, err := client.AppsV1().StatefulSets(v12.NamespaceAll).List(v12.ListOptions{})
	for _, stsset := range statefulSetList.Items {
		if stsset.Name == scaleTarget {
			scaleTargetFound = true
			log.Infof("remove annotations on statefulset: %v", stsset.Name)
			stsset.Annotations = removeHpaAnnotations(stsset.Annotations)
			_, err = client.AppsV1().StatefulSets(stsset.Namespace).Update(&stsset)
		}
	}

	if !scaleTargetFound {
		return errors.Errorf("scaleTarget: %v not found!", scaleTarget)
	}

	return nil
}

func SetDeploymentAutoscalingInfo(kubeConfig []byte, request hpa.DeploymentScalingRequest) error {
	client, err := helm.GetK8sConnection(kubeConfig)
	if err != nil {
		log.Errorf("Getting K8s client failed: %s", err.Error())
		return err
	}

	// find deployment & update hpa annotations
	// get doesn't work with v12.NamespaceAll only if you specify the namespace exactly
	// deployment, err := client.AppsV1().Deployments(v12.NamespaceAll).Get(request.Name, v12.GetOptions{})
	scaleTargetFound := false

	deploymentList, err := client.AppsV1().Deployments(v12.NamespaceAll).List(v12.ListOptions{})
	for _, dep := range deploymentList.Items {
		if dep.Name == request.ScaleTarget {
			scaleTargetFound = true
			log.Infof("set annotations on deployment: %v", dep.Name)
			setupHpaAnnotations(request, dep.Annotations)
			_, err = client.AppsV1().Deployments(dep.Namespace).Update(&dep)
		}
	}

	// find statefulset & update hpa annotations
	statefulSetList, err := client.AppsV1().StatefulSets(v12.NamespaceAll).List(v12.ListOptions{})
	for _, stsset := range statefulSetList.Items {
		if stsset.Name == request.ScaleTarget {
			scaleTargetFound = true
			log.Infof("set annotations on statefulset: %v", stsset.Name)
			setupHpaAnnotations(request, stsset.Annotations)
			_, err = client.AppsV1().StatefulSets(stsset.Namespace).Update(&stsset)
		}
	}

	if !scaleTargetFound {
		return errors.Errorf("scaleTarget: %v not found!", request.ScaleTarget)
	}

	return nil
}

func setupHpaAnnotations(request hpa.DeploymentScalingRequest, annotations map[string]string) {
	// TODO validation
	annotations[fmt.Sprintf("%v/minReplicas", hpaAnnotationPrefix)] = fmt.Sprint(request.MinReplicas)
	annotations[fmt.Sprintf("%v/maxReplicas", hpaAnnotationPrefix)] = fmt.Sprint(request.MaxReplicas)

	setupResourceMetricAnnotation(annotations, "cpu", request.Cpu)
	setupResourceMetricAnnotation(annotations, "memory", request.Memory)

	for customMetricName, customMetric := range request.CustomMetrics {
		setupCustomMetricAnnotation(annotations, customMetricName, customMetric)
	}
}

func removeHpaAnnotations(annotations map[string]string) map[string]string {
	newAnnotations := make(map[string]string, 0)
	for key, value := range annotations {
		if !strings.Contains(key, hpaAnnotationPrefix) {
			newAnnotations[key] = value
		}
	}
	return newAnnotations
}

func setupResourceMetricAnnotation(annotations map[string]string, prefix string, resourceMetric hpa.ResourceMetric) {
	if len(resourceMetric.TargetAverageValue) > 0 {
		switch resourceMetric.TargetAverageValueType {
		case hpa.PercentageValueType:
			annotations[fmt.Sprintf("%v.%v/targetAverageUtilization", prefix, hpaAnnotationPrefix)] = resourceMetric.TargetAverageValue
		case hpa.QuantityValueType:
			annotations[fmt.Sprintf("%v.%v/targetAverageValue", prefix, hpaAnnotationPrefix)] = resourceMetric.TargetAverageValue
		}
	}
}

func setupCustomMetricAnnotation(annotations map[string]string, customMetricName string, customMetric hpa.CustomMetric) {
	if len(customMetric.TargetAverageValue) > 0 {
		switch customMetric.Type {
		case "pod":
			annotations[fmt.Sprintf("pod.%v/%v", hpaAnnotationPrefix, customMetricName)] = customMetric.TargetAverageValue
		}
	}
}
