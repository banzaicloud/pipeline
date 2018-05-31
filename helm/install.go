package helm

import (
	"fmt"
	"github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	stableRepository = "stable"
	banzaiRepository = "banzaicloud-stable"
)

//PreInstall create's ServiceAccount and AccountRoleBinding
func PreInstall(helmInstall *helm.Install, kubeConfig []byte) error {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagHelmInstall})
	log.Info("start pre-install")

	client, err := GetK8sConnection(kubeConfig)
	if err != nil {
		log.Errorf("could not get kubernetes client: %s", err)
		return err
	}

	v1MetaData := metav1.ObjectMeta{
		Name: helmInstall.ServiceAccount, // "tiller",
	}

	serviceAccount := &apiv1.ServiceAccount{
		ObjectMeta: v1MetaData,
	}
	log.Info("create service account")
	for i := 0; i <= 5; i++ {
		_, err = client.CoreV1().ServiceAccounts(helmInstall.Namespace).Create(serviceAccount)
		if err != nil {
			log.Warnf("create service account failed: %s", err.Error())
			if strings.Contains(err.Error(), "etcdserver: request timed out") {
				time.Sleep(time.Duration(10) * time.Second)
				continue
			}
			if !strings.Contains(err.Error(), "already exists") {
				return errors.Wrap(err, fmt.Sprintf("create service account failed: %s", err))
			}
		}
		break
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
	log.Info("create cluster roles")
	clusterRoleName := helmInstall.ServiceAccount
	for i := 0; i <= 5; i++ {
		_, err = client.RbacV1().ClusterRoles().Create(clusterRole)
		if err != nil {
			if strings.Contains(err.Error(), "etcdserver: request timed out") {
				time.Sleep(time.Duration(10) * time.Second)
				continue
			} else if strings.Contains(err.Error(), "is forbidden") {
				_, errGet := client.RbacV1().ClusterRoles().Get("cluster-admin", metav1.GetOptions{})
				if errGet != nil {
					return errors.Wrap(err, fmt.Sprintf("clusterrole create error: %s cluster-admin not found: %s", err, errGet))
				}
				clusterRoleName = "cluster-admin"
				break
			}
			log.Warnf("create roles failed: %s", err.Error())
			if !strings.Contains(err.Error(), "already exists") {
				return errors.Wrap(err, fmt.Sprintf("create roles failed: %s", err))
			}
		}
		break
	}

	log.Debugf("ClusterRole Name: %s", clusterRoleName)
	log.Debugf("ServiceAccount Name: %s", helmInstall.ServiceAccount)
	clusterRoleBinding := &v1.ClusterRoleBinding{
		ObjectMeta: v1MetaData,
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName, // "tiller",
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      helmInstall.ServiceAccount, // "tiller",
				Namespace: helmInstall.Namespace,
			}},
	}
	log.Info("create cluster role bindings")
	for i := 0; i <= 5; i++ {

		_, err = client.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding)
		if err != nil {
			log.Warnf("create role bindings failed: %s", err.Error())
			if strings.Contains(err.Error(), "etcdserver: request timed out") {
				time.Sleep(time.Duration(10) * time.Second)
				continue
			}
			if !strings.Contains(err.Error(), "already exists") {
				return errors.Wrap(err, fmt.Sprintf("create role bindings failed: %s", err))
			}
		}
		break
	}

	return nil
}

// RetryHelmInstall retries for a configurable time/interval
// Azure AKS sometimes failing because of TLS handshake timeout, there are several issues on GitHub about that:
// https://github.com/Azure/AKS/issues/112, https://github.com/Azure/AKS/issues/116, https://github.com/Azure/AKS/issues/14
func RetryHelmInstall(helmInstall *helm.Install, kubeconfig []byte) error {
	log := logger.WithFields(logrus.Fields{"tag": "RetryHelmInstall"})
	retryAttempts := viper.GetInt(constants.HELM_RETRY_ATTEMPT_CONFIG)
	retrySleepSeconds := viper.GetInt(constants.HELM_RETRY_SLEEP_SECONDS)
	for i := 0; i <= retryAttempts; i++ {
		log.Infof("Waiting %d/%d", i, retryAttempts)
		err := Install(helmInstall, kubeconfig)
		if err != nil {
			if strings.Contains(err.Error(), "net/http: TLS handshake timeout") {
				time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
				continue
			}
		}
		return nil
	}
	return fmt.Errorf("timeout during helm install")
}

func CreateEnvSettings(helmRepoHome string) helm_env.EnvSettings {
	var settings helm_env.EnvSettings
	settings.Home = helmpath.Home(helmRepoHome)
	return settings
}

func GenerateHelmRepoPath(path string) string {
	var stateStorePath = config.GetStateStorePath("")
	const helmPostFix = "helm"
	return fmt.Sprintf("%s/%s/%s", stateStorePath, path, helmPostFix)
}

