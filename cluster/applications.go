package cluster

import (
	"github.com/banzaicloud/pipeline/helm"
	k8s "github.com/banzaicloud/pipeline/kubernetes"
	"github.com/banzaicloud/pipeline/model"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func GetState(dep helm.ApplicationDependency, kubeConfig []byte) string {
	// Start with none
	status := "None"
	if dep.Type == "crd" {
		//Check for custom resources
		ok, err := CheckCRD(kubeConfig, dep.Values)
		if err != nil {
			return status
		}
		if !ok {
			return "None"
		}
		return "Ready"
	}
	result, err := helm.ListDeployments(&dep.Chart.Name, kubeConfig)
	if err != nil {
		return status
	}
	if result.Releases[0].Info.Status.String() == "DEPLOYED" {
		status = "Ready"
	}
	return status
}

func SetDeploymentState(app model.ApplicationModel, depName string, state string) {
	for _, release := range app.Deployments {
		if release.Name == depName {
			release.Status = state
		}
	}
	app.Save()
}

func GetSpotGuideFile(name string) (*helm.SpotguideFile, error) {
	chart, err := helm.GetCatalogDetails(name)
	if err != nil {
		return nil, err
	}
	return chart.Spotguide, nil
}

func CreateApplication(name string, kubeConfig []byte) error {
	spotguide, err := GetSpotGuideFile(name)
	if err != nil {
		return err
	}
	CreateApplicationSpotguide(spotguide, kubeConfig)
	return nil
}

func CreateApplicationSpotguide(spotguide *helm.SpotguideFile, kubeConfig []byte) {
	// Create database entry
	am := model.ApplicationModel{
		Name: "test",
	}
	am.Save()
	for name, dependency := range spotguide.Depends {
		deployment := model.Deployment{
			Status: "PENDING",
			Name:   name,
			Chart:  dependency.Chart.Name,
		}
		am.Deployments = append(am.Deployments, deployment)
	}

	// Ensure dependencies
	for name, dependency := range spotguide.Depends {
		err := EnsureDependency(dependency, kubeConfig)
		if err != nil {
			SetDeploymentState(am, name, "FAILED")
		}

	}
	// Install application
	chart := helm.CatalogRepository + "/" + "TODOcahrtname"
	helm.CreateDeployment(chart, "", nil, nil, helm.CatalogPath)
}

func EnsureDependency(dependency helm.ApplicationDependency, kubeConfig []byte) error {
	state := GetState(dependency, kubeConfig)
	if state != "Ready" {
		InstallChart(dependency)
	}
	// We don't have to wait dependency
	if dependency.Type != "crd" {
		return nil
	}
	var err error
	//Check if dependency is available 10 to timeout
	for i := 0; i < 10; i++ {
		// Check crd should come back with error if not
		ready, err := CheckCRD(kubeConfig, dependency.Values)
		if err == nil {
			// Break cycle on error
			ready = false
			break
		}
		// If no errors happened we exit
		if ready {
			return nil
		}
		// Wait 2 sec for next check
		time.Sleep(2)
	}
	return errors.Wrap(err, "dependency is not ready")
}

func InstallChart(dependency helm.ApplicationDependency) {
	chart := dependency.Chart.Repository + "/" + dependency.Chart.Name
	//TODO connect with cluster
	helm.CreateDeployment(chart, "", nil, nil, helm.CatalogPath)
}

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
	return subSet(requiredCrds, availableCrds), nil
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
