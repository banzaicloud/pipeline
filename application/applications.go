package application

import (
	"fmt"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/catalog"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	k8s "github.com/banzaicloud/pipeline/kubernetes"
	"github.com/banzaicloud/pipeline/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"time"
)

var logger *logrus.Logger
var log *logrus.Entry

func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"action": "Helm"})
}

// SetDeploymentState Set a Deployment state related to an Application
func SetDeploymentState(app *model.ApplicationModel, depName string, state string) {
	for _, deployment := range app.Deployments {
		if deployment.Name == depName {
			deployment.Status = state
			model.GetDB().Save(deployment)
		}
	}
}

// GetSpotGuide get spotguide definition based on catalog name
func GetSpotGuide(env helm_env.EnvSettings, catalogName string) (*catalog.CatalogDetails, error) {
	chart, err := catalog.GetCatalogDetails(env, catalogName)
	if err != nil {
		return nil, err
	}
	if chart.Spotguide == nil {
		return nil, errors.New("spotguide file is missing from spotguide.yaml")
	}
	return chart, nil
}

// CreateApplication will gather, create and manage an application deployment
func CreateApplication(am model.ApplicationModel, options []catalog.ApplicationOptions, cluster cluster.CommonCluster) error {
	organization, err := auth.GetOrganizationById(am.OrganizationId)
	if err != nil {
		return err
	}
	env := catalog.GenerateGatalogEnv(organization.Name)
	// Create database entry
	cluster.GetStatus()
	//Todo check if cluster ready
	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return err
	}
	catalog, err := GetSpotGuide(env, am.CatalogName)
	if err != nil {
		model.GetDB().Model(&am).Update("status", err.Error())
		return err
	}
	am.Icon = catalog.Chart.Icon
	am.Description = catalog.Chart.Description
	am.Save()
	err = CreateApplicationDeployment(env, &am, options, catalog, kubeConfig)
	if err != nil {
		model.GetDB().Model(&am).Update("status", err.Error())
		return err
	}
	return nil
}

// CreateApplicationDeployment will deploy a Catalog with Dependency
func CreateApplicationDeployment(env helm_env.EnvSettings, am *model.ApplicationModel, options []catalog.ApplicationOptions, catalogInfo *catalog.CatalogDetails, kubeConfig []byte) error {
	for _, dependency := range catalogInfo.Spotguide.Depends {
		deployment := &model.Deployment{
			Status: "PENDING",
			Name:   dependency.Name,
			Chart:  dependency.Chart.Name,
		}
		model.GetDB().Save(&deployment)
		am.Deployments = append(am.Deployments, deployment)
	}
	deployment := &model.Deployment{
		Status: "PENDING",
		Name:   catalogInfo.Chart.Name,
		Chart:  am.CatalogName,
	}
	model.GetDB().Save(deployment)
	am.Deployments = append(am.Deployments, deployment)
	for _, d := range am.Deployments {
		log.Debugf("Deployment ID: %s", d.ID)
	}
	am.Save()
	// Ensure dependencies
	for _, dependency := range catalogInfo.Spotguide.Depends {
		err := EnsureDependency(env, dependency, kubeConfig)
		if err != nil {
			SetDeploymentState(am, dependency.Name, "FAILED")
			break
		}
		SetDeploymentState(am, dependency.Name, "READY")
	}
	// Install application
	chart := catalog.CatalogRepository + "/" + am.CatalogName
	values, err := catalog.CreateValuesFromOption(options)
	if err != nil {
		model.GetDB().Model(&am).Update("status", fmt.Sprintf("FAILED: %s", err))
		return err
	}
	ok, err := ChartPresented(am.CatalogName, kubeConfig)
	if err != nil {
		return err
	}
	if !ok {
		resp, err := helm.CreateDeployment(chart, "", values, kubeConfig, env)
		if err != nil {
			deployment.Update("FAILED")
			return err
		}
		model.GetDB().Model(deployment).Update("release_name", resp.Release.Name)
	}
	model.GetDB().Model(deployment).Update("status", "READY")
	model.GetDB().Model(&am).Update("status", "DEPLOYED")
	return nil
}

// EnsureDependency ensure remote dependency on a given Kubernetes endpoint
func EnsureDependency(env helm_env.EnvSettings, dependency catalog.ApplicationDependency, kubeConfig []byte) error {
	log.Debugf("Dependency: %#v", dependency)
	if dependency.Type != "crd" {
		EnsureChart(env, dependency, kubeConfig)
		return nil
	}
	ready, err := CheckCRD(kubeConfig, dependency.Values)
	if err != nil {
		// Break cycle on error
		return err
	}
	if ready {
		return nil
	}
	EnsureChart(env, dependency, kubeConfig)
	//Check if dependency is available 10 to timeout
	for i := 0; i < 15; i++ {
		// Check crd should come back with error if not
		ready, err := CheckCRD(kubeConfig, dependency.Values)
		if err != nil {
			// Break cycle on error
			break
		}
		// If no errors happened we exit
		if ready {
			return nil
		}
		// Wait 2 sec for next check
		time.Sleep(2 * time.Second)
	}
	return errors.Wrap(err, "dependency is not ready")
}

// ChartPresented check if a Chart presented on a given Kubernetes cluster
func ChartPresented(chartName string, kubeConfig []byte) (bool, error) {
	var filter string
	chartList, err := helm.ListDeployments(&filter, kubeConfig)
	if err != nil {
		return false, err
	}
	for _, c := range chartList.Releases {
		log.Debugf("Checking installed charts: %#v", c.Chart.Metadata.Name)
		if c.Chart.Metadata.Name == chartName {
			return true, nil
		}
	}
	log.Debugf("Dependency not found: %q", chartName)
	return false, nil
}

// EnsureChart ensures a given Helm chart is available on the given Kubernetes cluster
func EnsureChart(env helm_env.EnvSettings, dep catalog.ApplicationDependency, kubeConfig []byte) error {
	ok, err := ChartPresented(dep.Chart.Name, kubeConfig)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	chart := dep.Chart.Repository + "/" + dep.Chart.Name

	helm.CreateDeployment(chart, "", nil, kubeConfig, env)
	return nil
}

// CheckCRD check for CustomResourceDefinitions
func CheckCRD(kubeConfig []byte, requiredCrds []string) (bool, error) {
	clientset, err := k8s.GetApiExtensionClient(kubeConfig)
	if err != nil {
		return false, err
	}
	crds, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	var availableCrds []string
	for _, crd := range crds.Items {
		availableCrds = append(availableCrds, crd.Name)
	}
	log.Debugf("Required: %#v", requiredCrds)
	log.Debugf("Available: %#v", availableCrds)
	ok := subSet(requiredCrds, availableCrds)
	log.Debugf("Match: %v", ok)
	return ok, nil
}

// Is A subset of B
func subSet(a, b []string) bool {
	set := make(map[string]bool)
	for _, v := range b {
		set[v] = true
	}
	for _, v := range a {
		if !set[v] {
			return false
		}
	}
	return true
}