func DownloadChartFromRepo(name, path string) (string, error) {
	log := logger.WithFields(logrus.Fields{"tag": "DownloadChartFromRepo"})
	settings := CreateEnvSettings(path)
	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Getters:  getter.All(settings),
	}
	if _, err := os.Stat(settings.Home.Archive()); os.IsNotExist(err) {
		log.Infof("Creating '%s' directory.", settings.Home.Archive())
		os.MkdirAll(settings.Home.Archive(), 0744)
	}

	log.Infof("Downloading helm chart '%s' to '%s'", name, settings.Home.Archive())
	filename, _, err := dl.DownloadTo(name, "", settings.Home.Archive())
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, errors.Wrapf(err, "Could not create absolute path from %s", filename)
		}
		log.Debugf("Fetched helm chart '%s' to '%s'", name, filename)
		return lname, nil
	}

	return filename, errors.Wrapf(err, "Failed to download %q", name)
}

// Installs helm client on the cluster
func InstallHelmClient(path string) error {
	log := logger.WithFields(logrus.Fields{"tag": "InstallHelmClient"})
	settings := CreateEnvSettings(GenerateHelmRepoPath(path))
	if err := EnsureDirectories(settings); err != nil {
		return errors.Wrap(err, "Initializing helm directories failed!")
	}

	if err := ensureDefaultRepos(settings); err != nil {
		return errors.Wrap(err, "Setting up default repos failed!")
	}

	log.Info("Initializing helm client succeeded, happy helming!")
	return nil
}

func EnsureDirectories(env helm_env.EnvSettings) error {
	log := logger.WithFields(logrus.Fields{"tag": "EnsureHelmDirectories"})
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

	log.Info("Setting up helm directories.")

	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			log.Infof("Creating '%s'", p)
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
	log := logger.WithFields(logrus.Fields{"tag": "EnsureDefaultRepositories"})
	home := env.Home
	repoFile := home.RepositoryFile()

	stableRepositoryURL := viper.GetString("helm.stableRepositoryURL")
	banzaiRepositoryURL := viper.GetString("helm.banzaiRepositoryURL")

	log.Infof("Setting up default helm repos.")

	if fi, err := os.Stat(repoFile); err != nil {
		log.Infof("Creating %s", repoFile)
		f := repo.NewRepoFile()
		sr, err := InitRepo(stableRepository, stableRepositoryURL, env)
		if err != nil {
			return errors.Wrapf(err, "cannot init stable repo")
		}
		br, err := InitRepo(banzaiRepository, banzaiRepositoryURL, env)
		if err != nil {
			return errors.Wrapf(err, "cannot init banzai repo")
		}
		f.Add(sr, br)
		if err := f.WriteFile(repoFile, 0644); err != nil {
			return errors.Wrap(err, "cannot create file")
		}
	} else if fi.IsDir() {
		return errors.Errorf("%s must be a file, not a directory!", repoFile)
	}
	return nil
}

func InitRepo(repoName string, repoUrl string, env helm_env.EnvSettings) (*repo.Entry, error) {
	log := logger.WithFields(logrus.Fields{"tag": "InitHelmRepositories"})
	log.Infof("Adding %s repo with URL: %s", repoName, repoUrl)
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

// InstallLocalHelm install helm into the given path
func InstallLocalHelm(path string) error {
	log := logger.WithFields(logrus.Fields{"tag": "InstallLocalHelmClient"})

	if err := InstallHelmClient(path); err != nil {
		return err
	}
	log.Info("Helm client install succeeded")
	return nil
}

// Install uses Kubernetes client to install Tiller.
func Install(helmInstall *helm.Install, kubeConfig []byte) error {
	log := logger.WithFields(logrus.Fields{"tag": "InstallHelmClient"})

	err := PreInstall(helmInstall, kubeConfig)
	if err != nil {
		return err
	}

	opts := installer.Options{
		Namespace:      helmInstall.Namespace,
		ServiceAccount: helmInstall.ServiceAccount,
		UseCanary:      helmInstall.Canary,
		ImageSpec:      helmInstall.ImageSpec,
		MaxHistory:     helmInstall.MaxHistory,
	}
	kubeClient, err := GetK8sConnection(kubeConfig)
	if err != nil {
		return err
	}
	if err := installer.Install(kubeClient, &opts); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			//TODO shouldn'T we just skipp?
			return err
		}
		if helmInstall.Upgrade {
			if err := installer.Upgrade(kubeClient, &opts); err != nil {
				return errors.Wrap(err, "error when upgrading")
			}
			log.Info("Tiller (the Helm server-side component) has been upgraded to the current version.")
		} else {
			log.Info("Warning: Tiller is already installed in the cluster.")
		}
	} else {
		log.Info("Tiller (the Helm server-side component) has been installed into your Kubernetes Cluster.")
	}
	log.Info("Helm install finished")
	return nil
}
