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
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/gin-gonic/gin"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/pkg/brn"
	pkgCommmon "github.com/banzaicloud/pipeline/pkg/common"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/hpa"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/banzaicloud/pipeline/src/api/common"
)

const hpaAnnotationPrefix = "hpa.autoscaling.banzaicloud.io"

type scaleTargetNotFoundError struct {
	scaleTargetRef string
}

func (e *scaleTargetNotFoundError) Error() string {
	return fmt.Sprintf("scaleTarget: %v not found!", e.scaleTargetRef)
}

type HPAAPI struct {
	featureService integratedservices.Service
	clientFactory  common.ClientFactory
	configFactory  common.ConfigFactory
	clusterGetter  common.ClusterGetter
	errorHandler   emperror.Handler
}

// NewHPAAPI returns a new HPAAPI.
func NewHPAAPI(
	featureService integratedservices.Service,
	clientFactory common.ClientFactory,
	configFactory common.ConfigFactory,
	clusterGetter common.ClusterGetter,
	errorHandler emperror.Handler,
) HPAAPI {
	return HPAAPI{
		featureService: featureService,
		clientFactory:  clientFactory,
		configFactory:  configFactory,
		clusterGetter:  clusterGetter,
		errorHandler:   errorHandler,
	}
}

// PutHpaResource create/updates a Hpa resource annotations on scaleTarget - a K8s deployment/statefulset
func (a HPAAPI) PutHpaResource(c *gin.Context) {
	cluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if !ok {
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
			Error:   err.Error(),
		})
		return
	}

	err = scalingRequest.Validate()
	if err != nil {
		err := errors.Wrap(err, "Error parsing request:")
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   err.Error(),
		})
		return
	}

	secretID := brn.New(cluster.GetOrganizationId(), brn.SecretResourceType, cluster.GetConfigSecretId()).String()
	client, err := a.clientFactory.FromSecret(c.Request.Context(), secretID)
	if err != nil {
		a.errorHandler.Handle(err)

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kube client",
			Error:   err.Error(),
		})
		return
	}

	config, err := a.configFactory.FromSecret(c.Request.Context(), secretID)
	if err != nil {
		a.errorHandler.Handle(err)

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kube config",
			Error:   err.Error(),
		})
		return
	}

	// validate custom metrics query
	if len(scalingRequest.CustomMetrics) > 0 {
		ok, err := a.isMonitoringEnabled(c.Request.Context(), cluster.GetID())
		if err != nil {
			err := errors.WithDetails(
				errors.WithMessage(err, "could not determine monitoring status"),
				"clusterId", cluster.GetID(),
			)
			log.Error(err.Error())
			c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
				Code: http.StatusInternalServerError,
			})
			return
		}

		if !ok {
			err := errors.New("Monitoring should be enabled on cluster to be able to setup custom metrics")
			log.Error(err.Error())
			c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Monitoring is not deployed on cluster",
				Error:   err.Error(),
			})
			return
		}
	}
	for _, cm := range scalingRequest.CustomMetrics {
		value, err := runPrometheusQuery(config, client, cm.Query)
		if err != nil {
			log.Error(err.Error())
			c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Error validating custom metrics query!",
				Error:   err.Error(),
			})
			return
		}
		if len(value.String()) == 0 {

		}
	}

	err = setDeploymentAutoscalingInfo(client, *scalingRequest)
	if err != nil {
		httpStatusCode := http.StatusBadRequest
		if _, ok := err.(*scaleTargetNotFoundError); ok {
			httpStatusCode = http.StatusNotFound
		}

		err := errors.Wrap(err, "Error during request processing")
		log.Error(err.Error())
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error during request processing!",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusCreated)
}

