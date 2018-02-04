package helm

import (
	"fmt"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/utils"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/repo"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	stableRepository = "stable"
	banzaiRepository = "banzaicloud-stable"
)

//Create ServiceAccount and AccountRoleBinding
func PreInstall(helmInstall *helm.Install, kubeConfig *[]byte) error {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagHelmInstall})
	log.Info("start pre-install")

	client, err := GetK8sConnection(kubeConfig)
	if err != nil {
		utils.LogErrorf(constants.TagHelmInstall, "could not get kubernetes client: %s", err)
		return err
	}

	v1MetaData := metav1.ObjectMeta{
		Name: helmInstall.ServiceAccount, // "tiller",
	}

	serviceAccount := &apiv1.ServiceAccount{
		ObjectMeta: v1MetaData,
	}
	utils.LogInfo(constants.TagHelmInstall, "create service account")
	_, err = client.CoreV1().ServiceAccounts(helmInstall.Namespace).Create(serviceAccount)
	if err != nil {
		utils.LogErrorf(constants.TagHelmInstall, "create service account failed: %s", err)
		return err
	}

	clusterRoleBinding := &v1.ClusterRoleBinding{
		ObjectMeta: v1MetaData,
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     helmInstall.ServiceAccount, // "tiller",
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      helmInstall.ServiceAccount, // "tiller",
				Namespace: helmInstall.Namespace,
			}},
	}
	utils.LogInfo(constants.TagHelmInstall, "create cluster role bindings")
	_, err = client.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding)
	if err != nil {
		utils.LogErrorf(constants.TagHelmInstall, "create role bindings failed: %s", err)
		return err
	}
	clusterRole := &v1.ClusterRole{
		ObjectMeta: v1MetaData,
		Rules: []v1.PolicyRule{{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"*",
			},
			Verbs: []string{
				"*",
			},
		},
			{
				NonResourceURLs: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			}},
	}
	utils.LogInfo(constants.TagHelmInstall, "create cluster roles")
	_, err = client.RbacV1().ClusterRoles().Create(clusterRole)
	if err != nil {
		utils.LogErrorf(constants.TagHelmInstall, "create roles failed: %s", err)
		return err
	}

	return nil
}

// RetryHelmInstall retries for a configurable time/interval
// Azure AKS sometimes failing because of TLS handshake timeout, there are several issues on GitHub about that:
// https://github.com/Azure/AKS/issues/112, https://github.com/Azure/AKS/issues/116, https://github.com/Azure/AKS/issues/14
func RetryHelmInstall(helmInstall *helm.Install, commonCluster cluster.CommonCluster, path string) error {
	log := logger.WithFields(logrus.Fields{"tag": "RetryHelmInstall"})
	retryAttempts := viper.GetInt(constants.HELM_RETRY_ATTEMPT_CONFIG)
	retrySleepSeconds := viper.GetInt(constants.HELM_RETRY_SLEEP_SECONDS)
	kubeconfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error getting Kubernetes config")
	}
	for i := 0; i <= retryAttempts; i++ {
		log.Debugf("Waiting %d/%d", i, retryAttempts)
		response := Install(helmInstall, kubeconfig, path)
		if strings.Contains(response.Message, "net/http: TLS handshake timeout") {
			time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
			continue
		}
		return nil
	}
	switch commonCluster.GetType() {
	case constants.Amazon:
		log.Errorf("Timeout during waiting for AWS Control Plane to become healthy..")
	case constants.Azure:
		log.Errorf("Timeout during waiting for AKS to become healthy..")
		log.Errorf("https://github.com/Azure/AKS/issues/116")
	}
	return fmt.Errorf("timeout during helm install")
}

func createEnvSettings(helmRepoHome string) helm_env.EnvSettings {
	var settings helm_env.EnvSettings
	settings.Home = helmpath.Home(helmRepoHome)
	return settings
}

func generateHelmRepoPath(path string) string {
	const stateStorePath = "./statestore/"
	const helmPostFix = "/helm"
	return stateStorePath + path + helmPostFix
}

func downloadChartFromRepo(name string) (string, error) {
	settings := createEnvSettings("")
	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Getters:  getter.All(settings),
	}
	if _, err := os.Stat(settings.Home.Archive()); os.IsNotExist(err) {
		utils.LogInfof("downloadChartFromRepo", "Creating '%s' directory.", settings.Home.Archive())
		os.MkdirAll(settings.Home.Archive(), 0744)
	}

	utils.LogInfof("downloadChartFromRepo", "Downloading helm chart '%s' to '%s'", name, settings.Home.Archive())
	filename, _, err := dl.DownloadTo(name, "", settings.Home.Archive())
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, errors.Wrapf(err, "Could not create absolute path from %s", filename)
		}
		utils.LogDebugf("downloadChartFromRepo", "Fetched helm chart '%s' to '%s'", name, filename)
		return lname, nil
	}

	return filename, errors.Errorf("Failed to download %q", name)
}

// Installs helm client on the cluster
func installHelmClient(path string) error {
	const logTag = "installHelmClient"
	settings := createEnvSettings(generateHelmRepoPath(path))
	if err := ensureDirectories(settings); err != nil {
		return errors.Wrap(err, "Initializing helm directories failed!")
	}

	if err := ensureDefaultRepos(settings); err != nil {
		return errors.Wrap(err, "Setting up default repos failed!")
	}

	utils.LogInfo(logTag, "Initializing helm client succeeded, happy helming!")
	return nil
}

