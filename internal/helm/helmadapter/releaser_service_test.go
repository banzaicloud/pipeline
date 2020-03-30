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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"

	"github.com/banzaicloud/pipeline/internal/common"
)

func Test_releaser_Install(t *testing.T) {

	actionConfig := new(action.Configuration)

	envSettings := cli.New()
	//envSettings.RepositoryConfig = helm.NewHelmEnv("/Users/puski/Library/Preferences/helm/repositories.yaml").GetHome()

	valueOpts := &values.Options{}

	restGetter := NewCustomGetter(envSettings.RESTClientGetter(), getConfig(), common.NoopLogger{})

	if err := actionConfig.Init(restGetter, envSettings.Namespace(), "", debug); err != nil {
		t.Fatal(err)
	}

	installAction := action.NewInstall(actionConfig)
	installAction.GenerateName = true

	args := []string{"stable/mysql"} // TODO
	name, chart, err := installAction.NameAndChart(args)
	if err != nil {
		t.Fatal(err)
	}
	installAction.ReleaseName = name

	cp, err := installAction.ChartPathOptions.LocateChart(chart, envSettings)
	if err != nil {
		t.Fatal(err)
	}

	p := getter.All(envSettings)
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		t.Fatal(err)
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		t.Fatal(err)
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		t.Fatal(err)
	}

	if chartRequested.Metadata.Deprecated {
		t.Log("deprecated")
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
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		}
	}

	installAction.Namespace = envSettings.Namespace()
	releasePtr, err := installAction.Run(chartRequested, vals)
	if err != nil {
		t.Fatal(err)
	}

	print(releasePtr.Name)

	return
}

func getConfig() []byte {
	dat, err := ioutil.ReadFile("/Users/puski/go/src/github.com/banzaicloud/pipeline/internal/helm/helmadapter/kubeconfig_test.yaml")
	if err != nil {
		panic(err)
	}

	return dat
}

func debug(format string, v ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	log.Output(2, fmt.Sprintf(format, v...))
}