// NOTE: this is a temporary solution. At some point this functionality should be extracted.
func (a HPAAPI) isMonitoringEnabled(ctx context.Context, clusterID uint) (bool, error) {
	_, err := a.featureService.Details(ctx, clusterID, "monitoring")
	if integratedservices.IsIntegratedServiceNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func runPrometheusQuery(config *rest.Config, client kubernetes.Interface, query string) (model.Value, error) {
	prometheusEndpointPort := global.Config.Cluster.Autoscale.HPA.Prometheus.LocalPort
	prometheusServiceName := global.Config.Cluster.Autoscale.HPA.Prometheus.ServiceName
	pipelineSystemNamespace := global.Config.Cluster.Namespace
	serviceContext := global.Config.Cluster.Autoscale.HPA.Prometheus.ServiceContext
	prometheusPodLabels := labels.Set{"app": "prometheus", "prometheus": prometheusServiceName}
	log.Debugf("create kubernetes tunnel for %v", prometheusPodLabels)
	tunnel, err := k8sutil.NewKubeTunnel(pipelineSystemNamespace, client, config, prometheusPodLabels.AsSelector(), prometheusEndpointPort)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create kubernetes tunnel")
	}
	defer tunnel.Close()
	log.Debugf("tunnel listening on port: %d", tunnel.Local)

	cfg := promapi.Config{
		Address:      fmt.Sprintf("http://localhost:%d/%s", tunnel.Local, serviceContext),
		RoundTripper: &http.Transport{},
	}
	promClient, err := promapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	promAPI := promv1.NewAPI(promClient)
	value, _, err := promAPI.Query(context.Background(), query, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	var sampleValue model.SampleValue
	switch value.Type() {
	case model.ValVector:
		samples := value.(model.Vector)
		if len(samples) == 0 {
			return nil, fmt.Errorf("query '%s' returned no samples", query)
		}

		sampleValue = samples[0].Value
	case model.ValScalar:
		scalar := value.(*model.Scalar)
		sampleValue = scalar.Value
	}

	if sampleValue.String() == "NaN" {
		return nil, fmt.Errorf("query '%s' returned no samples: %s", query, sampleValue.String())
	}

	log.Debugf("prometheus query result: %s", sampleValue)

	return value, err
}

// DeleteHpaResource deletes a Hpa resource annotations from scaleTarget - K8s deployment/statefulset
func (a HPAAPI) DeleteHpaResource(c *gin.Context) {
	scaleTarget, ok := ginutils.RequiredQueryOrAbort(c, "scaleTarget")
	if !ok {
		return
	}
	log.Debugf("getting hpa details for scaleTarget: [%s]", scaleTarget)

	cluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		return
	}

	secretID := brn.New(cluster.GetOrganizationId(), brn.SecretResourceType, cluster.GetConfigSecretId()).String()
	client, err := a.clientFactory.FromSecret(c.Request.Context(), secretID)
	if err != nil {
		a.errorHandler.Handle(err)

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kube client",
			Error:   err.Error(),
		})
		return
	}

	err = deleteDeploymentAutoscalingInfo(client, scaleTarget)
	if err != nil {
		httpStatusCode := http.StatusInternalServerError
		if _, ok := err.(*scaleTargetNotFoundError); ok {
			httpStatusCode = http.StatusNotFound
		}

		err := errors.Wrap(err, "Error during request processing")
		log.Error(err.Error())
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error during request processing!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetHpaResource returns a Hpa resource bound to a K8s deployment/statefulset
func (a HPAAPI) GetHpaResource(c *gin.Context) {
	scaleTarget, ok := ginutils.RequiredQueryOrAbort(c, "scaleTarget")
	if !ok {
		return
	}
	log.Debugf("getting hpa details for scaleTarget: [%s]", scaleTarget)

	cluster, ok := a.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		return
	}

	secretID := brn.New(cluster.GetOrganizationId(), brn.SecretResourceType, cluster.GetConfigSecretId()).String()
	client, err := a.clientFactory.FromSecret(c.Request.Context(), secretID)
	if err != nil {
		a.errorHandler.Handle(err)

		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kube client",
			Error:   err.Error(),
		})
		return
	}

	deploymentResponse, err := getHpaResources(scaleTarget, client)
	if err != nil {
		httpStatusCode := http.StatusInternalServerError
		if _, ok := err.(*scaleTargetNotFoundError); ok {
			httpStatusCode = http.StatusNotFound
		}

		err := errors.Wrap(err, "Error during request processing")
		log.Error(err.Error())
		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error getting deployment",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, deploymentResponse)
}

