// Copyright © 2020 Banzai Cloud
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

package helmadapter

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/storage/driver"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// Components in charge for implementing release helm related operations.
type releaser struct {
	logger Logger
}

func NewReleaser(logger Logger) helm.Releaser {
	return releaser{
		logger: logger,
	}
}

func (r releaser) Install(_ context.Context, helmEnv helm.HelmEnv, kubeConfig helm.KubeConfigBytes, releaseInput helm.Release, options helm.Options) (string, error) {
	// customize the settings passed forward
	envSettings := r.processEnvSettings(helmEnv)

	// component processing the kubeconfig
	restClientGetter := NewCustomGetter(envSettings.RESTClientGetter(), kubeConfig, r.logger)

	ns := "default"
	if options.Namespace != "" {
		ns = options.Namespace
	}

	actionConfig, err := r.getActionConfiguration(restClientGetter, ns)
	if err != nil {
		return "", errors.WrapIf(err, "failed to get  action configuration")
	}

	installAction := action.NewInstall(actionConfig)
	installAction.Namespace = options.Namespace

	name, chartRef, err := installAction.NameAndChart(releaseInput.NameAndChartSlice())
	if err != nil {
		return "", errors.WrapIf(err, "failed to get  name  and chart")
	}
	installAction.ReleaseName = name
	installAction.Wait = options.Wait
	installAction.Timeout = time.Minute * 5

	cp, err := installAction.ChartPathOptions.LocateChart(chartRef, envSettings)
	if err != nil {
		return "", errors.WrapIf(err, "failed to locate chart")
	}

	p := getter.All(envSettings)

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return "", errors.WrapIf(err, "failed to load chart")
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return "", errors.WrapIf(err, "chart is not installable")
	}

	if chartRequested.Metadata.Deprecated {
		r.logger.Warn(" This chart is deprecated")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if installAction.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          installAction.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: envSettings.RepositoryConfig,
					RepositoryCache:  envSettings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return "", errors.WrapIf(err, "failed to update chart dependencies")
				}
			} else {
				return "", errors.WrapIf(err, "failed to check chart dependencies")
			}
		}
	}

	clientSet, err := k8sclient.NewClientFromKubeConfigWithTimeout(kubeConfig, time.Second*10)
	if err != nil {
		return "", errors.WrapIf(err, "failed to create kubernetes client")
	}

	namespaces, err := clientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return "", errors.WrapIf(err, "failed to list kubernetes namespaces")
	}

	foundNs := false
	for _, ns := range namespaces.Items {
		if ns.Name == installAction.Namespace {
			foundNs = true
		}
	}

	if !foundNs {
		if _, err := clientSet.CoreV1().Namespaces().Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: installAction.Namespace,
			},
		}); err != nil {
			return "", errors.WrapIf(err, "failed to create release namespace")
		}
	}

	releasePtr, err := installAction.Run(chartRequested, releaseInput.Values)
	if err != nil {
		return "", errors.WrapIf(err, "failed to install chart")
	}

	return releasePtr.Name, nil
}

func (r releaser) Uninstall(ctx context.Context, helmEnv helm.HelmEnv, kubeConfig helm.KubeConfigBytes, releaseName string, options helm.Options) error {
	// customize the settings passed forward
	envSettings := r.processEnvSettings(helmEnv)

	// component processing the kubeconfig
	restClientGetter := NewCustomGetter(envSettings.RESTClientGetter(), kubeConfig, r.logger)

	ns := "default"
	if options.Namespace != "" {
		ns = options.Namespace
	}
	actionConfig, err := r.getActionConfiguration(restClientGetter, ns)
	if err != nil {
		return errors.WrapIf(err, "failed to get action configuration")
	}

	uninstallAction := action.NewUninstall(actionConfig)
	uninstallAction.Timeout = time.Minute * 5

	res, err := uninstallAction.Run(releaseName)
	if err != nil {
		return err
	}
	if res != nil && res.Info != "" {
		r.logger.Debug(res.Info)
	}

	r.logger.Info("release successfully uninstalled", map[string]interface{}{"releaseName": releaseName})

	return nil
}