func ensureDirectories(env helm_env.EnvSettings) error {
	const logTag = "ensureDirectories"
	home := env.Home
	configDirectories := []string{
		home.String(),
		home.Repository(),
		home.Cache(),
		home.LocalRepository(),
		home.Plugins(),
		home.Starters(),
		home.Archive(),
	}

	utils.LogInfo(logTag, "Setting up helm directories.")

	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			utils.LogInfof(logTag, "Creating '%s'", p)
			if err := os.MkdirAll(p, 0755); err != nil {
				return errors.Wrapf(err, "Could not create '%s'", p)
			}
		} else if !fi.IsDir() {
			return errors.Errorf("'%s' must be a directory", p)
		}
	}
	return nil
}

func ensureDefaultRepos(env helm_env.EnvSettings) error {
	const logTag = "ensureDefaultRepos"
	home := env.Home
	repoFile := home.RepositoryFile()

	stableRepositoryURL := viper.GetString("helm.stableRepositoryURL")
	banzaiRepositoryURL := viper.GetString("helm.banzaiRepositoryURL")

	utils.LogInfo(logTag, "Setting up default helm repos.")

	if fi, err := os.Stat(repoFile); err != nil {
		utils.LogInfof(logTag, "Creating %s", repoFile)
		f := repo.NewRepoFile()
		sr, err := initRepo(stableRepository, stableRepositoryURL, env)
		if err != nil {
			return errors.Wrapf(err, "Cannot init stable repo!")
		}
		br, err := initRepo(banzaiRepository, banzaiRepositoryURL, env)
		if err != nil {
			return errors.Wrapf(err, "Cannot init banzai repo!")
		}
		f.Add(sr, br)
		if err := f.WriteFile(repoFile, 0644); err != nil {
			return errors.Wrap(err, "Cannot create file!")
		}
	} else if fi.IsDir() {
		return errors.Errorf("%s must be a file, not a directory!", repoFile)
	}
	return nil
}

func initRepo(repoName string, repoUrl string, env helm_env.EnvSettings) (*repo.Entry, error) {
	const logTag = "initStableRepo"
	utils.LogInfof(logTag, "Adding %s repo with URL: %s", repoName, repoUrl)
	c := repo.Entry{
		Name:  repoName,
		URL:   repoUrl,
		Cache: env.Home.CacheIndex(repoName),
	}
	r, err := repo.NewChartRepository(&c, getter.All(env))
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create a new ChartRepo")
	}

	// In this case, the cacheFile is always absolute. So passing empty string
	// is safe.
	if err := r.DownloadIndexFile(""); err != nil {
		return nil, errors.Errorf("Looks like %q is not a valid chart repository or cannot be reached: %s", repoUrl, err.Error())
	}

	return &c, nil
}

// Install uses Kubernetes client to install Tiller.
func Install(helmInstall *helm.Install, kubeConfig *[]byte, path string) *components.BanzaiResponse {

	//Installing helm client
	utils.LogInfo(constants.TagHelmInstall, "Installing helm client!")
	if err := installHelmClient(path); err != nil {
		utils.LogErrorf(constants.TagHelmInstall, "%+v\n", err)
		return &components.BanzaiResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    err.Error(),
		}
	}
	utils.LogInfo(constants.TagHelmInstall, "Helm client install succeeded")

	err := PreInstall(helmInstall, kubeConfig)
	if err != nil {
		return &components.BanzaiResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    err.Error(),
		}
	}

	opts := installer.Options{
		Namespace:      helmInstall.Namespace,
		ServiceAccount: helmInstall.ServiceAccount,
		UseCanary:      helmInstall.Canary,
		ImageSpec:      helmInstall.ImageSpec,
		MaxHistory:     helmInstall.MaxHistory,
	}
	_, kubeClient, err := getKubeClient(helmInstall.KubeContext)
	if err != nil {
		utils.LogErrorf(constants.TagHelmInstall, "could not get kubernetes client: %s", err)
		return &components.BanzaiResponse{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("could not get kubernetes client: %s", err),
		}
	}
	if err := installer.Install(kubeClient, &opts); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			utils.LogErrorf(constants.TagHelmInstall, "error installing: %s", err)
			return &components.BanzaiResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("error installing: %s", err),
			}
		}
		if helmInstall.Upgrade {
			if err := installer.Upgrade(kubeClient, &opts); err != nil {
				utils.LogErrorf(constants.TagHelmInstall, "error when upgrading: %s", err)
				return &components.BanzaiResponse{
					StatusCode: http.StatusInternalServerError,
					Message:    fmt.Sprintf("error when upgrading: %s", err),
				}
			}
			utils.LogInfo(constants.TagHelmInstall, "Tiller (the Helm server-side component) has been upgraded to the current version.")
		} else {
			utils.LogInfo(constants.TagHelmInstall, "Warning: Tiller is already installed in the cluster.")
		}
	} else {
		utils.LogInfo(constants.TagHelmInstall, "Tiller (the Helm server-side component) has been installed into your Kubernetes Cluster.")
	}
	utils.LogInfo(constants.TagHelmInstall, "Helm install finished")
	return &components.BanzaiResponse{
		StatusCode: http.StatusOK,
	}
}

// getKubeClient creates a Kubernetes config and client for a given kubeconfig context.
func getKubeClient(context string) (*rest.Config, kubernetes.Interface, error) {
	config, err := configForContext(context)
	if err != nil {
		return nil, nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	return config, client, nil
}

// configForContext creates a Kubernetes REST client configuration for a given kubeconfig context.
func configForContext(context string) (*rest.Config, error) {
	config, err := kube.GetConfig(context).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes config for context %q: %s", context, err)
	}
	return config, nil
}
