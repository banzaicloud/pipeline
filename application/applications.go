package application

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/catalog"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	k8s "github.com/banzaicloud/pipeline/kubernetes"
	"github.com/banzaicloud/pipeline/model"
	pkgCatalog "github.com/banzaicloud/pipeline/pkg/catalog"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helm_env "k8s.io/helm/pkg/helm/environment"
)

var log = config.Logger()

// Application states
const (
	CREATING = "CREATING"
	PENDING  = "PENDING"
	FAILED   = "FAILED"
	DEPLOYED = "DEPLOYED"
	READY    = "READY"
)

// DeleteApplication TODO
func DeleteApplication(app *model.Application, kubeConfig []byte, force bool) error {
	deployment, err := GetDeploymentByName(app, app.CatalogName)
	if err != nil {
		if force {
			return nil
		}
		return err
	}

	err = helm.DeleteDeployment(deployment.ReleaseName, kubeConfig)
	if !force {
		err = nil
	}

	return err
}

// GetDeploymentByName get a Deployment by Name
func GetDeploymentByName(app *model.Application, depName string) (*model.Deployment, error) {
	for _, deployment := range app.Deployments {
		if deployment.Name == depName {
			return deployment, nil
		}
	}
	return nil, errors.New("deployment not found")
}

// GetSpotGuide get spotguide definition based on catalog name
func GetSpotGuide(env helm_env.EnvSettings, catalogName string) (*catalog.CatalogDetails, error) {
	chart, err := catalog.GetCatalogDetails(env, catalogName)
	if err != nil {
		return nil, err
	}
	if chart == nil || chart.Spotguide == nil {
		return nil, errors.New("spotguide file is missing")
	}
	return chart, nil
}

// CreateApplication will gather, create and manage an application deployment
func CreateApplication(am *model.Application, options []pkgCatalog.ApplicationOptions, commonCluster cluster.CommonCluster) error {
	organization, err := auth.GetOrganizationById(am.OrganizationId)
	if err != nil {
		am.Update(model.Application{Status: FAILED, Message: err.Error()})
		return err
	}

	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		return err
	}

	env := catalog.GenerateCatalogEnv(organization.Name)
	// We need to ensure that catalog repository is present
	err = catalog.EnsureCatalog(env)
	if err != nil {
		am.Update(model.Application{Status: FAILED, Message: err.Error()})
		return err
	}
	catalog, err := GetSpotGuide(env, am.CatalogName)
	if err != nil {
		am.Update(model.Application{Status: FAILED, Message: err.Error()})
		return err
	}
	am.Icon = catalog.Chart.Icon
	am.Description = catalog.Chart.Description
	am.Save()

	// merge options and catalog (secret refs)
	err = mergeRefValues(catalog, options)
	if err != nil {
		am.Update(model.Application{Status: FAILED, Message: err.Error()})
		return err
	}

	err = CreateApplicationDeployment(env, am, options, catalog, kubeConfig)
	if err != nil {
		am.Update(model.Application{Status: FAILED, Message: err.Error()})
		return err
	}
	return nil
}

func getReleaseName(kubeConfig []byte) (string, error) {
	// Check if we want filter
	filter := ""
	// Generate release name
	var releaseName string
	// Check if it exists regenerate on collision
	lrs, err := helm.ListDeployments(&filter, kubeConfig)
	if err != nil {
		return "", err
	}
	for i := 0; i < 5; i++ {
		releaseName = pkgHelm.GenerateReleaseName()
		for _, release := range lrs.Releases {
			if release.Name == releaseName {
				continue
			}
		}
		return releaseName, nil
	}
	return "", fmt.Errorf("release name collision: %s", releaseName)
}

// CreateApplicationDeployment will deploy a Catalog with Dependency
func CreateApplicationDeployment(env helm_env.EnvSettings, am *model.Application, options []pkgCatalog.ApplicationOptions, catalogInfo *catalog.CatalogDetails, kubeConfig []byte) error {

	releaseName, err := getReleaseName(kubeConfig)
	if err != nil {
		return err
	}

	// Generate secrets for spotguide
	secretTag := fmt.Sprintf("application:%d", am.ID)

	for name, s := range catalogInfo.Spotguide.Secrets {

		request := secret.CreateSecretRequest{
			Name: releaseName + "-" + name,
			Tags: []string{secretTag},
		}

		if s.TLS != nil {
			request.Type = pkgSecret.TLSSecretType
			request.Values[pkgSecret.TLSHosts] = s.TLS.Hosts
			request.Values[pkgSecret.TLSValidity] = s.TLS.Validity
		}

		if s.Password != nil {
			request.Type = pkgSecret.PasswordSecretType
			request.Values[pkgSecret.Username] = s.Password.Username
			request.Values[pkgSecret.Password] = s.Password.Password
		}

		if _, err := secret.Store.Store(am.OrganizationId, &request); err != nil {
			return err
		}
	}

	// Install secrets into cluster for spotguide
	secretQuery := pkgSecret.ListSecretsQuery{Type: pkgSecret.AllSecrets, Tag: secretTag}
	_, err = cluster.InstallSecretsByK8SConfig(kubeConfig, am.OrganizationId, &secretQuery, helm.DefaultNamespace)
	if err != nil {
		return err
	}

	for _, dependency := range catalogInfo.Spotguide.Depends {
		deployment := &model.Deployment{
			Status: PENDING,
			Name:   dependency.Name,
			Chart:  dependency.Chart.Name,
		}
		model.GetDB().Save(&deployment)
		am.Deployments = append(am.Deployments, deployment)
	}
	// Add the catalog itself
	deployment := &model.Deployment{
		Status: PENDING,
		Name:   catalogInfo.Chart.Name,
		Chart:  am.CatalogName,
	}
	model.GetDB().Save(deployment)
	am.Deployments = append(am.Deployments, deployment)
	am.Save()

	// Ensure dependencies
	for _, dependency := range catalogInfo.Spotguide.Depends {
		d, err := GetDeploymentByName(am, dependency.Name)
		if err != nil {
			return err
		}
		releaseName, err := EnsureDependency(env, dependency, kubeConfig, releaseName)
		if err != nil {
			d.Update(model.Deployment{Status: FAILED, Message: err.Error()})
			break
		}
		d.Update(model.Deployment{Status: READY, ReleaseName: releaseName})
	}
	// Install application
	chart := catalog.CatalogRepository + "/" + am.CatalogName
	values, err := catalog.CreateValuesFromOption(options)
	if err != nil {
		am.Update(model.Application{Status: FAILED, Message: err.Error()})
		return err
	}
	ok, releaseName, err := ChartPresented(am.CatalogName, kubeConfig)
	if err != nil {
		return err
	}
	if !ok {
		resp, err := helm.CreateDeployment(chart, helm.DefaultNamespace, releaseName, values, kubeConfig, env)
		if err != nil {
			deployment.Update(model.Deployment{Status: FAILED, Message: err.Error()})
			return err
		}
		model.GetDB().Model(deployment).Update("release_name", resp.Release.Name)
	}
	deployment.Update(model.Deployment{Status: READY, ReleaseName: releaseName})
	am.Update(model.Application{Status: DEPLOYED})
	return nil
}