func (r releaser) List(_ context.Context, helmEnv helm.HelmEnv, kubeConfig helm.KubeConfigBytes, options helm.Options) ([]helm.Release, error) {
	// customize the settings passed forward
	envSettings := r.processEnvSettings(helmEnv)

	// component processing the kubeconfig
	restClientGetter := NewCustomGetter(envSettings.RESTClientGetter(), kubeConfig, r.logger)

	actionConfig, err := r.getActionConfiguration(restClientGetter, options.Namespace)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get action configuration")
	}

	listAction := action.NewList(actionConfig)
	listAction.SetStateMask()

	results, err := listAction.Run()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to list releases")
	}

	releases := make([]helm.Release, 0, len(results))
	for _, result := range results {
		releases = append(releases, helm.Release{
			ReleaseName: result.Name,
			ChartName:   result.Chart.Name(),
			Namespace:   result.Namespace,
			Values:      result.Chart.Values,
			Version:     result.Chart.Metadata.Version,
			ReleaseInfo: helm.ReleaseInfo{
				FirstDeployed: result.Info.FirstDeployed.Time,
				LastDeployed:  result.Info.LastDeployed.Time,
				Deleted:       result.Info.Deleted.Time,
				Description:   result.Info.Description,
				Status:        result.Info.Status.String(),
				Notes:         result.Info.Notes,
			},
		})
	}

	return releases, nil
}

func (r releaser) Get(_ context.Context, helmEnv helm.HelmEnv, kubeConfig helm.KubeConfigBytes, releaseInput helm.Release, options helm.Options) (helm.Release, error) {
	// customize the settings passed forward
	envSettings := r.processEnvSettings(helmEnv)

	// component processing the kubeconfig
	restClientGetter := NewCustomGetter(envSettings.RESTClientGetter(), kubeConfig, r.logger)

	actionConfig, err := r.getActionConfiguration(restClientGetter, options.Namespace)
	if err != nil {
		return helm.Release{}, errors.WrapIf(err, "failed to get action configuration")
	}

	getAction := action.NewGet(actionConfig)

	rawRelease, err := getAction.Run(releaseInput.ReleaseName)
	if err != nil {
		return helm.Release{}, errors.WrapIf(err, "failed to get release")
	}

	return helm.Release{
		ReleaseName: rawRelease.Name,
		ChartName:   rawRelease.Chart.Metadata.Name,
		Namespace:   rawRelease.Namespace,
		Values:      rawRelease.Chart.Values,
		Version:     rawRelease.Chart.Metadata.Version,
		ReleaseInfo: helm.ReleaseInfo{
			FirstDeployed: rawRelease.Info.FirstDeployed.Time,
			LastDeployed:  rawRelease.Info.LastDeployed.Time,
			Deleted:       rawRelease.Info.Deleted.Time,
			Description:   rawRelease.Info.Description,
			Status:        rawRelease.Info.Status.String(),
			Notes:         rawRelease.Info.Notes,
		},
	}, nil
}

func (r releaser) Upgrade(ctx context.Context, helmEnv helm.HelmEnv, kubeConfig helm.KubeConfigBytes, releaseInput helm.Release, options helm.Options) (string, error) {
	// customize the settings passed forward
	envSettings := r.processEnvSettings(helmEnv)

	// component processing the kubeconfig
	restClientGetter := NewCustomGetter(envSettings.RESTClientGetter(), kubeConfig, r.logger)

	ns := "default"
	if releaseInput.Namespace != "" {
		ns = releaseInput.Namespace
	}

	actionConfig, err := r.getActionConfiguration(restClientGetter, ns)
	if err != nil {
		return "", errors.WrapIf(err, "failed to get  action configuration")
	}

	upgradeAction := action.NewUpgrade(actionConfig)
	upgradeAction.Namespace = options.Namespace
	upgradeAction.Install = options.Install
	upgradeAction.Wait = options.Wait
	upgradeAction.Timeout = time.Minute * 5

	if upgradeAction.Version == "" && upgradeAction.Devel {
		r.logger.Debug("setting version to >0.0.0-0")
		upgradeAction.Version = ">0.0.0-0"
	}

	chartPath, err := upgradeAction.ChartPathOptions.LocateChart(releaseInput.ChartName, envSettings)
	if err != nil {
		return "", errors.WrapIf(err, "failed to locate chart")
	}

	if upgradeAction.Install {
		// If a release does not exist, install it.
		histClient := action.NewHistory(actionConfig)
		histClient.Max = 1
		if _, err := histClient.Run(releaseInput.ReleaseName); err == driver.ErrReleaseNotFound {
			r.logger.Debug("release doesn't exist, installing it now", map[string]interface{}{"releaseName": releaseInput.ReleaseName})

			rel, err := r.Install(ctx, helmEnv, kubeConfig, releaseInput, options)
			if err != nil {
				return "", errors.WrapIf(err, "failed to install release during upgrade")
			}

			return rel, nil
		} else if err != nil {
			return "", errors.WrapIf(err, "failed to install release during upgrade")
		}
	}

	// Check chart dependencies to make sure all are present in /charts
	ch, err := loader.Load(chartPath)
	if err != nil {
		return "", errors.WrapIf(err, "failed to load chart")
	}
	if req := ch.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(ch, req); err != nil {
			return "", errors.WrapIf(err, "failed to check dependencies")
		}
	}

	if ch.Metadata.Deprecated {
		r.logger.Warn("This chart is deprecated", map[string]interface{}{"chart": ch.Name()})
	}

	rel, err := upgradeAction.Run(releaseInput.ReleaseName, ch, releaseInput.Values)
	if err != nil {
		return "", errors.Wrap(err, "UPGRADE FAILED")
	}

	r.logger.Info("release has been upgraded. Happy Helming!", map[string]interface{}{"releaseName": releaseInput.ReleaseName})

	return rel.Name, nil
}