func getHpaResources(scaleTargetRef string, client kubernetes.Interface) (*hpa.DeploymentScalingInfo, error) {
	listOption := metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v1",
		},
	}
	hpaList, err := client.AutoscalingV2beta1().HorizontalPodAutoscalers(metav1.NamespaceAll).List(listOption)
	if err != nil {
		return nil, err
	}

	for _, hpaItem := range hpaList.Items {
		if !hpaBelongsToDeployment(hpaItem, scaleTargetRef) {
			continue
		}

		log.Debugf("hpa found: %v for scaleTragetRef: %v", hpaItem.Name, scaleTargetRef)
		deploymentItem := hpa.DeploymentScalingInfo{
			ScaleTarget:   scaleTargetRef,
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
			case v2beta1.ObjectMetricSourceType:
				log.Warnf("custom metric %v found for hpa: %v", metric.Object.MetricName, hpaItem.Name)
				deploymentItem.CustomMetrics[metric.Object.MetricName] = getCustomMetricStatus(hpaItem, metric)
			default:
				log.Warnf("metric found: %v for hpa: %v", metric.Type, hpaItem.Name)
			}
		}

		deploymentItem.Status.Message = generateStatusMessage(hpaItem.Status)

		if hpaItem.Name != scaleTargetRef {
			deploymentItem.Status.Message = "You can't edit this Horizontal Pod Autoscaler resource, in order to manage it with Pipeline please set the same name as deployment name."
		}
		return &deploymentItem, nil
	}

	return nil, &scaleTargetNotFoundError{scaleTargetRef: scaleTargetRef}
}

func generateStatusMessage(status v2beta1.HorizontalPodAutoscalerStatus) string {
	for _, condition := range status.Conditions {
		if condition.Type == v2beta1.ScalingActive {
			return fmt.Sprintf("%v=%v : %v", v2beta1.ScalingActive, condition.Status, condition.Message)
		}
	}
	return ""
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
				metricStatus.CurrentAverageValue = fmt.Sprint(k8sutil.GetResourceQuantityInBytes(&currentMetricStatus.Resource.CurrentAverageValue))
				metricStatus.CurrentAverageValueType = hpa.QuantityValueType
			}
		}
	}

	return metricStatus
}

func getCustomMetricStatus(hpaItem v2beta1.HorizontalPodAutoscaler, metric v2beta1.MetricSpec) hpa.CustomMetricStatus {
	metricStatus := hpa.CustomMetricStatus{}
	metricName := metric.Object.MetricName
	metricStatus.Query = hpaItem.Annotations[fmt.Sprintf("metric-config.object.%s.prometheus/query", metricName)]
	_, perReplica := hpaItem.Annotations[fmt.Sprintf("metric-config.object.%s.prometheus/per-replica", metricName)]
	if perReplica {
		metricStatus.TargetAverageValue = metric.Object.TargetValue.String()
	} else {
		metricStatus.TargetValue = metric.Object.TargetValue.String()
	}

	for _, currentMetricStatus := range hpaItem.Status.CurrentMetrics {
		if currentMetricStatus.Object != nil && currentMetricStatus.Object.MetricName == metricName {
			// if !currentMetricStatus.Object.CurrentValue.IsZero() {
			metricStatus.CurrentValue = currentMetricStatus.Object.CurrentValue.String()
			// }
		}
	}

	return metricStatus
}

func hpaBelongsToDeployment(hpa v2beta1.HorizontalPodAutoscaler, scaleTragetRef string) bool {
	if hpa.Spec.ScaleTargetRef.Name != scaleTragetRef {
		return false
	}
	return true
}