// EnsureDependency ensure remote dependency on a given Kubernetes endpoint
func EnsureDependency(env helm_env.EnvSettings, dependency pkgCatalog.ApplicationDependency, kubeConfig []byte, releaseName string) (string, error) {
	log.Debugf("Dependency: %#v", dependency)
	if dependency.Type != "crd" {
		releaseName, err := EnsureChart(env, dependency, kubeConfig, releaseName)
		if err != nil {
			return "", err
		}
		return releaseName, nil
	}
	ready, err := CheckCRD(kubeConfig, dependency.Values)
	if err != nil {
		// Break cycle on error
		return "", err
	}
	if ready {
		return releaseName, nil
	}

	releaseName, err = EnsureChart(env, dependency, kubeConfig, releaseName)
	if err != nil {
		return "", err
	}

	var retry int
	var timeout int
	if dependency.Retry != 0 {
		retry = dependency.Retry
	} else {
		retry = 15
	}
	if dependency.Timeout != 0 {
		retry = dependency.Timeout
	} else {
		timeout = 5
	}
	//Check if dependency is available 10 to timeout
	for i := 0; i < retry; i++ {
		// Check crd should come back with error if not
		ready, err := CheckCRD(kubeConfig, dependency.Values)
		if err != nil {
			// Break cycle on error
			break
		}
		// If no errors happened we exit
		if ready {
			return releaseName, nil
		}
		// Wait 2 sec for next check
		time.Sleep(time.Duration(timeout) * time.Second)
	}
	return "", errors.Wrap(err, "dependency is not ready")
}

// ChartPresented check if a Chart presented on a given Kubernetes cluster
func ChartPresented(chartName string, kubeConfig []byte) (bool, string, error) {
	var filter string
	chartList, err := helm.ListDeployments(&filter, kubeConfig)
	if err != nil {
		return false, "", err
	}
	for _, c := range chartList.Releases {
		log.Debugf("Checking installed charts: %#v", c.Chart.Metadata.Name)
		if c.Chart.Metadata.Name == chartName {
			return true, c.Name, nil
		}
	}
	log.Debugf("Dependency not found: %q", chartName)
	return false, "", nil
}

// EnsureChart ensures a given Helm chart is available on the given Kubernetes cluster
func EnsureChart(env helm_env.EnvSettings, dep pkgCatalog.ApplicationDependency, kubeConfig []byte, releaseName string) (string, error) {
	ok, releaseName, err := ChartPresented(dep.Chart.Name, kubeConfig)
	if err != nil {
		return "", err
	}
	if ok {
		return releaseName, nil
	}

	// TODO this is a workaround to not implement repository handling
	chart := catalog.CatalogRepository + "/" + dep.Chart.Name

	resp, err := helm.CreateDeployment(chart, dep.Namespace, releaseName, nil, kubeConfig, env)
	if err != nil {
		return "", err
	}
	return resp.Release.Name, nil
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

func mergeRefValues(catalog *catalog.CatalogDetails, options []pkgCatalog.ApplicationOptions) error {
	for _, option := range options {
		if option.Ref != "" {
			ref := strings.TrimPrefix(option.Ref, "#/")
			field := reflect.ValueOf(*catalog.Spotguide)
			path := strings.Split(ref, "/")
			for _, fieldName := range path {
				// skip double or ending slashes
				if fieldName != "" {
					if field.Kind() == reflect.Ptr {
						field = field.Elem()
					}
					if field.Kind() == reflect.Map {
						field = field.MapIndex(reflect.ValueOf(fieldName))
					} else if field.Kind() == reflect.Struct {
						field = field.FieldByName(strings.Title(fieldName))
					} else {
						return fmt.Errorf("Can't traverse '%s' ref in spotguide, type: %s", option.Ref, field.Kind().String())
					}
					if !field.IsValid() {
						return fmt.Errorf("Can't find '%s' ref in spotguide", option.Ref)
					}
				}
			}
			value := reflect.ValueOf(option.Value)
			field.Set(value)
		}
	}
	return nil
}