func (r releaser) Resources(_ context.Context, helmEnv helm.HelmEnv, kubeConfig helm.KubeConfigBytes, releaseInput helm.Release, options helm.Options) ([]helm.ReleaseResource, error) {
	// customize the settings passed forward
	envSettings := r.processEnvSettings(helmEnv)

	// component processing the kubeconfig
	restClientGetter := NewCustomGetter(envSettings.RESTClientGetter(), kubeConfig, r.logger)

	actionConfig, err := r.getActionConfiguration(restClientGetter, options.Namespace)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get action configuration")
	}

	getAction := action.NewGet(actionConfig)

	rawRelease, err := getAction.Run(releaseInput.ReleaseName)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get release")
	}

	resources, err := r.resourcesFromManifest(rawRelease.Manifest)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get release resources")
	}

	return resources, nil
}

// resourcesFromManifest digs out the resources from a release manifest
func (r releaser) resourcesFromManifest(manifest string) ([]helm.ReleaseResource, error) {
	var (
		rawManifest map[string]interface{}
		resources   []helm.ReleaseResource
		metadata    mapstructure.Metadata
	)

	decoder := yaml.NewDecoder(strings.NewReader(manifest))
	// iterate over yaml fragments in the manifest
	for decoder.Decode(&rawManifest) == nil {
		// helper struct to get the required information from the map
		var resource struct {
			Kind     string
			Metadata struct {
				Name string
			} `mapstructure: ",squash"` //do not change the tag!
		}

		// yaml fragment map into helper struct / metadata used to track conversion details - not yet used
		if err := mapstructure.DecodeMetadata(rawManifest, &resource, &metadata); err != nil {
			return nil, errors.WrapIf(err, "failed to process release manifest")
		}

		resources = append(resources, helm.ReleaseResource{
			Name: resource.Metadata.Name,
			Kind: resource.Kind,
		})
	}

	return resources, nil
}

// processEnvSettings emulates an cli.EnvSettings instance based on the passed in data
func (r releaser) processEnvSettings(helmEnv helm.HelmEnv) *cli.EnvSettings {
	envSettings := cli.New()
	envSettings.RepositoryConfig = helmEnv.GetHome()
	envSettings.RepositoryCache = helmEnv.GetRepoCache()

	return envSettings
}

// processEnvSettings emulates an cli.EnvSettings instance based on the passed in data
func (r releaser) processValues(providers getter.Providers, releaseInput helm.Release) (map[string]interface{}, error) {
	valueOpts := &values.Options{}

	for key, val := range releaseInput.Values {
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("%s=%s", key, val))
	}

	return valueOpts.MergeValues(providers)
}

func (r releaser) debugFnf(format string, v ...interface{}) {
	r.logger.Debug(fmt.Sprintf(format, v...))
}

func (r releaser) getActionConfiguration(clientGetter genericclioptions.RESTClientGetter, namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(clientGetter, namespace, "", r.debugFnf); err != nil {
		r.logger.Error("failed to initialize action config")
		return nil, errors.WrapIf(err, "failed to initialize  action config")
	}

	return actionConfig, nil
}

// isChartInstallable validates if a chart can be installed
//
// Application chart type is only installable
func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

type customGetter struct {
	delegate        genericclioptions.RESTClientGetter
	kubeConfigBytes []byte
	logger          Logger
}

func NewCustomGetter(delegate genericclioptions.RESTClientGetter, kubeconfig []byte, logger Logger) genericclioptions.RESTClientGetter {
	return customGetter{
		delegate:        delegate,
		kubeConfigBytes: kubeconfig,
		logger:          logger,
	}
}

func (c customGetter) ToRESTConfig() (*rest.Config, error) {
	return k8sclient.NewClientConfig(c.kubeConfigBytes)
}
func (c customGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return c.delegate.ToDiscoveryClient()
}

func (c customGetter) ToRESTMapper() (meta.RESTMapper, error) {
	return c.delegate.ToRESTMapper()
}

func (c customGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return c.delegate.ToRawKubeConfigLoader()
}