func deleteDeploymentAutoscalingInfo(client kubernetes.Interface, scaleTarget string) error {
	// find deployment & update hpa annotations
	// get doesn't work with metav1.NamespaceAll only if you specify the namespace exactly
	// deployment, err := client.AppsV1().Deployments(metav1.NamespaceAll).Get(request.Name, metav1.GetOptions{})
	scaleTargetFound := false
	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%v", scaleTarget),
	}
	deploymentList, err := client.AppsV1().Deployments(metav1.NamespaceAll).List(listOptions)
	if err != nil {
		return err
	}
	for _, dep := range deploymentList.Items {
		if dep.Name == scaleTarget {
			scaleTargetFound = true
			log.Debugf("remove annotations on deployment: %v", dep.Name)
			dep.Annotations = removeHpaAnnotations(dep.Annotations)
			_, err = client.AppsV1().Deployments(dep.Namespace).Update(&dep)
			if err != nil {
				return err
			}
		}
	}

	// find statefulset & update hpa annotations
	statefulSetList, err := client.AppsV1().StatefulSets(metav1.NamespaceAll).List(listOptions)
	if err != nil {
		return err
	}
	for _, stsset := range statefulSetList.Items {
		if stsset.Name == scaleTarget {
			scaleTargetFound = true
			log.Debugf("remove annotations on statefulset: %v", stsset.Name)
			stsset.Annotations = removeHpaAnnotations(stsset.Annotations)
			_, err = client.AppsV1().StatefulSets(stsset.Namespace).Update(&stsset)
			if err != nil {
				return err
			}
		}
	}

	if !scaleTargetFound {
		return &scaleTargetNotFoundError{scaleTargetRef: scaleTarget}
	}

	return nil
}

func setDeploymentAutoscalingInfo(client kubernetes.Interface, request hpa.DeploymentScalingRequest) error {
	// find deployment & update hpa annotations
	// get doesn't work with metav1.NamespaceAll only if you specify the namespace exactly
	// deployment, err := client.AppsV1().Deployments(metav1.NamespaceAll).Get(request.Name, metav1.GetOptions{})
	scaleTargetFound := false
	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%v", request.ScaleTarget),
	}
	deploymentList, err := client.AppsV1().Deployments(metav1.NamespaceAll).List(listOptions)
	if err != nil {
		return err
	}
	for _, dep := range deploymentList.Items {
		if dep.Name == request.ScaleTarget {
			scaleTargetFound = true
			log.Debugf("set annotations on deployment: %v", dep.Name)
			dep.Annotations = removeHpaAnnotations(dep.Annotations)
			setupHpaAnnotations(request, dep.Annotations)
			_, err = client.AppsV1().Deployments(dep.Namespace).Update(&dep)
			if err != nil {
				return err
			}
		}
	}

	// find statefulset & update hpa annotations
	statefulSetList, err := client.AppsV1().StatefulSets(metav1.NamespaceAll).List(listOptions)
	if err != nil {
		return err
	}
	for _, stsset := range statefulSetList.Items {
		if stsset.Name == request.ScaleTarget {
			scaleTargetFound = true
			log.Debugf("set annotations on statefulset: %v", stsset.Name)
			stsset.Annotations = removeHpaAnnotations(stsset.Annotations)
			setupHpaAnnotations(request, stsset.Annotations)
			_, err = client.AppsV1().StatefulSets(stsset.Namespace).Update(&stsset)
			if err != nil {
				return err
			}
		}
	}

	if !scaleTargetFound {
		return &scaleTargetNotFoundError{scaleTargetRef: request.ScaleTarget}
	}

	return nil
}

func setupHpaAnnotations(request hpa.DeploymentScalingRequest, annotations map[string]string) {
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
	if len(customMetric.TargetValue) > 0 {
		annotations[fmt.Sprintf("prometheus.%v.%v/targetValue", customMetricName, hpaAnnotationPrefix)] = customMetric.TargetValue
	} else if len(customMetric.TargetAverageValue) > 0 {
		annotations[fmt.Sprintf("prometheus.%v.%v/targetAverageValue", customMetricName, hpaAnnotationPrefix)] = customMetric.TargetAverageValue
	}

	if len(customMetric.TargetAverageValue) > 0 {

	}
	annotations[fmt.Sprintf("prometheus.%v.%v/query", customMetricName, hpaAnnotationPrefix)] = customMetric.Query
}
