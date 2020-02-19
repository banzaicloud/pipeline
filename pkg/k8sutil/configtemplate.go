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

package k8sutil

import (
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type AuthInfoFactory interface {
	CreateAuthInfo(string) *clientcmdapi.AuthInfo
}

// CreateAuthInfoFunc creates a provider specific AuthInfo object for the given cluster
type CreateAuthInfoFunc func(clusterName string) *clientcmdapi.AuthInfo

// CreateAuthInfo implements AuthInfoFactory for CreateAuthInfoFunc
func (f CreateAuthInfoFunc) CreateAuthInfo(clusterName string) *clientcmdapi.AuthInfo {
	return f(clusterName)
}

type ConfigBase struct {
	ClusterName              string
	APIEndpoint              string
	CertificateAuthorityData []byte
}

func ExtractConfigBase(config *clientcmdapi.Config) *ConfigBase {
	configBase := &ConfigBase{}

	currentContextName := config.CurrentContext
	if currentContext, ok := config.Contexts[currentContextName]; ok {
		if server, ok := config.Clusters[currentContext.Cluster]; ok {
			configBase.APIEndpoint = server.Server
			configBase.CertificateAuthorityData = server.CertificateAuthorityData
			configBase.ClusterName = currentContext.Cluster
		}
	}

	return configBase
}

// CreateConfigFromTemplate creates a minimal Kubernetes Config based on the given information
func (configBase *ConfigBase) CreateConfigFromTemplate(authInfo AuthInfoFactory) *clientcmdapi.Config {
	return &clientcmdapi.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: configBase.ClusterName,
		Clusters: map[string]*clientcmdapi.Cluster{
			configBase.ClusterName: {
				Server:                   configBase.APIEndpoint,
				CertificateAuthorityData: configBase.CertificateAuthorityData,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			configBase.ClusterName: {
				AuthInfo: configBase.ClusterName,
				Cluster:  configBase.ClusterName,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			configBase.ClusterName: authInfo.CreateAuthInfo(configBase.ClusterName),
		},
	}
}
