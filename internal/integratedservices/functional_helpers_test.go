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

package integratedservices_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	pkgCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	cluster2 "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/kubernetes"
	"github.com/banzaicloud/pipeline/pkg/hook"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

type importedCluster struct {
	*cluster.KubeCluster
}

func (i importedCluster) GetClusterUID(_ context.Context, _ uint) (string, error) {
	return i.GetUID(), nil
}

func (i importedCluster) GetClusterOrgID(_ context.Context, _ uint) (uint, error) {
	return i.GetOrganizationId(), nil
}

func (i importedCluster) KubeConfig(_ context.Context, _ uint) ([]byte, error) {
	return i.GetK8sConfig()
}

func importCluster(kubeconfigContent, name string, orgID, userID uint) (*cluster.KubeCluster, error) {
	createSecretRequest := secret.CreateSecretRequest{
		Name: fmt.Sprintf("%s-kubeconfig", name),
		Type: secrettype.Kubernetes,
		Values: map[string]string{
			secrettype.K8SConfig: kubeconfigContent,
		},
		Tags: []string{
			secret.TagKubeConfig,
			secret.TagBanzaiReadonly,
		},
	}

	secretID, err := secret.Store.Store(orgID, &createSecretRequest)
	if err != nil {
		return nil, err
	}

	cluster, err := cluster.CreateKubernetesClusterFromRequest(&cluster2.CreateClusterRequest{
		Name:     name,
		Cloud:    "kubernetes",
		SecretId: secretID,
		Properties: &cluster2.CreateClusterProperties{
			CreateClusterKubernetes: &kubernetes.CreateClusterKubernetes{
				Metadata: map[string]string{},
			},
		},
	}, orgID, userID)
	if err != nil {
		return nil, err
	}

	err = cluster.SetStatus(pkgCluster.Running, "just fine")
	if err != nil {
		return nil, err
	}

	err = cluster.CreateCluster()
	if err != nil {
		return nil, err
	}

	err = cluster.Persist()
	if err != nil {
		return nil, err
	}

	return cluster, err
}

func loadConfig() *cmd.Config {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("::"),
	)
	v.SetConfigFile(filepath.Join(os.Getenv("PIPELINE_CONFIG_DIR"), "config.yaml"))

	// Load common configuration
	cmd.Configure(v, pflag.NewFlagSet("pipeline-test", pflag.ExitOnError))

	err := v.ReadInConfig()
	emperror.Panic(errors.WithMessage(err, "failed to read configuration"))

	var config cmd.Config
	err = v.Unmarshal(&config, hook.DecodeHookWithDefaults())
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	err = config.Process()
	emperror.Panic(errors.WithMessage(err, "failed to process configuration"))

	return &config
}
